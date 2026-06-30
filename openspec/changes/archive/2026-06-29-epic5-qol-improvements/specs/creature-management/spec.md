## MODIFIED Requirements

### Requirement: DM can toggle the dead state of any entity
The DM SHALL be able to mark any entity as dead or revive it. Dead entities remain visible in the tracker and are rendered greyed-out on all clients. When the DM marks an entity as dead via the Kill action, the entity's `current_hp` MUST be set to 0 simultaneously. When reviving, `current_hp` remains at 0 until explicitly changed.

#### Scenario: DM marks an entity as dead
- **WHEN** a DM-role client triggers the Kill action for an entity that is currently alive
- **THEN** the client SHALL send `dm_update_entity` with both `dead: true` and `current_hp: 0`; the server SHALL update both fields atomically and broadcast the updated `RoomState`; all clients SHALL render the entity greyed-out

#### Scenario: DM revives a dead entity
- **WHEN** a DM-role client triggers the Revive action for an entity that is currently dead
- **THEN** the client SHALL send `dm_update_entity` with `dead: false`; the server SHALL set `dead = false` and broadcast; `current_hp` SHALL remain at 0 (Unconscious state) until the DM explicitly sets it to a non-zero value
