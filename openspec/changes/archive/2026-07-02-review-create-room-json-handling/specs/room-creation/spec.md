## MODIFIED Requirements

### Requirement: DM can create a combat room

The system SHALL provide a `POST /api/rooms` endpoint, available only to authenticated users, that creates a new combat room owned by the requesting user and returns a unique room ID. The request body MAY include an optional `edition` field (`"5e"` or `"5.5e"`); if the body is valid JSON but the `edition` field is omitted or not a recognized value, the server SHALL default to `"5e"`. If the request body is not valid JSON, the server SHALL respond with HTTP 400 and SHALL NOT create a room.

#### Scenario: Successful room creation with edition
- **WHEN** an authenticated client sends `POST /api/rooms` with body `{ "edition": "5.5e" }`
- **THEN** the server SHALL respond with HTTP 201 and a JSON body containing a unique `room_id` and the resolved `edition: "5.5e"`; the created room's `owner_user_id` SHALL be set to the requesting user's id

#### Scenario: Successful room creation without edition defaults to 5e
- **WHEN** an authenticated client sends `POST /api/rooms` with no body
- **THEN** the server SHALL respond with HTTP 201 and `edition: "5e"` in the response

#### Scenario: Successful room creation with an unrecognized edition value defaults to 5e
- **WHEN** an authenticated client sends `POST /api/rooms` with body `{ "edition": "3e" }`
- **THEN** the server SHALL respond with HTTP 201 and `edition: "5e"` in the response

#### Scenario: Room creation rejected for a malformed request body
- **WHEN** an authenticated client sends `POST /api/rooms` with a body that is not valid JSON
- **THEN** the server SHALL respond with HTTP 400 and SHALL NOT create a room

#### Scenario: Room creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/rooms`
- **THEN** the server SHALL respond with HTTP 401 and SHALL NOT create a room

#### Scenario: Room ID is unique among active rooms
- **WHEN** a room is created and a room with the generated ID already exists in memory
- **THEN** the server SHALL regenerate the ID until a unique one is found before responding

#### Scenario: Room initialized with empty state
- **WHEN** a room is created
- **THEN** the room's combat state SHALL contain no entities, `is_started` set to false, `round` set to 0, `active_index` set to 0, and `edition` set to the resolved value
