## ADDED Requirements

### Requirement: Room state has a defined data model
The system SHALL represent each room's combat state using the following structure:

- `room_id` (string): the room's unique identifier
- `is_started` (bool): whether combat has been started by the DM
- `round` (int): the current round number, starting at 0
- `active_index` (int): index into the entities slice for the currently active turn
- `entities` (array of Entity)

Each Entity SHALL have:
- `id` (string): UUID, assigned at creation
- `name` (string): display name
- `type` (string): one of `player`, `creature`, `companion`
- `owner_id` (string): for companions, the `id` of the owning player entity; empty otherwise
- `session_id` (string): the WS connection identifier for player-type entities; empty for creatures
- `max_hp` (int)
- `current_hp` (int)
- `temp_hp` (int)
- `initiative` (int)
- `conditions` (array of strings): e.g., `["Prone", "Stunned"]`

#### Scenario: Entity created with required fields
- **WHEN** any entity is added to a room
- **THEN** the entity SHALL have a server-generated UUID `id`, a non-empty `name`, a valid `type`, and numeric HP fields initialized to their given values

### Requirement: Full room state is broadcast on every mutation
The system SHALL serialize and send the complete `RoomState` as a JSON message to all connected clients whenever any part of the room state changes.

#### Scenario: Broadcast after entity added
- **WHEN** an entity is added to the room
- **THEN** the server SHALL broadcast the full updated `RoomState` JSON to every connected client in that room

#### Scenario: Broadcast after entity modified
- **WHEN** any field of any entity is updated (HP, conditions, initiative, etc.)
- **THEN** the server SHALL broadcast the full updated `RoomState` JSON to every connected client in that room

#### Scenario: Broadcast after turn advances
- **WHEN** the DM advances the turn
- **THEN** the server SHALL broadcast the updated `RoomState` (with new `active_index` and/or `round`) to every connected client

### Requirement: Room state access is concurrent-safe
The system SHALL protect all reads and writes to a room's state with a mutex to prevent data races under concurrent WebSocket connections.

#### Scenario: Concurrent modifications do not corrupt state
- **WHEN** two clients send state-mutating messages simultaneously
- **THEN** the server SHALL serialize the mutations (one completes before the other begins) and both clients SHALL receive a consistent final state

### Requirement: Frontend is responsible for role-based data presentation
The client application SHALL determine what data to display based on the connected user's role. The server sends identical full state to all clients.

#### Scenario: Player does not see exact creature HP
- **WHEN** a player-role client receives a `RoomState` broadcast
- **THEN** the player view SHALL NOT render exact `current_hp` or `max_hp` for entities with `type: "creature"`; it SHALL render a qualitative label instead (e.g., Healthy, Injured, Dying)

#### Scenario: DM sees full data for all entities
- **WHEN** a DM-role client receives a `RoomState` broadcast
- **THEN** the DM view SHALL render exact HP and all fields for every entity regardless of type
