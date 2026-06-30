# Spec: Monster Scrubber

## Purpose

Defines the CLI tool that reads a locally cloned 5etools bestiary repository and bulk-upserts normalized monster documents into MongoDB. The tool is the primary mechanism for populating the monster collection with compendium data for a given edition.

## Requirements

### Requirement: CLI accepts required flags for source path and edition
The scrubber SHALL be invocable via `go run ./cmd/scrubber --source <path> --edition <edition>`. Both flags are required. `--source` MUST be a path to the root of a local 5etools repository. `--edition` MUST be one of `"5e"` or `"5.5e"`. The tool SHALL exit with a non-zero status and a descriptive error message if either flag is missing or `--edition` holds an unrecognised value.

#### Scenario: Both flags provided and valid
- **WHEN** the scrubber is invoked with `--source ./data/5etools --edition 5e`
- **THEN** the tool SHALL proceed to read bestiary files from `./data/5etools/data/bestiary/`

#### Scenario: Missing --edition flag
- **WHEN** the scrubber is invoked without `--edition`
- **THEN** the tool SHALL print an error message and exit with a non-zero status code

#### Scenario: Invalid --edition value
- **WHEN** the scrubber is invoked with `--edition 4e`
- **THEN** the tool SHALL print an error message indicating the allowed values and exit with a non-zero status code

---

### Requirement: Scrubber reads all bestiary JSON files from the source directory
The tool SHALL read every file matching the pattern `bestiary-*.json` under `<source>/data/bestiary/`. Files that cannot be parsed as valid JSON SHALL be skipped with a warning log; the tool SHALL continue processing remaining files.

#### Scenario: Multiple bestiary files present
- **WHEN** the source directory contains `bestiary-mm.json`, `bestiary-vgm.json`, and `bestiary-mtf.json`
- **THEN** the tool SHALL process monster entries from all three files in a single run

#### Scenario: A bestiary file contains malformed JSON
- **WHEN** one file in the bestiary directory is not valid JSON
- **THEN** the tool SHALL log a warning identifying the file and continue processing the remaining files without exiting

#### Scenario: No bestiary files found
- **WHEN** the source directory contains no files matching `bestiary-*.json`
- **THEN** the tool SHALL log a warning and exit with a non-zero status code

---

### Requirement: Scrubber normalizes 5etools fields into the monster schema
For each monster entry, the tool SHALL map 5etools fields to the MongoDB schema as follows:

| 5etools field | MongoDB field | Transformation |
|---|---|---|
| `name` | `name` | No transformation |
| `hp.average` | `max_hp` | Integer, direct copy |
| `dex` | `initiative_modifier` | `floor((dex - 10) / 2)`; default `0` if field absent |
| `source` | `source_book` | No transformation |
| `source` + `name` | `five_etools_id` | `{name-kebab}-{source-lower}` (see URL requirement) |
| `--edition` flag | `edition` | Set from CLI flag |
| — | `is_custom` | Always `false` |
| — | `source_type` | Always `"url"` |

For entries with a `_copy` field and no own `hp`, the tool SHALL inherit `hp.average` and `dex` from the referenced base creature. `name`, `source_book`, `five_etools_id`, and `reference_url` SHALL always be derived from the entry itself. Entries whose base creature cannot be resolved SHALL be logged and skipped.

#### Scenario: Monster with standard dex score
- **WHEN** a 5etools entry has `"dex": 14`
- **THEN** the stored `initiative_modifier` SHALL be `2`

#### Scenario: Monster with odd dex score below 10
- **WHEN** a 5etools entry has `"dex": 9`
- **THEN** the stored `initiative_modifier` SHALL be `-1`

#### Scenario: Monster entry missing dex field
- **WHEN** a 5etools entry has no `dex` field (e.g., an object or trap statblock)
- **THEN** the tool SHALL store `initiative_modifier: 0` and log a warning for that entry

#### Scenario: Monster entry uses _copy with no own hp
- **WHEN** a 5etools entry has a `_copy` field pointing to a resolvable base creature and no own `hp`
- **THEN** the tool SHALL inherit `hp.average` and `dex` from the base creature and upsert the entry with its own `name` and `source`

#### Scenario: Monster entry uses _copy with unresolvable base
- **WHEN** a 5etools entry has a `_copy` field pointing to a creature not found in any bestiary file
- **THEN** the tool SHALL log a warning and skip the entry

---

### Requirement: Scrubber auto-generates reference_url per edition
The tool SHALL construct `reference_url` for each monster using the edition-specific base URL and the monster's name and source code.

URL construction rules:
- `name-kebab`: monster name lowercased with spaces replaced by hyphens
- `source-lower`: source code field lowercased
- `five_etools_id`: `{name-kebab}-{source-lower}`
- `"5e"` base: `https://2014.5e.tools/bestiary/`
- `"5.5e"` base: `https://5e.tools/bestiary/`
- Final URL: `{base}{five_etools_id}.html`

#### Scenario: 5e monster URL generation
- **WHEN** a 5etools entry has `"name": "Aarakocra"`, `"source": "MM"`, and `--edition 5e`
- **THEN** `reference_url` SHALL be `https://2014.5e.tools/bestiary/aarakocra-mm.html` and `five_etools_id` SHALL be `aarakocra-mm`

#### Scenario: 5.5e monster URL generation
- **WHEN** a 5etools entry has `"name": "Mercenary Envoy"`, `"source": "AITFRFCD"`, and `--edition 5.5e`
- **THEN** `reference_url` SHALL be `https://5e.tools/bestiary/mercenary-envoy-aitfrfcd.html` and `five_etools_id` SHALL be `mercenary-envoy-aitfrfcd`

---

### Requirement: Scrubber upserts monsters idempotently on { name, edition }
Each monster SHALL be upserted into the `monsters` collection using `{ name, edition }` as the composite filter key. Re-running the scrubber against the same source SHALL update existing documents in place and SHALL NOT create duplicate entries.

#### Scenario: Monster already exists for the same edition
- **WHEN** a document with `{ name: "Goblin", edition: "5e" }` already exists and the scrubber processes a Goblin entry with `--edition 5e`
- **THEN** the existing document SHALL be updated with the latest field values; no duplicate SHALL be created

#### Scenario: Same monster name exists for a different edition
- **WHEN** a document with `{ name: "Goblin", edition: "5e" }` exists and the scrubber processes a Goblin entry with `--edition 5.5e`
- **THEN** a new document `{ name: "Goblin", edition: "5.5e" }` SHALL be inserted alongside the existing one

---

### Requirement: Scrubber prints a completion summary
Upon finishing all files, the tool SHALL print to stdout the total number of documents processed, inserted (new), and updated (existing).

#### Scenario: Successful run
- **WHEN** the scrubber completes without fatal errors
- **THEN** the tool SHALL print a summary line such as `Done: 1234 processed, 1200 inserted, 34 updated` and exit with status 0
