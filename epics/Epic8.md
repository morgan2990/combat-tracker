# Epic 8: Monster Data Scrubber Pipeline

## US8.1: MongoDB Schema Extension for Edition-Aware Monsters
**As a** Backend Developer,  
**I want to** extend the MongoDB monster document schema with edition, initiative modifier, and provenance fields,  
**So that** the database can hold both 5e and 5.5e versions of the same creature and distinguish scraped entries from DM-created ones.

### Acceptance Criteria:
- **AC 1:** The `monsters` collection schema must be extended with the following new fields:
    - `edition` (String, restricted to `"5e"` or `"5.5e"`)
    - `initiative_modifier` (Integer, derived from the creature's Dexterity score)
    - `is_custom` (Boolean — `false` for scrubber-imported monsters, `true` for DM-created monsters)
- **AC 2:** The existing unique index on `name` must be dropped and replaced with a compound unique index on `{ name, edition }`, so that the same creature name can coexist once per edition.
- **AC 3:** The `store/monster.go` service must be updated to include the new fields in all create and upsert operations. DM-created monsters (from US7.2) must set `is_custom = true` and must require an `edition` value.

---

## US8.2: CLI Scrubber Tool
**As a** Developer,  
**I want** a standalone CLI command that reads a local 5etools bestiary repository and bulk-upserts its monsters into MongoDB,  
**So that** I can populate the database with a full compendium of creatures for a given edition without manual data entry.

### Technical Note & Context:
5etools maintains separate repositories for the 5e (2014) and 5.5e (2024) rulesets. The scrubber is run once per edition, pointed at the locally cloned repository. The tool must be idempotent — re-running it refreshes existing records rather than duplicating them.

### CLI Flags:
- `--source` — path to the root of the local 5etools repository
- `--edition` — target edition for this run (`"5e"` or `"5.5e"`)

### Acceptance Criteria:
- **AC 1:** The scrubber must be implemented as a standalone Go program at `cmd/scrubber/main.go`, runnable via `go run ./cmd/scrubber --source <path> --edition <edition>`.
- **AC 2:** The scrubber must read all `bestiary-*.json` files found under `<source>/data/bestiary/` and process every monster entry within them.
- **AC 3:** For each monster entry, the scrubber must normalize the following 5etools fields into the MongoDB schema:
    - `hp.average` → `max_hp`
    - `floor((dex - 10) / 2)` → `initiative_modifier`
    - `name` → `name`
    - `source` → used for URL generation (see AC 4)
    - `edition` → set from the `--edition` flag
    - `is_custom` → set to `false`
    - `source_type` → set to `"URL"`
- **AC 4:** The scrubber must auto-generate a `reference_url` for each monster using the following rules based on the `--edition` flag:
    - `"5e"` → `https://2014.5e.tools/bestiary/{name-kebab}-{source-lower}.html`
    - `"5.5e"` → `https://5e.tools/bestiary/{name-kebab}-{source-lower}.html`
    - Where `name-kebab` is the monster name lowercased with spaces replaced by hyphens, and `source-lower` is the source code lowercased (e.g., `"MM"` → `"mm"`).
- **AC 5:** Each monster must be upserted into MongoDB using `{ name, edition }` as the composite key. Existing records with the same key must be updated in place; no duplicate documents may be created.
- **AC 6:** The scrubber must print a summary upon completion indicating the total number of documents processed, inserted, and updated.
