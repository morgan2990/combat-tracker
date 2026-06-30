## ADDED Requirements

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
