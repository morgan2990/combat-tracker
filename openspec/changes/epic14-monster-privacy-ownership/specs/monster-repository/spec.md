## MODIFIED Requirements

### Requirement: DM can register a URL-type monster template
The system SHALL expose `POST /api/monsters` accepting a JSON body with `name` (string), `max_hp` (integer), `edition` (string, required, one of `"5e"` or `"5.5e"`), `private` (boolean, optional, defaults to `false`), and optional fields: `source_type` ("url" | "pdf"), `reference_url` (string), `five_etools_id` (string), `source_book` (string). The request SHALL require authentication. The server SHALL insert a new document into the `custom_monsters` collection (never the official `monsters` collection) with `is_custom: true`, `owner_id` set to the authenticated requester's user id, and `owner_display_name` set to the requester's display name. Each submission SHALL insert a fresh document with its own MongoDB `id` — there is no upsert-by-name deduplication; re-submitting the same name creates a second, independent custom monster. The response SHALL include the document's `id`. The insert SHALL also be mirrored into the Typesense search index on a best-effort basis.

#### Scenario: DM registers a new URL-type monster
- **WHEN** an authenticated DM sends `POST /api/monsters` with JSON `{ "name": "Goblin", "max_hp": 7, "edition": "5e", "source_type": "url", "reference_url": "https://2014.5e.tools/bestiary/goblin-mm.html" }`
- **THEN** the server SHALL insert a document into `custom_monsters` with `is_custom: true`, `owner_id` set from the session, `private: false` (default), return HTTP 200 with the document's `id`, and best-effort mirror the document into Typesense

#### Scenario: DM registers a monster with no statblock reference
- **WHEN** an authenticated DM sends `POST /api/monsters` with only `name`, `max_hp`, and `edition`
- **THEN** the server SHALL insert the document into `custom_monsters` with `source_type` absent and return HTTP 200 with an `id`

#### Scenario: DM marks a monster private at creation
- **WHEN** an authenticated DM sends `POST /api/monsters` with `"private": true`
- **THEN** the server SHALL store `private: true` on the created document

#### Scenario: Two DMs register monsters with the same name and edition
- **WHEN** DM Alice and DM Bob each send `POST /api/monsters` with `{ "name": "Goblin", "edition": "5e", "max_hp": 7 }`
- **THEN** the server SHALL insert two independent documents into `custom_monsters`, each with its own `id` and `owner_id`, and neither write SHALL affect the other's document

#### Scenario: Unauthenticated request is rejected
- **WHEN** a request to `POST /api/monsters` is made with no valid session
- **THEN** the server SHALL return HTTP 401

#### Scenario: Edition field missing from request
- **WHEN** an authenticated DM sends `POST /api/monsters` without an `edition` field
- **THEN** the server SHALL return HTTP 400

### Requirement: DM can register a PDF-type monster template
The system SHALL expose `POST /api/monsters` accepting `multipart/form-data` with fields `name`, `max_hp`, `edition` (required), `source_type: "pdf"`, `private` (optional, `"true"`/`"false"`, defaults to false), and a file part named `pdf`. The request SHALL require authentication. Since custom monster names are not unique across owners, the PDF SHALL be uploaded to MinIO under a key derived from the document's id (`custom-monsters/{id}.pdf`), not its name; `pdf_object_key` SHALL be set on the stored document accordingly. The server SHALL insert a new document into `custom_monsters` with `is_custom: true`, `owner_id`, and `owner_display_name` set from the authenticated requester. The response SHALL include the document's `id`, and the insert SHALL also be mirrored into the Typesense search index on a best-effort basis.

#### Scenario: DM registers a PDF-type monster
- **WHEN** an authenticated DM sends `POST /api/monsters` as multipart with a valid PDF file, `source_type: "pdf"`, and a valid `edition`
- **THEN** the server SHALL upload the file to MinIO under `custom-monsters/{id}.pdf`, insert a `custom_monsters` document with the matching `pdf_object_key`, `is_custom: true`, `owner_id`, and an `id`, then return HTTP 200

