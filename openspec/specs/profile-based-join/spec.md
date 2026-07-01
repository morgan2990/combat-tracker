# Spec: Profile-Based Join

## Purpose

Defines how the join flow integrates with persistent player profiles: requiring a profile lookup before opening a WebSocket connection, passing max_hp via the connection URL, auto-loading companions from the profile, and propagating shared initiative when the player sets their initiative value.

## Requirements

### Requirement: A user must be logged in and select one of their own PCs to join a room

A user SHALL only be able to join a room as a player if they are authenticated and select one of their own PCs. The Dashboard SHALL list the user's PCs directly (from `GET /api/me`) — no name lookup or "Find my character" step is needed, since the system already knows which PCs belong to the logged-in user.

#### Scenario: Logged-in user selects a PC to join with
- **WHEN** an authenticated user with at least one PC selects a PC from their Dashboard's "My Characters" list and enters a room code
- **THEN** the frontend SHALL proceed directly to opening a WebSocket connection using that PC's `id` — no profile-lookup round trip is needed first

#### Scenario: User with no PCs is directed to create one
- **WHEN** an authenticated user with zero PCs attempts to join a room as a player
- **THEN** the frontend SHALL direct them to create a character first (`/characters/new`) and SHALL NOT allow the join form to be submitted

#### Scenario: Joining as a player requires authentication
- **WHEN** an unauthenticated visitor attempts to reach the player-join flow
- **THEN** the frontend SHALL redirect to the login/signup screen

### Requirement: Player joins room with stats resolved server-side from the owned PC

Once a PC is selected, the player SHALL enter their initiative and submit the join form. The frontend SHALL open the WebSocket connection with `pc_id` in the URL; the server SHALL resolve `max_hp` and `name` by looking up that PC's MongoDB document (verifying ownership), rather than trusting any client-supplied stat values.

#### Scenario: Player submits the join form after selecting a PC
- **WHEN** a player has selected a PC and entered an initiative value, then clicks "Join Room"
- **THEN** the frontend SHALL open a WebSocket connection to `/ws?room_id=X&role=player&pc_id=Y`

#### Scenario: Server creates entity with PC's stored stats on setup_character
- **WHEN** a connected player-role client sends `{ "type": "setup_character", "initiative": M }`
- **THEN** the server SHALL create the player entity using the `max_hp` and `name` from the PC document resolved at connection time (via `pc_id`, ownership-checked), and the provided initiative, with `current_hp` equal to `max_hp`

### Requirement: Companions auto-load server-side from the PC's stored companions on join

When the player entity is created, the server SHALL itself look up all companion documents whose `parent_pc_id` matches the joining PC and create a corresponding companion entity for each — the client SHALL NOT need to send individual `add_companion` messages for this initial load. Companions load with their saved `max_hp` and `shares_initiative` flag; initiative is `null` initially.

#### Scenario: Server creates companion entities from the PC's stored companions
- **WHEN** a player's `setup_character` succeeds and their PC has one or more companion documents
- **THEN** the server SHALL create a companion entity for each, without requiring any further client messages

#### Scenario: Companions load with null initiative
- **WHEN** a companion entity is server-instantiated from a PC's stored companions
- **THEN** the server SHALL create the companion entity with `initiative: null`

#### Scenario: PC with no companions joins normally
- **WHEN** a player's PC has zero linked companion documents
- **THEN** the server SHALL create only the player entity, with no companion entities added

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
