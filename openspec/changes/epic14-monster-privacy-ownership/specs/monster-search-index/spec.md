## MODIFIED Requirements

### Requirement: Typesense monster collection schema
The system SHALL initialize a Typesense collection named `monsters` with fields: `id` (string), `name` (string, facet), `max_hp` (int32), `initiative_modifier` (int32, optional), `edition` (string, facet), `is_custom` (bool), `private` (bool, optional), `owner_id` (string, optional), `owner_display_name` (string, optional). The collection SHALL be created at server startup if it does not already exist.

#### Scenario: Collection created on first startup
- **WHEN** the server starts and no `monsters` collection exists in Typesense
- **THEN** the server SHALL create it with the specified schema, including the `is_custom`, `private`, `owner_id`, and `owner_display_name` fields

#### Scenario: Collection already exists
- **WHEN** the server starts and a `monsters` collection already exists in Typesense
- **THEN** the server SHALL NOT attempt to recreate it

### Requirement: MongoDB writes are mirrored to Typesense on a best-effort basis
After a successful MongoDB write to either the `monsters` collection (official) or the `custom_monsters` collection (DM-authored), the system SHALL upsert the same document (mapped to the Typesense schema fields, including `is_custom`, and — for custom monsters — `private`, `owner_id`, `owner_display_name`) into the single Typesense `monsters` collection, keyed by the document's MongoDB `id`. A failure to write to Typesense SHALL be logged and SHALL NOT cause the originating MongoDB write or request to be reported as failed.

#### Scenario: Official monster write is mirrored
- **WHEN** a MongoDB write to the `monsters` collection succeeds and the subsequent Typesense upsert also succeeds
- **THEN** the document SHALL be queryable via Typesense search immediately afterward with `is_custom: false`

#### Scenario: Custom monster write is mirrored
- **WHEN** a MongoDB write to the `custom_monsters` collection succeeds and the subsequent Typesense upsert also succeeds
- **THEN** the document SHALL be queryable via Typesense search immediately afterward with `is_custom: true`, `private`, `owner_id`, and `owner_display_name` populated

#### Scenario: Typesense write fails
- **WHEN** a MongoDB upsert succeeds but the subsequent Typesense upsert fails (e.g., Typesense unreachable)
- **THEN** the system SHALL log the failure, the MongoDB document SHALL remain saved, and the caller SHALL NOT receive an error solely because of the Typesense failure

## ADDED Requirements

### Requirement: Deleting a custom monster removes its Typesense mirror
When a `custom_monsters` document is deleted from MongoDB, the system SHALL best-effort delete the corresponding document from the Typesense `monsters` collection by its id. A failure to remove the Typesense document SHALL be logged and SHALL NOT cause the delete request to be reported as failed.

#### Scenario: Deleted custom monster no longer appears in search
- **WHEN** a DM deletes their own custom monster and the Typesense removal succeeds
- **THEN** subsequent searches SHALL NOT return that document

#### Scenario: Typesense removal fails during delete
- **WHEN** the MongoDB deletion succeeds but the Typesense removal fails
- **THEN** the system SHALL log the failure and the delete request SHALL still be reported as successful to the caller
