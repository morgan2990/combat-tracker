# Spec: Profile-Based Join

## Purpose

Defines how the join flow integrates with persistent player profiles: requiring a profile lookup before opening a WebSocket connection, passing max_hp via the connection URL, auto-loading companions from the profile, and propagating shared initiative when the player sets their initiative value.

## Requirements

### Requirement: Player must have a saved profile to join a room
A player SHALL only be able to join a room if a matching profile exists in MongoDB. The join screen SHALL present a "Find my character" button that fetches the profile before the WebSocket connection is opened. If no profile is found the player cannot proceed.

#### Scenario: Player finds their profile and proceeds to join
- **WHEN** a player enters their character name and clicks "Find my character"
- **THEN** the frontend SHALL call `GET /api/entities/:name`; if a profile is returned, the frontend SHALL pre-populate the `max_hp` field (read-only) and reveal the initiative input field

#### Scenario: Profile not found — player is blocked
- **WHEN** a player clicks "Find my character" and the server returns HTTP 404
- **THEN** the frontend SHALL display an error message directing the player to create a profile at `/characters/new` and SHALL NOT allow the join form to be submitted

#### Scenario: Profile fetch fails due to server error
- **WHEN** the `GET /api/entities/:name` call returns a non-200 response other than 404
- **THEN** the frontend SHALL display a generic "Service unavailable" error and SHALL NOT allow the join form to be submitted

### Requirement: Player joins room with max_hp loaded from profile
Once a profile is found, the player SHALL enter their initiative and submit the join form. The frontend SHALL pass `max_hp` from the fetched profile to the server via the WebSocket connection URL so the server can create the entity without trusting client-submitted stats.

#### Scenario: Player submits the join form after profile fetch
- **WHEN** a player has a fetched profile and enters an initiative value and clicks "Join Room"
- **THEN** the frontend SHALL open a WebSocket connection to `/ws?room_id=X&name=Y&role=player&max_hp=N` where N is the profile's `max_hp`

#### Scenario: Server creates entity with profile max_hp on setup_character
- **WHEN** a connected player-role client sends `{ "type": "setup_character", "initiative": M }`
- **THEN** the server SHALL create the player entity using the `max_hp` stored on the client's session (from the WS query param) and the provided initiative, with `current_hp` equal to `max_hp`

### Requirement: Companions auto-load from profile on join
After the player entity is created, the frontend SHALL automatically send one `add_companion` message per companion in the fetched profile. Companions load with their saved `max_hp` and `shares_initiative` flag; initiative is `null` initially.

#### Scenario: Frontend auto-sends add_companion for each profile companion
- **WHEN** the fetched profile includes one or more companions
- **THEN** after `setup_character` succeeds, the frontend SHALL send `{ "type": "add_companion", "name": "...", "max_hp": N, "shares_initiative": true/false }` for each companion in order

#### Scenario: Companions load with null initiative
- **WHEN** a companion is auto-loaded via `add_companion` from a profile
- **THEN** the server SHALL create the companion entity with `initiative: null`

### Requirement: Shared initiative propagates when player sets their initiative
When a player sends `set_initiative`, the server SHALL automatically copy that initiative value to all companions in the room whose `SharesInitiative` flag is true and whose `OwnerID` matches the player's entity ID.

#### Scenario: Player sets initiative — shared companions update automatically
- **WHEN** a player sends `{ "type": "set_initiative", "initiative": 14 }` and has companions with `shares_initiative: true`
- **THEN** the server SHALL set the player entity's initiative to 14 AND set each sharing companion's initiative to 14, then broadcast

#### Scenario: Player sets initiative — non-shared companions unaffected
- **WHEN** a player sends `set_initiative` and has companions with `shares_initiative: false`
- **THEN** the server SHALL only update the player entity's initiative; non-sharing companions retain their existing (null or set) initiative value

#### Scenario: set_initiative rejected before character is set up
- **WHEN** a player sends `set_initiative` before completing `setup_character`
- **THEN** the server SHALL ignore the message and send no broadcast
