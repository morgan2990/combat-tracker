## MODIFIED Requirements

### Requirement: Player can add a companion or summoned creature
A player SHALL be able to add a companion entity linked to their own entity. The companion appears in the initiative tracker and is treated as a separate combatant. The `add_companion` message now includes a `shares_initiative` boolean field and no longer requires an `initiative` value (it is nullable).

#### Scenario: Player adds a companion with initiative
- **WHEN** a player sends `{ "type": "add_companion", "name": "Wolf", "max_hp": 18, "initiative": 12, "shares_initiative": false }` over their WebSocket connection
- **THEN** the server SHALL create a new entity with `type: "companion"`, `owner_id` set to the player's entity ID, the provided `name`, `max_hp`, `initiative`, `shares_initiative`, and `current_hp` equal to `max_hp`

#### Scenario: Player adds a companion with null initiative (profile auto-load)
- **WHEN** a player sends `add_companion` with no `initiative` field (or `initiative: null`)
- **THEN** the server SHALL create the companion entity with `initiative: null`

#### Scenario: Companion appears sorted in tracker
- **WHEN** a companion entity is created
- **THEN** the server SHALL insert it into `State.Entities`, re-sort descending by initiative (null values sort last), and broadcast the updated `RoomState` to all connected clients

#### Scenario: Add companion rejected when player has no entity
- **WHEN** a player without an established entity (i.e., has not completed setup) sends `add_companion`
- **THEN** the server SHALL ignore the message

## ADDED Requirements

### Requirement: Companion with shared initiative copies owner's initiative automatically
When a player sends `set_initiative`, the server SHALL propagate that initiative value to all companion entities in the room that have `SharesInitiative: true` and whose `OwnerID` matches the player's entity ID.

#### Scenario: Shared companion receives initiative on owner set
- **WHEN** a player sends `{ "type": "set_initiative", "initiative": 14 }` and has a companion in the room with `shares_initiative: true`
- **THEN** the server SHALL set the companion's `initiative` to 14 alongside the player entity's initiative in the same operation, then broadcast

#### Scenario: Non-shared companion is not affected by owner's set_initiative
- **WHEN** a player sends `set_initiative` and has a companion in the room with `shares_initiative: false`
- **THEN** the server SHALL leave that companion's initiative unchanged
