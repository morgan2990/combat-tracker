## MODIFIED Requirements

### Requirement: Room state has a defined data model
The system SHALL represent each room's combat state using the following structure:

- `room_id` (string): the room's unique identifier
- `is_started` (bool): whether combat has been started by the DM
- `round` (int): the current round number, starting at 0
- `active_index` (int): index into the entities slice for the currently active turn; always refers to the sorted order
- `entities` (array of Entity): **maintained in descending initiative order at all times while `is_started` is false; order is frozen once `is_started` becomes true**

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

#### Scenario: Entities sorted after addition
- **WHEN** any entity is added to `State.Entities` and `is_started` is false
- **THEN** the server SHALL sort `State.Entities` in descending order by `initiative` using a stable sort before broadcasting

#### Scenario: Order frozen when combat starts
- **WHEN** `is_started` becomes true
- **THEN** the server SHALL NOT re-sort `State.Entities` on any subsequent mutation; `active_index` continues to refer to the fixed order
