# Spec: Player Character Setup

## Purpose

Defines how a player establishes their entity in a room after connecting via WebSocket, including the setup form flow, server-side entity creation, and reconnection re-linking for returning players.

## Requirements

### Requirement: Player configures their character after connecting
After a WebSocket connection is established, a new player (one without an existing entity in the room) SHALL be presented with a setup form to enter their Max HP and Initiative before appearing in the tracker.

#### Scenario: New player submits setup form
- **WHEN** a player without an existing entity sends `{ "type": "setup_character", "max_hp": N, "initiative": M }` over their WebSocket connection
- **THEN** the server SHALL create a new entity with `type: "player"`, the player's name, the provided `max_hp` and `initiative`, `current_hp` equal to `max_hp`, and `session_id` equal to the player's session ID

#### Scenario: Entity appears sorted in tracker after setup
- **WHEN** a player's entity is created via `setup_character`
- **THEN** the server SHALL insert the entity into `State.Entities`, re-sort the slice descending by initiative, and broadcast the updated `RoomState` to all connected clients

#### Scenario: Setup rejected with invalid values
- **WHEN** a player sends `setup_character` with `max_hp` ≤ 0
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: Reconnecting player skips setup and re-links to existing entity
If a player connects with a name that matches an entity already present in `State.Entities`, the server SHALL re-link that entity to the new session instead of treating the name as a conflict.

#### Scenario: Entity re-linked on reconnection
- **WHEN** a player connects with a name matching an existing `player`-type entity in `State.Entities`
- **THEN** the server SHALL update that entity's `session_id` to the new session ID and register the connection normally

#### Scenario: Client detects existing entity and skips setup form
- **WHEN** a newly connected player-role client receives the first `RoomState` broadcast
- **THEN** the client SHALL check whether any entity matches `name === myName && type === "player"`; if found, it SHALL display the combat view directly without showing the setup form

#### Scenario: Re-linked player retains ownership of companions
- **WHEN** a player reconnects and their entity is re-linked to the new session
- **THEN** companion entities whose `owner_id` equals the re-linked entity's `id` SHALL remain editable by that player in the new session
