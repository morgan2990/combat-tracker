## ADDED Requirements

### Requirement: DM can edit any entity's stats without ownership restriction
The DM SHALL be able to send a `dm_update_entity` action to modify HP, temp HP, conditions, initiative, dead state, and (for creatures only) name of any entity in the room, bypassing the ownership checks applied to player-initiated updates.

#### Scenario: DM updates a creature's HP, conditions, and name
- **WHEN** a DM-role client sends `{ "type": "dm_update_entity", "entity_id": "...", "name": "Goblin Chief", "current_hp": 8, "temp_hp": 0, "initiative": 14, "conditions": ["Stunned"], "dead": false }`
- **THEN** the server SHALL apply all provided fields to the target entity, clamp `current_hp` to `[0, max_hp]`, and broadcast the updated `RoomState`

#### Scenario: DM updates a player entity's HP and conditions
- **WHEN** a DM-role client sends `dm_update_entity` targeting a `player`-type entity with updated `current_hp` and `conditions`
- **THEN** the server SHALL apply the HP and condition changes; the `name` field SHALL be ignored for player-type entities even if provided

#### Scenario: DM updates a companion entity's HP
- **WHEN** a DM-role client sends `dm_update_entity` targeting a `companion`-type entity
- **THEN** the server SHALL apply the HP, condition, and initiative changes; the `name` field SHALL be ignored for companion-type entities even if provided

#### Scenario: Initiative change triggers re-sort with active entity preservation
- **WHEN** a DM-role client sends `dm_update_entity` with an `initiative` value that differs from the entity's current initiative
- **THEN** the server SHALL update the entity's initiative, re-sort `State.Entities` descending by initiative, update `active_index` to continue pointing at the same entity that was active before the sort, and broadcast the updated `RoomState`

#### Scenario: DM HP input is clamped server-side
- **WHEN** a DM sends `dm_update_entity` with `current_hp` greater than the entity's `max_hp`
- **THEN** the server SHALL clamp `current_hp` to `max_hp` before applying the update

#### Scenario: Non-DM cannot use dm_update_entity
- **WHEN** a player-role client sends `dm_update_entity`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: DM HP field supports smart delta and absolute input
The DM client SHALL parse the HP input value before sending: a value prefixed with `+` or `-` is treated as a delta applied to the current HP; a bare number is treated as the absolute target value. The resolved absolute HP is sent to the server.

#### Scenario: DM applies a positive delta
- **WHEN** the DM types `+5` in the HP field for an entity with `current_hp: 20` and `max_hp: 32`
- **THEN** the client SHALL send `current_hp: 25` in the `dm_update_entity` message

#### Scenario: DM applies a negative delta
- **WHEN** the DM types `-12` in the HP field for an entity with `current_hp: 20`
- **THEN** the client SHALL send `current_hp: 8` in the `dm_update_entity` message; if the result is below 0 the client SHALL clamp it to 0 before sending

#### Scenario: DM sets an absolute HP value
- **WHEN** the DM types `15` (no sign prefix) in the HP field
- **THEN** the client SHALL send `current_hp: 15` directly, clamped to `[0, max_hp]`
