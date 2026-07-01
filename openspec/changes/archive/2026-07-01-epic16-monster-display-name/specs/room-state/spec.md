## MODIFIED Requirements

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
- **THEN** the player view SHALL immediately render all entities, including previously-hidden creatures, with no transition delay or animation

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
