## MODIFIED Requirements

### Requirement: Scrubber upserts monsters idempotently on { name, edition }
Each monster SHALL be upserted into the `monsters` collection using `{ name, edition }` as the composite filter key, via the same shared upsert path used by the live API's manual monster-creation endpoint. Re-running the scrubber against the same source SHALL update existing documents in place and SHALL NOT create duplicate entries. Each successful MongoDB upsert SHALL also be mirrored into the Typesense search index on a best-effort basis (a Typesense failure SHALL be logged but SHALL NOT abort processing of the entry or subsequent entries). If an existing document at `{ name, edition }` has `is_custom: true`, the scrubber's write (always `is_custom: false`) SHALL be skipped for that entry, preserving the DM-customized document unchanged.

#### Scenario: Monster already exists for the same edition
- **WHEN** a document with `{ name: "Goblin", edition: "5e" }` already exists with `is_custom: false` and the scrubber processes a Goblin entry with `--edition 5e`
- **THEN** the existing document SHALL be updated with the latest field values, mirrored into Typesense, and no duplicate SHALL be created

#### Scenario: Same monster name exists for a different edition
- **WHEN** a document with `{ name: "Goblin", edition: "5e" }` exists and the scrubber processes a Goblin entry with `--edition 5.5e`
- **THEN** a new document `{ name: "Goblin", edition: "5.5e" }` SHALL be inserted alongside the existing one and mirrored into Typesense

#### Scenario: Scrubber would overwrite a DM-customized monster
- **WHEN** a document with `{ name: "Goblin", edition: "5e", is_custom: true }` exists (created via the manual monster-creation endpoint) and the scrubber processes a Goblin entry with `--edition 5e`
- **THEN** the scrubber SHALL skip writing that entry, the existing custom document SHALL remain unchanged in both MongoDB and Typesense, and the tool SHALL continue processing remaining entries

#### Scenario: Typesense unavailable during a scrubber run
- **WHEN** the scrubber processes an entry, the MongoDB upsert succeeds, but the Typesense mirror fails
- **THEN** the scrubber SHALL log the failure, count the entry as processed in its MongoDB-write tally, and continue to the next entry without aborting the run

#### Scenario: Re-running the scrubber backfills Typesense for pre-existing data
- **WHEN** the scrubber is re-run with the same `--source` and `--edition` as a prior import that predates Typesense integration
- **THEN** every successfully processed, non-custom-protected entry SHALL be upserted into both MongoDB and Typesense
