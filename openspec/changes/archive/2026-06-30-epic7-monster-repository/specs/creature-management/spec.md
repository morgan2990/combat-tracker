## MODIFIED Requirements

### Requirement: DM can add ephemeral creature entities to the room
The DM SHALL be able to add creature-type entities at any time, including during active combat. Creatures added mid-combat are sorted in by initiative with the active entity position preserved. The DM MAY include a `quantity` field (integer, default 1) to add multiple identical creatures in one action; when `quantity` is greater than 1, each creature SHALL be named with an auto-number suffix (e.g. "Goblin 1", "Goblin 2", "Goblin 3"). The DM MAY include `source_type`, `reference_url`, and `pdf_object_key` fields to associate a statblock reference with each created entity; all created entities from one `add_creature` message SHALL share the same statblock reference.

#### Scenario: DM adds a creature before combat
- **WHEN** a DM-role client sends `{ "type": "add_creature", "name": "Goblin", "max_hp": 14, "initiative": 11 }` and `is_started` is false
- **THEN** the server SHALL create a new entity with `type: "creature"`, the provided fields, `current_hp` equal to `max_hp`, `dead: false`, sort `State.Entities` descending by initiative, and broadcast the updated `RoomState`

#### Scenario: DM adds a creature mid-combat
- **WHEN** a DM-role client sends `add_creature` and `is_started` is true
- **THEN** the server SHALL create the creature entity, re-sort `State.Entities` descending by initiative, update `active_index` to continue pointing at the same entity that was active before the sort, and broadcast the updated `RoomState`

#### Scenario: Non-DM cannot add creatures
- **WHEN** a player-role client sends `add_creature`
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: DM adds multiple creatures using quantity
- **WHEN** a DM-role client sends `add_creature` with `name: "Goblin"`, `max_hp: 7`, and `quantity: 3`
- **THEN** the server SHALL create three entities named "Goblin 1", "Goblin 2", "Goblin 3", each with `current_hp: 7` and independent state, re-sort all entities by initiative, and broadcast a single updated `RoomState` after all three are inserted

#### Scenario: DM adds a creature with a statblock reference
- **WHEN** a DM-role client sends `add_creature` with `source_type: "url"` and `reference_url: "https://example.com/goblin.webp"`
- **THEN** the server SHALL set `source_type` and `reference_url` on the created entity and include those fields in the broadcast `RoomState`

#### Scenario: DM adds multiple creatures with a statblock reference
- **WHEN** a DM-role client sends `add_creature` with `quantity: 2`, `source_type: "url"`, and `reference_url: "https://example.com/goblin.webp"`
- **THEN** the server SHALL set the same `source_type` and `reference_url` on each of the two created entities
