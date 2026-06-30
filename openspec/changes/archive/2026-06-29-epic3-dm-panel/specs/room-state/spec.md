## MODIFIED Requirements

### Requirement: Room state has a defined data model
The system SHALL represent each room's combat state using the following structure:

- `room_id` (string): the room's unique identifier
- `is_started` (bool): whether combat has been started by the DM
- `round` (int): the current round number; 0 before combat starts, 1 when combat begins, incremented each time the turn order wraps
- `active_index` (int): index into the entities slice for the currently active turn; always refers to the current sorted order and is preserved by ID across re-sorts
- `entities` (array of Entity): maintained in descending initiative order at all times; re-sorts triggered by the DM always preserve the active entity's position by entity ID

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
- `dead` (bool): true when the DM has marked the entity as dead; dead entities remain in the list and are rendered greyed-out on all clients

#### Scenario: Entity created with required fields
- **WHEN** any entity is added to a room
- **THEN** the entity SHALL have a server-generated UUID `id`, a non-empty `name`, a valid `type`, numeric HP fields initialized to their given values, and `dead` initialized to `false`

#### Scenario: Entities sorted after any addition or DM initiative change
- **WHEN** any entity is added to `State.Entities`, or a DM changes an entity's initiative
- **THEN** the server SHALL sort `State.Entities` in descending order by `initiative` using a stable sort; if `is_started` is true the server SHALL also update `active_index` to the new position of the entity that was active before the sort

#### Scenario: Active entity tracked by ID across re-sorts
- **WHEN** a re-sort occurs while `is_started` is true
- **THEN** the server SHALL record the `id` of the entity at `active_index` before sorting, perform the sort, then scan the sorted slice to find that entity's new index and set `active_index` accordingly
