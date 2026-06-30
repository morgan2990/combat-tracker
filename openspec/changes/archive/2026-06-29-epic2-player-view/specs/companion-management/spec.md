## ADDED Requirements

### Requirement: Player can add a companion or summoned creature
A player SHALL be able to add a companion entity linked to their own entity. The companion appears in the initiative tracker and is treated as a separate combatant.

#### Scenario: Player adds a companion
- **WHEN** a player sends `{ "type": "add_companion", "name": "Wolf", "max_hp": 18, "initiative": 12 }` over their WebSocket connection
- **THEN** the server SHALL create a new entity with `type: "companion"`, `owner_id` set to the player's entity ID, the provided `name`, `max_hp`, `initiative`, and `current_hp` equal to `max_hp`

#### Scenario: Companion appears sorted in tracker
- **WHEN** a companion entity is created
- **THEN** the server SHALL insert it into `State.Entities`, re-sort descending by initiative, and broadcast the updated `RoomState` to all connected clients

#### Scenario: Add companion rejected when player has no entity
- **WHEN** a player without an established entity (i.e., has not completed setup) sends `add_companion`
- **THEN** the server SHALL ignore the message

### Requirement: Companion owner has full edit permissions
The player who owns a companion SHALL be able to update its HP and conditions using the same `update_entity` action used for their own entity.

#### Scenario: Owner updates companion HP
- **WHEN** a player sends `update_entity` targeting a companion whose `owner_id` matches the player's entity ID
- **THEN** the server SHALL apply the update and broadcast the updated state

#### Scenario: Non-owner cannot edit companion
- **WHEN** a player sends `update_entity` targeting a companion whose `owner_id` does not match the player's entity ID
- **THEN** the server SHALL reject the action, make no state change, and send no broadcast

### Requirement: Companion entities persist after player disconnect
When a player's WebSocket connection closes, their companion entities SHALL remain in `State.Entities` until explicitly removed by the DM.

#### Scenario: Companion remains after player disconnect
- **WHEN** a player's connection closes (normally or abnormally)
- **THEN** any companion entities with `owner_id` equal to the disconnected player's entity ID SHALL remain in `State.Entities` and continue to appear in all clients' initiative trackers

#### Scenario: Reconnected player retakes companion ownership
- **WHEN** a player reconnects and their entity is re-linked (via the reconnection flow)
- **THEN** their companion entities (matched by `owner_id`) SHALL again be editable by the reconnected session
