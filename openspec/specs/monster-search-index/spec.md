# Spec: Monster Search Index

## Purpose

Defines the Typesense-backed search index that mirrors monster documents out of MongoDB, and the write-path guarantees (reliable ID resolution, best-effort mirroring, custom-monster protection, non-fatal startup) that keep the index consistent with the system of record without requiring a dedicated reindex tool.

## Requirements

### Requirement: Typesense monster collection schema
The system SHALL initialize a Typesense collection named `monsters` with fields: `id` (string), `name` (string, facet), `max_hp` (int32), `initiative_modifier` (int32), `edition` (string, facet). The collection SHALL be created at server startup if it does not already exist.

#### Scenario: Collection created on first startup
- **WHEN** the server starts and no `monsters` collection exists in Typesense
- **THEN** the server SHALL create it with the specified schema

#### Scenario: Collection already exists
- **WHEN** the server starts and a `monsters` collection already exists in Typesense
- **THEN** the server SHALL NOT attempt to recreate it

### Requirement: Monster upserts reliably resolve a MongoDB document ID
The system SHALL use `FindOneAndReplace` with upsert enabled and `ReturnDocument: After` (not `ReplaceOne`) when upserting a monster document keyed by `{name, edition}`, so the resulting document's `_id` is always available to the caller — whether the operation inserted a new document or replaced an existing one.

#### Scenario: Upsert of a new monster
- **WHEN** a monster is upserted for a `{name, edition}` pair with no existing document
- **THEN** the operation SHALL insert a new document and return it with a populated `id`

#### Scenario: Upsert replacing an existing monster
- **WHEN** a monster is upserted for a `{name, edition}` pair with an existing document
- **THEN** the operation SHALL replace the document in place and return it with its existing (unchanged) `id`

### Requirement: MongoDB writes are mirrored to Typesense on a best-effort basis
After a successful MongoDB upsert, the system SHALL upsert the same document (mapped to the Typesense schema fields) into the Typesense `monsters` collection, keyed by the document's MongoDB `id`. A failure to write to Typesense SHALL be logged and SHALL NOT cause the MongoDB write or the originating request/entry to be reported as failed.

#### Scenario: Typesense write succeeds
- **WHEN** a MongoDB upsert succeeds and the subsequent Typesense upsert also succeeds
- **THEN** the document SHALL be queryable via Typesense search immediately afterward

#### Scenario: Typesense write fails
- **WHEN** a MongoDB upsert succeeds but the subsequent Typesense upsert fails (e.g., Typesense unreachable)
- **THEN** the system SHALL log the failure, the MongoDB document SHALL remain saved, and the caller SHALL NOT receive an error solely because of the Typesense failure

### Requirement: Custom monsters are protected from non-custom overwrites
Before performing a MongoDB upsert, if a document already exists at `{name, edition}` with `is_custom: true`, and the incoming write has `is_custom: false`, the system SHALL skip the write (logging that it was skipped) and return the existing document unchanged. Writes with `is_custom: true` SHALL always proceed regardless of the existing document's state.

#### Scenario: Non-custom write attempts to overwrite a custom monster
- **WHEN** a write with `is_custom: false` targets a `{name, edition}` pair whose existing document has `is_custom: true`
- **THEN** the system SHALL skip the write and the existing custom document SHALL remain unchanged in both MongoDB and Typesense

#### Scenario: Custom write always proceeds
- **WHEN** a write with `is_custom: true` targets any `{name, edition}` pair, custom or not
- **THEN** the system SHALL perform the upsert normally

#### Scenario: Non-custom write with no conflicting custom document
- **WHEN** a write with `is_custom: false` targets a `{name, edition}` pair with no existing document, or an existing document with `is_custom: false`
- **THEN** the system SHALL perform the upsert normally

### Requirement: Re-running the scrubber fully backfills the search index
Because the scrubber's upserts flow through the same MongoDB-then-Typesense write path as any other monster write, re-running the scrubber against a previously-imported source SHALL result in every processed monster being present in Typesense, without requiring any dedicated backfill or reindexing tool.

#### Scenario: Backfilling pre-existing MongoDB data
- **WHEN** the scrubber is re-run with the same `--source` and `--edition` used for a prior import that predates Typesense integration
- **THEN** every successfully processed monster SHALL be upserted into Typesense as part of that run, in addition to MongoDB

### Requirement: Typesense unavailability does not block server startup
A failure to connect to or initialize Typesense at server startup SHALL be logged but SHALL NOT prevent the server from starting and serving non-search functionality.

#### Scenario: Typesense unreachable at startup
- **WHEN** the server starts and Typesense cannot be reached
- **THEN** the server SHALL log the failure and continue starting normally
