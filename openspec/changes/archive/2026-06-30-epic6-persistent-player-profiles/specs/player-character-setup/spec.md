## MODIFIED Requirements

### Requirement: Player configures their character after connecting
After a WebSocket connection is established, a new player (one without an existing entity in the room) SHALL be presented with a setup prompt to confirm their Max HP (pre-loaded from their profile, read-only) and enter their initiative. The player SHALL then send `setup_character` to create their entity, followed by `set_initiative` to set their initiative value.

#### Scenario: New player submits setup_character
- **WHEN** a player without an existing entity sends `{ "type": "setup_character" }` over their WebSocket connection
- **THEN** the server SHALL create a new entity with `type: "player"`, the player's name, `max_hp` from the session's stored profile value (passed via WS query param), `current_hp` equal to `max_hp`, and `initiative: null`

#### Scenario: Entity appears sorted in tracker after setup
- **WHEN** a player's entity is created via `setup_character`
- **THEN** the server SHALL insert the entity into `State.Entities`, re-sort the slice descending by initiative (null values sort last), and broadcast the updated `RoomState` to all connected clients

#### Scenario: Setup rejected when max_hp is missing from session
- **WHEN** a player sends `setup_character` but no `max_hp` was stored on their session (i.e., they connected without a valid profile query param)
- **THEN** the server SHALL reject the message and send no broadcast

## ADDED Requirements

### Requirement: Player sets initiative as a separate step after joining
After creating their entity via `setup_character`, a player SHALL send a `set_initiative` message to assign their initiative value. This is a distinct step from entity creation, allowing companions to be loaded first and shared-initiative propagation to occur in one action.

#### Scenario: Player sends set_initiative
- **WHEN** a player with an existing entity sends `{ "type": "set_initiative", "initiative": M }`
- **THEN** the server SHALL update the entity's `initiative` field to M and broadcast the updated `RoomState`

#### Scenario: set_initiative rejected before character is set up
- **WHEN** a player sends `set_initiative` before completing `setup_character`
- **THEN** the server SHALL ignore the message and send no broadcast
