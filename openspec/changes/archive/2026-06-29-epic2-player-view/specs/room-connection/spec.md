## MODIFIED Requirements

### Requirement: Users connect to a room via WebSocket
The system SHALL provide a WebSocket endpoint at `GET /ws` that accepts connection parameters as query strings: `room_id`, `name`, `role` (`dm` or `player`), and optionally `dm_token`.

#### Scenario: Valid player connection
- **WHEN** a client sends a WebSocket upgrade to `/ws?room_id=X7K2P&name=Aragorn&role=player`
- **THEN** the server SHALL validate the room exists and the name is not already claimed by an **active** connection, upgrade the connection, register the player, and immediately broadcast the full room state to all connected clients

#### Scenario: Valid DM connection
- **WHEN** a client sends a WebSocket upgrade to `/ws?room_id=X7K2P&name=DungeonMaster&role=dm&dm_token=abc12345`
- **THEN** the server SHALL validate the room exists and the token matches, upgrade the connection, and immediately broadcast the full room state to all connected clients

#### Scenario: Invalid room ID
- **WHEN** a client attempts to connect with a `room_id` that does not exist in the server registry
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4004

#### Scenario: Incorrect DM token
- **WHEN** a client attempts to connect with `role=dm` and a `dm_token` that does not match the room's stored token
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4003

#### Scenario: Duplicate player name on active connection
- **WHEN** a client attempts to connect with `role=player` and a `name` already registered by a **currently active** WebSocket connection in that room
- **THEN** the server SHALL reject the WebSocket upgrade with close code 4009

#### Scenario: Player reconnects with name matching existing entity
- **WHEN** a client connects with `role=player` and a `name` that matches an entity in `State.Entities` but has **no active connection** holding that name
- **THEN** the server SHALL accept the connection, update the matching entity's `session_id` to the new session, and broadcast the full room state