#### Scenario: Two DMs upload PDFs for same-named monsters without collision
- **WHEN** DM Alice and DM Bob each register a PDF-type monster named "Goblin" for the same edition
- **THEN** each upload SHALL be stored under its own document's id-keyed MinIO object, and neither SHALL overwrite the other's file

#### Scenario: PDF upload missing edition field
- **WHEN** an authenticated DM sends a multipart `POST /api/monsters` without an `edition` form value
- **THEN** the server SHALL return HTTP 400

#### Scenario: PDF upload exceeds size limit
- **WHEN** a client sends a PDF file larger than 20 MB
- **THEN** the server SHALL reject the request and return HTTP 413

#### Scenario: MinIO is unreachable during PDF upload
- **WHEN** a client sends a valid PDF upload request but MinIO cannot be reached
- **THEN** the server SHALL return HTTP 502 with an error message; no document SHALL be written to MongoDB

### Requirement: DM can look up a monster template by exact name
The system SHALL expose `GET /api/monsters/:name` that returns the **official** monster document (from the `monsters` collection) for the given name, including its MongoDB `id`. This endpoint SHALL NOT search the `custom_monsters` collection, since custom monster names are not unique across owners.

#### Scenario: Official monster found
- **WHEN** a client sends `GET /api/monsters/Goblin` and an official document with `name: "Goblin"` exists in the `monsters` collection
- **THEN** the server SHALL return HTTP 200 with the document as JSON, including its `id`

#### Scenario: Monster not found
- **WHEN** a client sends `GET /api/monsters/Unknown` and no matching official document exists
- **THEN** the server SHALL return HTTP 404

### Requirement: Backend can stream a monster PDF from MinIO
The system SHALL expose `GET /api/monsters/:name/pdf` that proxies the PDF object from MinIO for an **official** monster document (from the `monsters` collection) to the client as a streamed response. This endpoint SHALL NOT resolve custom monsters.

#### Scenario: PDF streamed successfully
- **WHEN** a client sends `GET /api/monsters/Beholder/pdf` and the official monster document has `pdf_object_key` set and the object exists in MinIO
- **THEN** the server SHALL stream the object bytes with `Content-Type: application/pdf` and HTTP 200

#### Scenario: Monster has no PDF
- **WHEN** a client sends `GET /api/monsters/Goblin/pdf` and the official monster document has no `pdf_object_key`
- **THEN** the server SHALL return HTTP 404

#### Scenario: MinIO object missing
- **WHEN** a client sends `GET /api/monsters/Beholder/pdf`, the official document has `pdf_object_key` set, but the object does not exist in MinIO
- **THEN** the server SHALL return HTTP 404

## ADDED Requirements

### Requirement: DM can look up a custom monster template by id
The system SHALL expose `GET /api/monsters/custom/:id` that returns the `custom_monsters` document for the given MongoDB id, including `owner_id`, `owner_display_name`, and `private`. The request SHALL require authentication. If the document is `private: true` and `owner_id` does not match the authenticated requester, the server SHALL return HTTP 403 instead of the document.

#### Scenario: Owner fetches their own private custom monster
- **WHEN** an authenticated DM requests `GET /api/monsters/custom/:id` for their own document with `private: true`
- **THEN** the server SHALL return HTTP 200 with the document

#### Scenario: Non-owner fetches a private custom monster
- **WHEN** an authenticated DM requests `GET /api/monsters/custom/:id` for a document owned by a different user with `private: true`
- **THEN** the server SHALL return HTTP 403

#### Scenario: Any authenticated DM fetches a public custom monster
- **WHEN** an authenticated DM requests `GET /api/monsters/custom/:id` for a document with `private: false` owned by a different user
- **THEN** the server SHALL return HTTP 200 with the document

