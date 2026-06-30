## MODIFIED Requirements

### Requirement: DM can register a URL-type monster template
The system SHALL expose `POST /api/monsters` accepting a JSON body with `name` (string), `max_hp` (integer), `edition` (string, required, one of `"5e"` or `"5.5e"`), and optional fields: `source_type` ("url" | "pdf"), `reference_url` (string), `five_etools_id` (string), `source_book` (string). The document SHALL be upserted into the `monsters` collection keyed by `{ name, edition }` and SHALL set `is_custom: true`. The response SHALL include the document's MongoDB `id`. The upsert SHALL also be mirrored into the Typesense search index on a best-effort basis.

#### Scenario: DM registers a new URL-type monster
- **WHEN** a client sends `POST /api/monsters` with JSON `{ "name": "Goblin", "max_hp": 7, "edition": "5e", "source_type": "url", "reference_url": "https://2014.5e.tools/bestiary/goblin-mm.html" }`
- **THEN** the server SHALL upsert the document with `is_custom: true`, return HTTP 200 with the document's `id` included in the response, and best-effort mirror the document into Typesense

#### Scenario: DM registers a monster with no statblock reference
- **WHEN** a client sends `POST /api/monsters` with only `name`, `max_hp`, and `edition`
- **THEN** the server SHALL upsert the document with `is_custom: true`, `source_type` absent, and return HTTP 200 with an `id`

#### Scenario: DM re-registers an existing monster for the same edition
- **WHEN** a client sends `POST /api/monsters` with a `name` and `edition` combination that already exists in the collection
- **THEN** the server SHALL replace the existing document, return HTTP 200, and the response `id` SHALL match the existing document's unchanged `id`

#### Scenario: Edition field missing from request
- **WHEN** a client sends `POST /api/monsters` without an `edition` field
- **THEN** the server SHALL return HTTP 400

#### Scenario: DM registers a monster with the same name as a scraped monster in a different edition
- **WHEN** a scrubbed document `{ name: "Goblin", edition: "5.5e" }` exists and a client sends `POST /api/monsters` with `{ "name": "Goblin", "edition": "5e", "max_hp": 7 }`
- **THEN** the server SHALL insert a new document `{ name: "Goblin", edition: "5e", is_custom: true }` with its own `id`, without affecting the scraped document

### Requirement: DM can register a PDF-type monster template
The system SHALL expose `POST /api/monsters` accepting `multipart/form-data` with fields `name`, `max_hp`, `edition` (required), `source_type: "pdf"`, and a file part named `pdf`. The PDF SHALL be uploaded to MinIO under key `monsters/{name}.pdf`; `pdf_object_key` SHALL be set on the stored document. The document SHALL set `is_custom: true`. The response SHALL include the document's MongoDB `id`, and the upsert SHALL also be mirrored into the Typesense search index on a best-effort basis.

#### Scenario: DM registers a PDF-type monster
- **WHEN** a client sends `POST /api/monsters` as multipart with a valid PDF file, `source_type: "pdf"`, and a valid `edition`
- **THEN** the server SHALL upload the file to MinIO, store the document with `pdf_object_key: "monsters/{name}.pdf"`, `is_custom: true`, and an `id`, then return HTTP 200

#### Scenario: PDF upload missing edition field
- **WHEN** a client sends a multipart `POST /api/monsters` without an `edition` form value
- **THEN** the server SHALL return HTTP 400

#### Scenario: PDF upload exceeds size limit
- **WHEN** a client sends a PDF file larger than 20 MB
- **THEN** the server SHALL reject the request and return HTTP 413

#### Scenario: MinIO is unreachable during PDF upload
- **WHEN** a client sends a valid PDF upload request but MinIO cannot be reached
- **THEN** the server SHALL return HTTP 502 with an error message; no document SHALL be written to MongoDB

### Requirement: DM can look up a monster template by exact name
The system SHALL expose `GET /api/monsters/:name` that returns the monster document for the given name, including its MongoDB `id`.

#### Scenario: Monster found
- **WHEN** a client sends `GET /api/monsters/Goblin` and a document with `name: "Goblin"` exists in the `monsters` collection
- **THEN** the server SHALL return HTTP 200 with the document as JSON, including its `id`

#### Scenario: Monster not found
- **WHEN** a client sends `GET /api/monsters/Unknown` and no matching document exists
- **THEN** the server SHALL return HTTP 404
