## MODIFIED Requirements

### Requirement: Room state is persisted to MongoDB
The system SHALL store a snapshot of each room's combat state in a MongoDB `rooms` collection, keyed by a unique `room_id` index. The persisted document SHALL include `room_id`, `owner_user_id`, `is_combat_active`, `current_round`, `active_turn_entity_id` (nullable), `edition`, and `entities`. Each persisted entity SHALL carry enough fields to fully reconstitute a `room.Entity` on restore — `id`, `name`, `type`, `owner_id`, `max_hp`, `current_hp`, `temp_hp`, `initiative`, `shares_initiative`, `conditions`, `dead`, `source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`, `initiative_roll`, `display_name` — plus a `connected` field. A narrower shape that omits `type`, `owner_id`, `dead`, or `display_name` is insufficient: combat logic (turn filtering, companion ownership, dead/alive state) and player-facing name masking depend on these fields being present after a restore.

#### Scenario: Snapshot reflects combat-active state
- **WHEN** a room's snapshot is persisted while `State.IsStarted` is true
- **THEN** the document SHALL have `is_combat_active: true`, `current_round` set to `State.Round`, and `active_turn_entity_id` set to the `id` of the entity at `State.ActiveIndex`

#### Scenario: Snapshot reflects pre-combat state
- **WHEN** a room's snapshot is persisted while `State.IsStarted` is false
- **THEN** the document SHALL have `is_combat_active: false` and `active_turn_entity_id` set to `null`

#### Scenario: Connection status reflects live client map
- **WHEN** a snapshot is built for an entity of type `pc` whose `session_id` is a key in the room's current `Clients` map
- **THEN** that entity's `connected` field in the snapshot SHALL be `true`; if no such live session exists, it SHALL be `false`

#### Scenario: Non-PC entities have no independent connection
- **WHEN** a snapshot is built for an entity of type `creature` or `companion`
- **THEN** the `connected` field SHALL NOT be derived from any `session_id` lookup (these entity types carry no `session_id`)

#### Scenario: Display name survives a restart
- **WHEN** a room containing a creature with a non-empty `DisplayName` is persisted, the process restarts, and the room is restored via `GetOrRestoreRoom`
- **THEN** the restored entity's `DisplayName` SHALL equal the value it had before persistence

#### Scenario: Absent display name survives a restart as empty
- **WHEN** a room containing a creature with an empty `DisplayName` is persisted and later restored
- **THEN** the restored entity's `DisplayName` SHALL be the empty string, not `null` or a missing field causing a decode error