#### Scenario: Custom monster not found
- **WHEN** an authenticated DM requests `GET /api/monsters/custom/:id` for an id with no matching document
- **THEN** the server SHALL return HTTP 404

### Requirement: DM can edit their own custom monster template
The system SHALL expose `PUT /api/monsters/custom/:id` accepting the same fields as creation (`name`, `max_hp`, `edition`, `private`, and statblock-reference fields). The request SHALL require authentication. If `owner_id` on the existing document does not match the authenticated requester, the server SHALL return HTTP 403 and SHALL NOT modify the document. On success, the server SHALL update the document in place (same `id`) and best-effort mirror the change into Typesense.

#### Scenario: Owner edits their own custom monster
- **WHEN** the owning DM sends `PUT /api/monsters/custom/:id` with updated `max_hp` and `private` values
- **THEN** the server SHALL update the document, preserve its `id`, and return HTTP 200 with the updated document

#### Scenario: Non-owner attempts to edit
- **WHEN** a DM who does not own the document sends `PUT /api/monsters/custom/:id`
- **THEN** the server SHALL return HTTP 403 and SHALL NOT modify the document

### Requirement: DM can delete their own custom monster template
The system SHALL expose `DELETE /api/monsters/custom/:id`. The request SHALL require authentication. If `owner_id` on the existing document does not match the authenticated requester, the server SHALL return HTTP 403 and SHALL NOT delete the document. On success, the server SHALL remove the document from `custom_monsters` and best-effort remove the corresponding document from the Typesense index (failure to remove from Typesense SHALL be logged and SHALL NOT cause the endpoint to report failure).

#### Scenario: Owner deletes their own custom monster
- **WHEN** the owning DM sends `DELETE /api/monsters/custom/:id`
- **THEN** the server SHALL remove the document from `custom_monsters`, best-effort remove it from Typesense, and return HTTP 204

#### Scenario: Non-owner attempts to delete
- **WHEN** a DM who does not own the document sends `DELETE /api/monsters/custom/:id`
- **THEN** the server SHALL return HTTP 403 and SHALL NOT delete the document

#### Scenario: Typesense removal fails
- **WHEN** the MongoDB deletion succeeds but the Typesense document removal fails (e.g. Typesense unreachable)
- **THEN** the server SHALL log the failure and still return HTTP 204 for the delete request

### Requirement: DM can list their own custom monster templates
The system SHALL expose `GET /api/monsters/custom?mine=true` (or equivalent) that returns all `custom_monsters` documents whose `owner_id` matches the authenticated requester, regardless of their `private` value. The request SHALL require authentication.

#### Scenario: DM lists their own monsters
- **WHEN** an authenticated DM who has created three custom monsters (two public, one private) requests their own list
- **THEN** the server SHALL return all three documents

#### Scenario: DM's list excludes other owners' monsters
- **WHEN** an authenticated DM requests their own list and another DM has also created custom monsters
- **THEN** the response SHALL NOT include the other DM's documents, public or private

### Requirement: Backend can stream a custom monster PDF by id
The system SHALL expose `GET /api/monsters/custom/:id/pdf` that proxies the PDF object from MinIO for a `custom_monsters` document. The request SHALL require authentication. If the document is `private: true` and `owner_id` does not match the authenticated requester, the server SHALL return HTTP 403 instead of streaming the file.

#### Scenario: Owner streams their own private monster's PDF
- **WHEN** the owning DM requests `GET /api/monsters/custom/:id/pdf` for their own private document with `pdf_object_key` set
- **THEN** the server SHALL stream the PDF with HTTP 200

#### Scenario: Non-owner requests a private monster's PDF
- **WHEN** a different DM requests `GET /api/monsters/custom/:id/pdf` for a private document they do not own
- **THEN** the server SHALL return HTTP 403 and SHALL NOT stream the file
