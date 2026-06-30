# Spec: Monster Search

## Purpose

Defines the edition-filtered monster search endpoint used by the DM Combat Panel. This endpoint establishes the stable URL contract that Epic 12 (Typesense autocomplete) will fulfil — when Epic 12 lands, only the backend query changes; the route and response shape remain identical.

## ADDED Requirements

### Requirement: Backend exposes an edition-filtered monster search endpoint
The system SHALL expose `GET /api/search/monsters?q=<name>&edition=<edition>`. Both query parameters are required. The endpoint SHALL return HTTP 400 if either parameter is missing or if `edition` is not `"5e"` or `"5.5e"`. The endpoint SHALL query the `monsters` collection for an exact match on `{ name: q, edition: edition }` and return the result as a JSON array.

#### Scenario: Monster found for the given edition
- **WHEN** a client sends `GET /api/search/monsters?q=Goblin&edition=5e` and a document `{ name: "Goblin", edition: "5e" }` exists
- **THEN** the server SHALL return HTTP 200 with a JSON array containing that document

#### Scenario: Monster not found for the given edition
- **WHEN** a client sends `GET /api/search/monsters?q=Goblin&edition=5.5e` and no document matches `{ name: "Goblin", edition: "5.5e" }`
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]`

#### Scenario: Same name exists for a different edition only
- **WHEN** a document `{ name: "Goblin", edition: "5e" }` exists and a client sends `GET /api/search/monsters?q=Goblin&edition=5.5e`
- **THEN** the server SHALL return HTTP 200 with an empty JSON array — the 5e document SHALL NOT be returned

#### Scenario: Missing q parameter
- **WHEN** a client sends `GET /api/search/monsters?edition=5e` with no `q`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Invalid edition parameter
- **WHEN** a client sends `GET /api/search/monsters?q=Goblin&edition=4e`
- **THEN** the server SHALL return HTTP 400

---

### Requirement: DM Combat Panel search bar uses the edition-filtered endpoint
The DM Combat Panel monster search input SHALL dispatch requests to `GET /api/search/monsters` using the room's current `edition` from the WebSocket state, replacing any prior direct-lookup behaviour.

#### Scenario: DM searches for a monster in an active room
- **WHEN** the DM types a monster name into the search input in a room with `edition: "5e"`
- **THEN** the frontend SHALL request `GET /api/search/monsters?q=<input>&edition=5e` and display the returned result

#### Scenario: Search respects room edition automatically
- **WHEN** the room was created with `edition: "5.5e"` and the DM searches for "Goblin"
- **THEN** the frontend SHALL send `edition=5.5e` in the request without any manual configuration by the DM
