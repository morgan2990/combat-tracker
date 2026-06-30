# Spec: Creature Management

## Purpose

Defines how the DM adds ephemeral creature entities to a room, toggles their dead state, removes them individually, and batch-removes all dead creatures.

## Requirements

### Requirement: DM can add ephemeral creature entities to the room
The DM SHALL be able to add creature-type entities at any time, including during active combat. Creatures added mid-combat are sorted in by initiative with the active entity position preserved.

#### Scenario: DM adds a creature before combat
- **WHEN** a DM-role client sends `{ "type": "add_creature", "name": "Goblin", "max_hp": 14, "initiative": 11 }` and `is_started` is false
- **THEN** the server SHALL create a new entity with `type: "creature"`, the provided fields, `current_hp` equal to `max_hp`, `dead: false`, sort `State.Entities` descending by initiative, and broadcast the updated `RoomState`

#### Scenario: DM adds a creature mid-combat
- **WHEN** a DM-role client sends `add_creature` and `is_started` is true
- **THEN** the server SHALL create the creature entity, re-sort `State.Entities` descending by initiative, update `active_index` to continue pointing at the same entity that was active before the sort, and broadcast the updated `RoomState`

#### Scenario: Non-DM cannot add creatures
- **WHEN** a player-role client sends `add_creature`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: DM can toggle the dead state of any entity
The DM SHALL be able to mark any entity as dead or revive it. Dead entities remain visible in the tracker and are rendered greyed-out on all clients.

#### Scenario: DM marks an entity as dead
- **WHEN** a DM-role client sends `dm_update_entity` with `dead: true` for an entity that is currently alive
- **THEN** the server SHALL set `entity.dead = true` and broadcast the updated `RoomState`; all clients SHALL render the entity greyed-out

#### Scenario: DM revives a dead entity
- **WHEN** a DM-role client sends `dm_update_entity` with `dead: false` for an entity that is currently dead
- **THEN** the server SHALL set `entity.dead = false` and broadcast the updated `RoomState`; all clients SHALL render the entity normally

### Requirement: DM can remove individual entities from the tracker
The DM SHALL be able to hard-delete any entity from `State.Entities`. The `active_index` is adjusted to remain coherent after removal.

#### Scenario: DM removes an entity that is not currently active
- **WHEN** a DM-role client sends `{ "type": "remove_entity", "entity_id": "..." }` and the target entity's index differs from `active_index`
- **THEN** the server SHALL remove the entity; if the removed index was before `active_index`, decrement `active_index` by one; broadcast the updated `RoomState`

#### Scenario: DM removes the currently active entity
- **WHEN** a DM-role client sends `remove_entity` targeting the entity at `active_index`
- **THEN** the server SHALL remove the entity and keep `active_index` at its current value (which now points to the next entity), wrapping to 0 if the removed entity was last; broadcast the updated `RoomState`

#### Scenario: Non-DM cannot remove entities
- **WHEN** a player-role client sends `remove_entity`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: DM can batch-remove all dead creature entities
The DM SHALL be able to purge all entities that are both `dead: true` and `type: "creature"` with a single action. Player and companion entities are never removed by this action regardless of their dead state.

#### Scenario: DM removes all dead creatures
- **WHEN** a DM-role client sends `{ "type": "remove_dead_creatures" }`
- **THEN** the server SHALL remove every entity satisfying `dead == true AND type == "creature"`, adjust `active_index` if any removed entity was before or at the current active position, and broadcast the updated `RoomState`

#### Scenario: Remove all dead creatures with no eligible entities
- **WHEN** a DM sends `remove_dead_creatures` and no entities satisfy the filter
- **THEN** the server SHALL make no state change and send no broadcast

#### Scenario: Non-DM cannot batch-remove creatures
- **WHEN** a player-role client sends `remove_dead_creatures`
- **THEN** the server SHALL ignore the message and send no broadcast
