## ADDED Requirements

### Requirement: Room state is persisted to MongoDB
The system SHALL store a snapshot of each room's combat state in a MongoDB `rooms` collection, keyed by a unique `room_id` index. The persisted document SHALL include `room_id`, `dm_token`, `is_combat_active`, `current_round`, `active_turn_entity_id` (nullable), `edition`, and `entities`. Each persisted entity SHALL carry enough fields to fully reconstitute a `room.Entity` on restore — `id`, `name`, `type`, `owner_id`, `max_hp`, `current_hp`, `temp_hp`, `initiative`, `shares_initiative`, `conditions`, `dead`, `source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`, `initiative_roll` — plus a `connected` field. A narrower shape that omits `type`, `owner_id`, or `dead` is insufficient: combat logic (turn filtering, companion ownership, dead/alive state) depends on these fields being present after a restore.

#### Scenario: Snapshot reflects combat-active state
- **WHEN** a room's snapshot is persisted while `State.IsStarted` is true
- **THEN** the document SHALL have `is_combat_active: true`, `current_round` set to `State.Round`, and `active_turn_entity_id` set to the `id` of the entity at `State.ActiveIndex`

#### Scenario: Snapshot reflects pre-combat state
- **WHEN** a room's snapshot is persisted while `State.IsStarted` is false
- **THEN** the document SHALL have `is_combat_active: false` and `active_turn_entity_id` set to `null`

#### Scenario: Connection status reflects live client map
- **WHEN** a snapshot is built for an entity of type `player` whose `session_id` is a key in the room's current `Clients` map
- **THEN** that entity's `connected` field in the snapshot SHALL be `true`; if no such live session exists, it SHALL be `false`

#### Scenario: Non-player entities have no independent connection
- **WHEN** a snapshot is built for an entity of type `creature` or `companion`
- **THEN** the `connected` field SHALL NOT be derived from any `session_id` lookup (these entity types carry no `session_id`)

### Requirement: Room state changes trigger persistence
The system SHALL write the current room snapshot to MongoDB immediately (as a fire-and-forget operation that does not block the WebSocket message loop) whenever any of the following occurs: a player joins, a player leaves, combat is started, combat is ended, or the active turn advances to the next entity. All other state-mutating events SHALL mark the room dirty without writing immediately.

#### Scenario: Immediate write on combat start
- **WHEN** the DM starts combat in a room
- **THEN** the server SHALL persist a snapshot of that room's state to MongoDB without waiting for the periodic sweep

#### Scenario: Immediate write on turn advance
- **WHEN** the active turn advances to the next entity (via `next_turn`)
- **THEN** the server SHALL persist a snapshot reflecting the new `active_turn_entity_id` and, if the round wrapped, the new `current_round`

#### Scenario: Immediate write on join and leave
- **WHEN** a player or DM connects to or disconnects from a room
- **THEN** the server SHALL persist a snapshot of that room's state reflecting the updated connection

#### Scenario: Non-trigger mutation only marks the room dirty
- **WHEN** an entity's HP, conditions, or initiative are updated outside of the five named trigger events
- **THEN** the server SHALL mark the room dirty without performing an immediate MongoDB write

### Requirement: A periodic sweep persists dirty rooms
The system SHALL run a background process that, every 30 seconds, persists a snapshot for every room marked dirty since its last successful save, then clears that room's dirty flag.

#### Scenario: Sweep persists a dirty room
- **WHEN** the periodic sweep runs and a room has been marked dirty since its last save
- **THEN** the server SHALL write a current snapshot of that room to MongoDB and clear its dirty flag

#### Scenario: Sweep skips clean rooms
- **WHEN** the periodic sweep runs and a room has not been marked dirty since its last save
- **THEN** the server SHALL NOT perform a MongoDB write for that room

### Requirement: Rooms are restored from MongoDB when absent from memory
The system SHALL provide a single lookup path that first checks the in-memory room registry, and on a miss, queries MongoDB by `room_id`. If found in MongoDB, the server SHALL inflate the room's state back into the in-memory registry (with an empty connection map, since no WebSocket connections survive a process restart) before returning it.

#### Scenario: Room found in memory
- **WHEN** a lookup is performed for a `room_id` present in the in-memory registry
- **THEN** the server SHALL return that room directly without querying MongoDB

#### Scenario: Room found only in MongoDB
- **WHEN** a lookup is performed for a `room_id` absent from the in-memory registry but present in the `rooms` MongoDB collection
- **THEN** the server SHALL decode the stored snapshot into a new in-memory room (restoring `room_id`, `dm_token`, combat state, and entities, with `active_index` resolved from `active_turn_entity_id`), register it in the in-memory registry, and return it

#### Scenario: Active turn ID cannot be resolved on restore
- **WHEN** a restored snapshot's `active_turn_entity_id` does not match any entity in the restored `entities` list
- **THEN** the server SHALL set the restored room's `active_index` to `0` rather than failing the restore

#### Scenario: Room not found anywhere
- **WHEN** a lookup is performed for a `room_id` absent from both the in-memory registry and MongoDB
- **THEN** the lookup SHALL report not-found to the caller

### Requirement: A REST endpoint exposes room existence and restoration
The system SHALL provide a `GET /api/rooms/{room_id}` endpoint that performs the in-memory-then-MongoDB lookup and returns room metadata on success.

#### Scenario: Room exists (in memory or MongoDB)
- **WHEN** a client sends `GET /api/rooms/{room_id}` for a room found in memory or restorable from MongoDB
- **THEN** the server SHALL respond with HTTP 200 and the room's `edition` and `is_combat_active` fields

#### Scenario: Room does not exist
- **WHEN** a client sends `GET /api/rooms/{room_id}` for a `room_id` not found in memory or MongoDB
- **THEN** the server SHALL respond with HTTP 404
