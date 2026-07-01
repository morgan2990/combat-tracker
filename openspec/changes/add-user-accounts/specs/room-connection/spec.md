## MODIFIED Requirements

### Requirement: Users connect to a room via WebSocket

The system SHALL provide a WebSocket endpoint at `GET /ws` that accepts connection parameters as query strings: `room_id`, `role` (`dm` or `player`), and, when `role=player`, `pc_id` identifying which of the connecting user's own PCs to bring into the room. The connecting user's identity SHALL be resolved from the session cookie sent automatically with the upgrade request — no `dm_token` or freeform `name` parameter is accepted. Room lookup SHALL check the in-memory registry first and, on a miss, attempt to restore the room from MongoDB before deciding the room does not exist.

#### Scenario: Valid player connection
- **WHEN** an authenticated client sends a WebSocket upgrade to `/ws?room_id=X7K2P&role=player&pc_id=pc_8f2a` and `pc_8f2a` belongs to the connecting user
- **THEN** the server SHALL validate the room exists (in memory or restorable from MongoDB) and that PC is not already claimed by an **active** connection in that room, upgrade the connection, register the player using the PC's stored name, and immediately broadcast the full room state to all connected clients

#### Scenario: Valid DM connection
- **WHEN** an authenticated client sends a WebSocket upgrade to `/ws?room_id=X7K2P&role=dm` and the connecting user is that room's `owner_user_id`
- **THEN** the server SHALL validate the room exists (in memory or restorable from MongoDB), upgrade the connection, and immediately broadcast the full room state to all connected clients

#### Scenario: Connection rejected when not authenticated
- **WHEN** a client without a valid session cookie attempts a WebSocket upgrade to `/ws`
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4001

#### Scenario: Invalid room ID
- **WHEN** a client attempts to connect with a `room_id` that does not exist in the server's in-memory registry **and** is not found in MongoDB
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4004

#### Scenario: Room restored from MongoDB on connect
- **WHEN** a client attempts to connect with a `room_id` absent from the in-memory registry but present in MongoDB
- **THEN** the server SHALL restore the room into the in-memory registry, then proceed with normal validation (ownership or PC ownership), and broadcast the full restored room state on success

#### Scenario: Connecting as DM without owning the room
- **WHEN** an authenticated client attempts to connect with `role=dm` to a room whose `owner_user_id` does not match the connecting user
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4003

#### Scenario: Connecting with a PC you don't own
- **WHEN** an authenticated client attempts to connect with `role=player` and a `pc_id` that does not belong to the connecting user
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4003

#### Scenario: Duplicate PC on active connection
- **WHEN** a client attempts to connect with `role=player` and a `pc_id` already registered by a **currently active** WebSocket connection in that room
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4009

#### Scenario: Player reconnects with a PC matching an existing entity
- **WHEN** a client connects with `role=player` and a `pc_id` that matches an entity already in `State.Entities` for that room but has **no active connection** holding that PC
- **THEN** the server SHALL accept the connection, update the matching entity's `session_id` to the new session, and broadcast the full room state

### Requirement: Connections are cleaned up on disconnect

The system SHALL remove a client's session and free their PC's claim on a room when their WebSocket connection closes.

#### Scenario: Player disconnects
- **WHEN** a player's WebSocket connection closes (normally or abnormally)
- **THEN** the server SHALL free the player's `pc_id` claim on that room so it can be reclaimed by a new connection

#### Scenario: State broadcast after disconnect
- **WHEN** a client disconnects
- **THEN** the server SHALL broadcast the updated room state to all remaining connected clients
