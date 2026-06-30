## MODIFIED Requirements

### Requirement: Users connect to a room via WebSocket
The system SHALL provide a WebSocket endpoint at `GET /ws` that accepts connection parameters as query strings: `room_id`, `name`, `role` (`dm` or `player`), and optionally `dm_token`. Room lookup SHALL check the in-memory registry first and, on a miss, attempt to restore the room from MongoDB before deciding the room does not exist.

#### Scenario: Valid player connection
- **WHEN** a client sends a WebSocket upgrade to `/ws?room_id=X7K2P&name=Aragorn&role=player`
- **THEN** the server SHALL validate the room exists (in memory or restorable from MongoDB) and the name is not already claimed by an **active** connection, upgrade the connection, register the player, and immediately broadcast the full room state to all connected clients

#### Scenario: Valid DM connection
- **WHEN** a client sends a WebSocket upgrade to `/ws?room_id=X7K2P&name=DungeonMaster&role=dm&dm_token=abc12345`
- **THEN** the server SHALL validate the room exists (in memory or restorable from MongoDB) and the token matches, upgrade the connection, and immediately broadcast the full room state to all connected clients

#### Scenario: Invalid room ID
- **WHEN** a client attempts to connect with a `room_id` that does not exist in the server's in-memory registry **and** is not found in MongoDB
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4004

#### Scenario: Room restored from MongoDB on connect
- **WHEN** a client attempts to connect with a `room_id` absent from the in-memory registry but present in MongoDB
- **THEN** the server SHALL restore the room into the in-memory registry, then proceed with normal validation (DM token or player name) and broadcast the full restored room state on success

#### Scenario: Incorrect DM token
- **WHEN** a client attempts to connect with `role=dm` and a `dm_token` that does not match the room's stored token
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4003

#### Scenario: Duplicate player name on active connection
- **WHEN** a client attempts to connect with `role=player` and a `name` already registered by a **currently active** WebSocket connection in that room
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4009

#### Scenario: Player reconnects with name matching existing entity
- **WHEN** a client connects with `role=player` and a `name` that matches an entity in `State.Entities` but has **no active connection** holding that name
- **THEN** the server SHALL accept the connection, update the matching entity's `session_id` to the new session, and broadcast the full room state
