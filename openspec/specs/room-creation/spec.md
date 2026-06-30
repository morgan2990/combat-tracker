# Spec: Room Creation

## Purpose

Defines how DMs create new combat rooms and how room state is stored. A room is the top-level container for a combat session, identified by a short unique ID and protected by a DM token.

## Requirements

### Requirement: DM can create a combat room
The system SHALL provide a `POST /api/rooms` endpoint that creates a new combat room and returns a unique room ID and DM token. The request body MAY include an optional `edition` field (`"5e"` or `"5.5e"`); if omitted or invalid, the server SHALL default to `"5e"`.

#### Scenario: Successful room creation with edition
- **WHEN** a client sends `POST /api/rooms` with body `{ "edition": "5.5e" }`
- **THEN** the server SHALL respond with HTTP 201 and a JSON body containing a unique `room_id`, a `dm_token`, and the resolved `edition: "5.5e"`

#### Scenario: Successful room creation without edition defaults to 5e
- **WHEN** a client sends `POST /api/rooms` with no body
- **THEN** the server SHALL respond with HTTP 201 and `edition: "5e"` in the response

#### Scenario: Room ID is unique among active rooms
- **WHEN** a room is created and a room with the generated ID already exists in memory
- **THEN** the server SHALL regenerate the ID until a unique one is found before responding

#### Scenario: Room initialized with empty state
- **WHEN** a room is created
- **THEN** the room's combat state SHALL contain no entities, `is_started` set to false, `round` set to 0, `active_index` set to 0, and `edition` set to the resolved value

### Requirement: DM can create a room from the browser UI
The DM join screen SHALL present an edition selector (5e / 5.5e) alongside the DM name field before the room is created. The selected edition SHALL be included in the `POST /api/rooms` request body.

#### Scenario: DM creates a room with edition selected
- **WHEN** the DM selects "5.5e", enters their display name, and clicks "Create New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5.5e" }`, receive `room_id`, `dm_token`, and `edition` from the response, and immediately open a WebSocket connection to that room

#### Scenario: DM creates a room without changing the default edition
- **WHEN** the DM does not change the edition selector (default: "5e") and clicks "Create New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5e" }`

#### Scenario: DM can still rejoin an existing room manually
- **WHEN** the DM enters a room code and DM token in the "Rejoin Existing Room" form and clicks Rejoin
- **THEN** the client SHALL open a WebSocket connection to the specified room using the provided credentials

#### Scenario: Room creation API failure is surfaced to the DM
- **WHEN** `POST /api/rooms` returns a non-2xx status
- **THEN** the client SHALL display an error message and SHALL NOT attempt a WebSocket connection

### Requirement: Room state is mirrored to MongoDB
The system SHALL store the authoritative, fast-path room state in the Go server's process memory, and additionally mirror that state to a MongoDB `rooms` collection so it survives a server restart.

#### Scenario: State is accessible after creation
- **WHEN** a room has been created
- **THEN** subsequent WebSocket connections referencing that `room_id` SHALL find the room and its state in the server's in-memory registry

#### Scenario: State survives a restart
- **WHEN** the server process restarts after a room has had at least one persisted snapshot
- **THEN** a subsequent lookup for that room's `room_id` SHALL restore its state (entities, round, active turn, edition) from MongoDB rather than reporting the room as not found
