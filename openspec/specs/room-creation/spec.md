# Spec: Room Creation

## Purpose

Defines how DMs create new combat rooms and how room state is stored. A room is the top-level container for a combat session, identified by a short unique ID and protected by a DM token.

## Requirements

### Requirement: DM can create a combat room
The system SHALL provide a `POST /api/rooms` endpoint that creates a new combat room and returns a unique room ID and DM token. No request body is required.

#### Scenario: Successful room creation
- **WHEN** a client sends `POST /api/rooms`
- **THEN** the server responds with HTTP 201 and a JSON body containing a unique `room_id` (5–6 alphanumeric characters) and a `dm_token` (random string)

#### Scenario: Room ID is unique among active rooms
- **WHEN** a room is created and a room with the generated ID already exists in memory
- **THEN** the server SHALL regenerate the ID until a unique one is found before responding

#### Scenario: Room initialized with empty state
- **WHEN** a room is created
- **THEN** the room's combat state SHALL contain no entities, `is_started` set to false, `round` set to 0, and `active_index` set to 0

### Requirement: DM can create a room from the browser UI
The DM join screen SHALL provide a "Create New Room" flow that calls `POST /api/rooms` from the browser and auto-connects to the created room, so the DM does not need to use external tools to obtain a room ID and DM token.

#### Scenario: DM creates a room via the UI
- **WHEN** the DM enters their display name and clicks "Create New Room"
- **THEN** the client SHALL call `POST /api/rooms`, receive `room_id` and `dm_token` from the response, and immediately open a WebSocket connection to that room using those credentials without requiring the DM to copy or enter any values manually

#### Scenario: DM can still rejoin an existing room manually
- **WHEN** the DM enters a room code and DM token in the "Rejoin Existing Room" form and clicks Rejoin
- **THEN** the client SHALL open a WebSocket connection to the specified room using the provided credentials, enabling reconnection after a page reload

#### Scenario: Room creation API failure is surfaced to the DM
- **WHEN** `POST /api/rooms` returns a non-2xx status
- **THEN** the client SHALL display an error message to the DM and SHALL NOT attempt a WebSocket connection

### Requirement: Room state is stored in memory only
The system SHALL store all room state in the Go server's process memory with no persistence layer.

#### Scenario: State is accessible after creation
- **WHEN** a room has been created
- **THEN** subsequent WebSocket connections referencing that `room_id` SHALL find the room and its state in the server's registry

#### Scenario: State is lost on restart
- **WHEN** the server process restarts
- **THEN** all room state SHALL be lost (no recovery mechanism is required)
