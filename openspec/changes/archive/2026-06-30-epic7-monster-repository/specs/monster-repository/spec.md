## ADDED Requirements

### Requirement: DM can register a URL-type monster template
The system SHALL expose `POST /api/monsters` accepting a JSON body with `name` (string, unique identifier), `max_hp` (integer), and optional fields: `source_type` ("url" | "pdf"), `reference_url` (string), `five_etools_id` (string), `source_book` (string). The document SHALL be upserted into the `monsters` collection keyed by `name`.

#### Scenario: DM registers a new URL-type monster
- **WHEN** a client sends `POST /api/monsters` with JSON `{ "name": "Goblin", "max_hp": 7, "source_type": "url", "reference_url": "https://example.com/goblin.webp" }`
- **THEN** the server SHALL upsert the document into the `monsters` collection and return HTTP 200

#### Scenario: DM registers a monster with no statblock reference
- **WHEN** a client sends `POST /api/monsters` with only `name` and `max_hp`
- **THEN** the server SHALL upsert the document with `source_type` absent and return HTTP 200

#### Scenario: DM re-registers an existing monster
- **WHEN** a client sends `POST /api/monsters` with a `name` that already exists in the collection
- **THEN** the server SHALL replace the existing document with the new data and return HTTP 200

### Requirement: DM can register a PDF-type monster template
The system SHALL expose `POST /api/monsters` accepting `multipart/form-data` with fields `name`, `max_hp`, `source_type: "pdf"`, and a file part named `pdf`. The PDF SHALL be uploaded to MinIO under key `monsters/{name}.pdf`; `pdf_object_key` SHALL be set on the stored document.

#### Scenario: DM registers a PDF-type monster
- **WHEN** a client sends `POST /api/monsters` as multipart with a valid PDF file in the `pdf` field and `source_type: "pdf"`
- **THEN** the server SHALL upload the file to MinIO, store the document with `pdf_object_key: "monsters/{name}.pdf"`, and return HTTP 200

#### Scenario: PDF upload exceeds size limit
- **WHEN** a client sends a PDF file larger than 20 MB
- **THEN** the server SHALL reject the request and return HTTP 413

#### Scenario: MinIO is unreachable during PDF upload
- **WHEN** a client sends a valid PDF upload request but MinIO cannot be reached
- **THEN** the server SHALL return HTTP 502 with an error message; no document SHALL be written to MongoDB

### Requirement: DM can look up a monster template by exact name
The system SHALL expose `GET /api/monsters/:name` that returns the monster document for the given name.

#### Scenario: Monster found
- **WHEN** a client sends `GET /api/monsters/Goblin` and a document with `name: "Goblin"` exists in the `monsters` collection
- **THEN** the server SHALL return HTTP 200 with the document as JSON

#### Scenario: Monster not found
- **WHEN** a client sends `GET /api/monsters/Unknown` and no matching document exists
- **THEN** the server SHALL return HTTP 404

### Requirement: Backend can stream a monster PDF from MinIO
The system SHALL expose `GET /api/monsters/:name/pdf` that proxies the PDF object from MinIO to the client as a streamed response.

#### Scenario: PDF streamed successfully
- **WHEN** a client sends `GET /api/monsters/Beholder/pdf` and the monster document has `pdf_object_key` set and the object exists in MinIO
- **THEN** the server SHALL stream the object bytes with `Content-Type: application/pdf` and HTTP 200

#### Scenario: Monster has no PDF
- **WHEN** a client sends `GET /api/monsters/Goblin/pdf` and the monster document has no `pdf_object_key`
- **THEN** the server SHALL return HTTP 404

#### Scenario: MinIO object missing
- **WHEN** a client sends `GET /api/monsters/Beholder/pdf`, the document has `pdf_object_key` set, but the object does not exist in MinIO
- **THEN** the server SHALL return HTTP 404
