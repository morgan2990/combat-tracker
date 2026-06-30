## ADDED Requirements

### Requirement: Player updates their own entity's HP and conditions
A player SHALL be able to send an `update_entity` action to modify the `current_hp`, `temp_hp`, and `conditions` of an entity they own. The server SHALL validate ownership before applying any change.

#### Scenario: Player updates own HP
- **WHEN** a player sends `{ "type": "update_entity", "entity_id": "...", "current_hp": N, "temp_hp": M, "conditions": [...] }` where the entity's `session_id` matches the sender's session
- **THEN** the server SHALL apply the updated values to the entity and broadcast the full updated `RoomState` to all connected clients

#### Scenario: Player updates own conditions
- **WHEN** a player sends `update_entity` with a `conditions` array containing a subset of the predefined condition list
- **THEN** the server SHALL replace the entity's `conditions` array with the provided value and broadcast the updated state

#### Scenario: Unauthorized modification rejected
- **WHEN** a player sends `update_entity` with an `entity_id` whose `session_id` does not match the sender's session AND whose `owner_id` does not match the sender's entity ID
- **THEN** the server SHALL ignore the message, make no state change, and send no broadcast

#### Scenario: HP clamped to valid range
- **WHEN** a player sends `update_entity` with `current_hp` greater than `max_hp`
- **THEN** the server SHALL clamp `current_hp` to `max_hp` before applying the update

#### Scenario: Modification broadcast to all clients in real-time
- **WHEN** a valid `update_entity` action is processed
- **THEN** the server SHALL broadcast the updated `RoomState` to every connected client in the room within the same request cycle

### Requirement: Initiative tracker displays all combatants ordered by initiative
The player view SHALL display the full entity list sorted from highest to lowest initiative, with a clear visual indicator on the currently active entity.

#### Scenario: Initiative list rendered in descending order
- **WHEN** a player-role client renders the initiative tracker
- **THEN** entities SHALL be displayed in the order received from the server (which is already sorted descending by initiative)

#### Scenario: Active turn highlighted
- **WHEN** `is_started` is true and `active_index` points to an entity
- **THEN** the client SHALL render a distinct visual indicator (e.g., highlight or arrow) on that entity

#### Scenario: Fog of war applied to creatures
- **WHEN** a player-role client renders an entity with `type: "creature"`
- **THEN** the client SHALL NOT display exact `current_hp` or `max_hp`; it SHALL display a qualitative label (Healthy / Hurt / Injured / Dying / Dead) derived from the HP ratio

#### Scenario: Full HP shown for party members
- **WHEN** a player-role client renders an entity with `type: "player"` or `type: "companion"`
- **THEN** the client SHALL display exact `current_hp` / `max_hp` and all active conditions
