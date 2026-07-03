# Spec: Item Search

## Purpose

Defines the item catalog (Typesense-backed search over a MongoDB-stored 5etools item dataset) and the endpoints used to search it and look up a single item by id, mirroring the existing monster-search architecture.

## Requirements

### Requirement: Typesense item collection schema
The system SHALL initialize a Typesense collection named `items` with fields: `id` (string), `name` (string, facet), `edition` (string, facet), `category` (string, facet), `rarity` (string, facet, optional). The collection SHALL be created at server startup if it does not already exist. Unlike the `monsters` collection, no `is_custom`, `private`, `owner_id`, or `owner_display_name` fields are included, since the item catalog has no custom-item or ownership concept.

#### Scenario: Collection created on first startup
- **WHEN** the server starts and no `items` collection exists in Typesense
- **THEN** the server SHALL create it with the specified schema

#### Scenario: Collection already exists
- **WHEN** the server starts and an `items` collection already exists in Typesense
- **THEN** the server SHALL NOT attempt to recreate it

### Requirement: Item catalog is stored in MongoDB with a stable id
The system SHALL store item catalog documents in a MongoDB collection, each with a generated `id`, `name`, `edition`, `category` (human-readable, not a raw 5etools type code), `rarity` (optional), `reference_url`, `five_etools_id`, and `source_book`. Upserts SHALL be keyed by `{name, edition}` and SHALL use `FindOneAndReplace` with upsert enabled and `ReturnDocument: After`, so the resulting document's `id` is always available regardless of whether the operation inserted or replaced.

#### Scenario: Upsert of a new item
- **WHEN** an item is upserted for a `{name, edition}` pair with no existing document
- **THEN** the operation SHALL insert a new document and return it with a populated `id`

#### Scenario: Upsert replacing an existing item
- **WHEN** an item is upserted for a `{name, edition}` pair with an existing document
- **THEN** the operation SHALL replace the document in place and return it with its existing (unchanged) `id`

### Requirement: MongoDB writes are mirrored to Typesense on a best-effort basis
After a successful MongoDB write to the item catalog collection, the system SHALL upsert the corresponding document (mapped to the Typesense schema fields) into the `items` Typesense collection, keyed by the document's MongoDB `id`. A failure to write to Typesense SHALL be logged and SHALL NOT cause the originating MongoDB write to be reported as failed.

#### Scenario: Item write is mirrored
- **WHEN** a MongoDB write to the item catalog collection succeeds and the subsequent Typesense upsert also succeeds
- **THEN** the document SHALL be queryable via Typesense search immediately afterward

#### Scenario: Typesense write fails
- **WHEN** a MongoDB upsert succeeds but the subsequent Typesense upsert fails
- **THEN** the system SHALL log the failure, the MongoDB document SHALL remain saved, and the caller SHALL NOT receive an error solely because of the Typesense failure

### Requirement: Typesense unavailability does not block server startup
A failure to connect to or initialize Typesense at server startup SHALL be logged but SHALL NOT prevent the server from starting and serving non-search functionality. This applies jointly with the existing monster-search Typesense client — items do not introduce a second Typesense connection.

#### Scenario: Typesense unreachable at startup
- **WHEN** the server starts and Typesense cannot be reached
- **THEN** the server SHALL log the failure and continue starting normally, and item search SHALL behave as specified in the "Typesense unavailable at query time" scenario below

### Requirement: A scrubber path populates the item catalog from 5etools source data
The system SHALL provide a way to parse 5etools' `items-base.json` (`baseitem` array — generic equipment categories) and `items.json` (`item` array — named and magic items) from a local source checkout, resolve `_copy` inheritance where present, translate raw `type` codes into human-readable `category` values, build a `reference_url` pointing to the item's 5etools page, and upsert each into the item catalog via the shared upsert path defined above.

#### Scenario: Base equipment category is scrubbed
- **WHEN** the scrubber processes `items-base.json`'s `baseitem` entry for "Longsword"
- **THEN** the item catalog SHALL contain a document for "Longsword" with a human-readable `category` (not a raw type code) and a `reference_url`

#### Scenario: Named/magic item is scrubbed
- **WHEN** the scrubber processes `items.json`'s `item` entry for "Rope of Climbing"
- **THEN** the item catalog SHALL contain a document for "Rope of Climbing" with its `rarity` and `reference_url` populated

#### Scenario: Copy-inherited item is resolved
- **WHEN** the scrubber processes an `item` entry with a `_copy` reference to another entry
- **THEN** the resulting catalog document SHALL reflect the merged/inherited fields, not a partial or unresolved entry

#### Scenario: Re-running the scrubber fully backfills the search index
- **WHEN** the scrubber is re-run against the same source checkout
- **THEN** every successfully processed item SHALL be upserted into both MongoDB and Typesense, without requiring a dedicated backfill tool

### Requirement: Backend exposes an edition-filtered item search endpoint
The system SHALL expose `GET /api/search/items?q=<query>&edition=<edition>`. The request SHALL require authentication. Both query parameters are required. The endpoint SHALL return HTTP 400 if either is missing or if `edition` is not `"5e"` or `"5.5e"`, and HTTP 401 if the requester is not authenticated. The endpoint SHALL query the Typesense `items` collection with typo tolerance and prefix matching on `name`, filtered to `edition:=<edition>`. Matching hits SHALL be returned as a JSON array of `{ id, name, category, rarity }` objects.

#### Scenario: Typo-tolerant match
- **WHEN** an authenticated user sends `GET /api/search/items?q=Longswrd&edition=5e`
- **THEN** the server SHALL return results including "Longsword" despite the misspelling

#### Scenario: Prefix match
- **WHEN** an authenticated user sends `GET /api/search/items?q=Rope&edition=5e`
- **THEN** the server SHALL return results including items whose names start with "Rope"

#### Scenario: Results are filtered to the requested edition
- **WHEN** a document `{ name: "Potion of Healing", edition: "5e" }` exists and an authenticated user sends `GET /api/search/items?q=Potion&edition=5.5e`
- **THEN** the 5e document SHALL NOT appear in the results

#### Scenario: Unauthenticated search is rejected
- **WHEN** a request to `GET /api/search/items` is made with no valid session
- **THEN** the server SHALL return HTTP 401

#### Scenario: No matches
- **WHEN** an authenticated user sends `GET /api/search/items?q=Xyzzy&edition=5e` and nothing matches
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]`

#### Scenario: Missing q parameter
- **WHEN** an authenticated user sends `GET /api/search/items?edition=5e` with no `q`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Invalid edition parameter
- **WHEN** an authenticated user sends `GET /api/search/items?q=Rope&edition=4e`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Typesense unavailable at query time
- **WHEN** an authenticated user sends a valid `GET /api/search/items` request and Typesense cannot be reached
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]` rather than an error

### Requirement: Backend exposes an item lookup endpoint for selection follow-up
The system SHALL expose `GET /api/items/{id}`, requiring authentication, returning the full catalog document (including `reference_url`) for the given id, or HTTP 404 if no item catalog document has that id.

#### Scenario: Item found by id
- **WHEN** an authenticated user sends `GET /api/items/{id}` for an existing item catalog document
- **THEN** the server SHALL return HTTP 200 with the full document, including `reference_url`

#### Scenario: Item not found
- **WHEN** an authenticated user sends `GET /api/items/{id}` for an id with no matching document
- **THEN** the server SHALL respond with HTTP 404
