## MODIFIED Requirements

### Requirement: Room state has a defined data model
The system SHALL represent each room's combat state using the following structure:

- `room_id` (string): the room's unique identifier
- `edition` (string): the ruleset for this room — `"5e"` or `"5.5e"`; set at creation, immutable thereafter
- `is_started` (bool): whether combat has been started by the DM
- `round` (int): the current round number; 0 before combat starts, 1 when combat begins, incremented each time the turn order wraps
- `active_index` (int): index into the entities slice for the currently active turn; always refers to the current sorted order and is preserved by ID across re-sorts
- `entities` (array of Entity): maintained in descending initiative order at all times; re-sorts triggered by the DM always preserve the active entity's position by entity ID

Each Entity SHALL have:
- `id` (string): UUID, assigned at creation
- `name` (string): display name
- `type` (string): one of `pc`, `creature`, `companion`, `lair_action`
- `owner_id` (string): for companions, the `id` of the owning PC entity; empty otherwise
- `session_id` (string): the WS connection identifier for PC-type entities; empty for creatures
- `max_hp` (int)
- `current_hp` (int)
- `temp_hp` (int)
- `initiative` (int)
- `conditions` (array of strings): e.g., `["Prone", "Stunned"]`
- `dead` (bool): true when the DM has marked the entity as dead; dead entities remain in the list and are rendered greyed-out on all clients

#### Scenario: Room state includes edition after creation
- **WHEN** a room is created with `edition: "5.5e"`
- **THEN** the `RoomState` broadcast to all clients SHALL include `"edition": "5.5e"`

#### Scenario: Edition is present in every broadcast
- **WHEN** any mutation triggers a state broadcast (entity added, turn advanced, etc.)
- **THEN** the broadcast SHALL include the `edition` field with the value set at room creation

#### Scenario: Entity created with required fields
- **WHEN** any entity is added to a room
- **THEN** the entity SHALL have a server-generated UUID `id`, a non-empty `name`, a valid `type`, numeric HP fields initialized to their given values, and `dead` initialized to `false`

#### Scenario: Entities sorted after any addition or DM initiative change
- **WHEN** any entity is added to `State.Entities`, or a DM changes an entity's initiative
- **THEN** the server SHALL sort `State.Entities` in descending order by `initiative` using a stable sort; if `is_started` is true the server SHALL also update `active_index` to the new position of the entity that was active before the sort; entities of `type: "lair_action"` are additionally subject to the tie-break rule defined in the `lair-actions` capability

#### Scenario: Active entity tracked by ID across re-sorts
- **WHEN** a re-sort occurs while `is_started` is true
- **THEN** the server SHALL record the `id` of the entity at `active_index` before sorting, perform the sort, then scan the sorted slice to find that entity's new index and set `active_index` accordingly

### Requirement: Frontend is responsible for role-based data presentation
The client application SHALL determine what data to display based on the connected user's role. The server sends identical full state to all clients.

#### Scenario: Player does not see exact creature HP
- **WHEN** a player-role client receives a `RoomState` broadcast
- **THEN** the player view SHALL NOT render exact `current_hp` or `max_hp` for entities with `type: "creature"`; it SHALL render a qualitative label instead (e.g., Healthy, Injured, Dying)

#### Scenario: DM sees full data for all entities
- **WHEN** a DM-role client receives a `RoomState` broadcast
- **THEN** the DM view SHALL render exact HP and all fields for every entity regardless of type

#### Scenario: Player does not see staged creatures before combat starts
- **WHEN** a player-role client receives a `RoomState` broadcast with `is_started: false`
- **THEN** the player view SHALL exclude all entities with `type: "creature"` from the rendered initiative ladder; entities with `type: "pc"` or `type: "companion"` SHALL still render

#### Scenario: Player sees the full list the instant combat starts
- **WHEN** a player-role client receives a `RoomState` broadcast where `is_started` has transitioned from `false` to `true`
- **THEN** the player view SHALL immediately render all entities, including previously-hidden creatures, with no transition delay or animation (subject to the `is_hidden` filter below, which applies independently of `is_started`)

#### Scenario: Player sees a staging placeholder when creatures are hidden
- **WHEN** a player-role client has `is_started: false` and the room's entity list contains at least one entity with `type: "creature"` but no visible (non-creature) entities
- **THEN** the player view SHALL render a staging-specific placeholder message distinct from the empty-room message shown when the entity list is entirely empty

#### Scenario: DM sees staged creatures regardless of combat state
- **WHEN** a DM-role client receives a `RoomState` broadcast with `is_started: false`
- **THEN** the DM view SHALL render all entities, including creatures, exactly as it does when `is_started` is `true`

#### Scenario: DM sees both names when an alias is set
- **WHEN** a DM-role client renders an entity with a non-empty `display_name`
- **THEN** the DM view SHALL render `"{display_name} ({name})"` (e.g. `"Guard 1 (Goblin 1)"`)

#### Scenario: DM sees only the base name when no alias is set
- **WHEN** a DM-role client renders an entity with an empty `display_name`
- **THEN** the DM view SHALL render `name` alone, with no parenthetical

#### Scenario: Player sees only the alias when one is set
- **WHEN** a player-role client renders an entity with a non-empty `display_name`
- **THEN** the player view SHALL render `display_name` only; the entity's `name` field SHALL NOT appear anywhere in the rendered row

#### Scenario: Player falls back to the base name when no alias is set
- **WHEN** a player-role client renders an entity with an empty `display_name`
- **THEN** the player view SHALL render `name`, exactly as it does today for entities with no alias concept

#### Scenario: Player does not see entities marked hidden
- **WHEN** a player-role client receives a `RoomState` broadcast containing an entity with `is_hidden: true`
- **THEN** the player view SHALL completely omit that entity from the rendered initiative ladder — no name, HP, condition, or turn-order slot SHALL appear — regardless of `is_started`

#### Scenario: DM always sees hidden entities with distinct styling
- **WHEN** a DM-role client renders an entity with `is_hidden: true`
- **THEN** the DM view SHALL render that entity (never omit it) with a visually distinct treatment (e.g. reduced opacity) so the DM can tell at a glance which entities are currently hidden from players

#### Scenario: Hidden and pre-combat masking compose without conflict
- **WHEN** a player-role client evaluates an entity that is both `type: "creature"` with `is_started: false`, and separately has `is_hidden: true`
- **THEN** the entity remains omitted from the player view under either condition; toggling `is_hidden` to `false` does not reveal the entity while `is_started` is still `false`, and starting combat does not reveal an entity that still has `is_hidden: true`

#### Scenario: Lair actions default to hidden, independent of combat state
- **WHEN** a `lair_action` entity is created via `add_lair_action`
- **THEN** it is created with `is_hidden: true`; because its `type` is not `"creature"`, the pre-combat blanket-hide from the "Player does not see staged creatures before combat starts" scenario does not apply to it, and `is_hidden` alone governs whether a player-role client can see it, regardless of `is_started`

#### Scenario: DM reveals a lair action to players
- **WHEN** the DM toggles `is_hidden` to `false` on a `lair_action` entity via `toggle_entity_visibility`
- **THEN** player-role clients SHALL render that entity on their next broadcast, subject to the same row-rendering rules (no HP/status UI) defined in the `lair-actions` capability
