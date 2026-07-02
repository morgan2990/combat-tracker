# Spec: Room Creation

## Purpose

Defines how DMs create new combat rooms and how room state is stored. A room is the top-level container for a combat session, identified by a short unique ID and protected by a DM token.

## Requirements

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

### Requirement: DM can create a room from the browser UI

The DM Dashboard SHALL present an edition selector (5e / 5.5e) and a "+ New Room" action that creates a room owned by the logged-in user and immediately opens a WebSocket connection to it as DM — no token is returned, copied, or re-entered.

#### Scenario: DM creates a room with edition selected
- **WHEN** the logged-in DM selects "5.5e" and clicks "+ New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5.5e" }`, receive `room_id` and `edition` from the response, and immediately open a WebSocket connection to that room with `role=dm`

#### Scenario: DM creates a room without changing the default edition
- **WHEN** the logged-in DM does not change the edition selector (default: "5e") and clicks "+ New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5e" }`

#### Scenario: Room creation API failure is surfaced to the DM
- **WHEN** `POST /api/rooms` returns a non-2xx status
- **THEN** the client SHALL display an error message and SHALL NOT attempt a WebSocket connection

#### Scenario: Owned rooms appear on the dashboard without re-entering credentials
- **WHEN** a logged-in user has previously created one or more rooms
- **THEN** the Dashboard's "My Rooms" list SHALL show each room with an action to open it directly, with no room code or token required

### Requirement: Room state is mirrored to MongoDB
The system SHALL store the authoritative, fast-path room state in the Go server's process memory, and additionally mirror that state to a MongoDB `rooms` collection so it survives a server restart.

#### Scenario: State is accessible after creation
- **WHEN** a room has been created
- **THEN** subsequent WebSocket connections referencing that `room_id` SHALL find the room and its state in the server's in-memory registry

#### Scenario: State survives a restart
- **WHEN** the server process restarts after a room has had at least one persisted snapshot
- **THEN** a subsequent lookup for that room's `room_id` SHALL restore its state (entities, round, active turn, edition) from MongoDB rather than reporting the room as not found
