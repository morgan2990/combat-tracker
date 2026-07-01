## MODIFIED Requirements

### Requirement: A user can create or update their own PC

An authenticated user SHALL be able to create a PC by sending a `POST /api/pcs` request with a `name` and `max_hp`; the server SHALL generate an `id` and set `owner_user_id` from the session. The user SHALL be able to update a PC they own via `PUT /api/pcs/:id`. `name` is a display label only — it is NOT required to be unique, globally or per-user.

#### Scenario: New PC is created
- **WHEN** an authenticated client sends `POST /api/pcs` with `{ "name": "Aria", "max_hp": 16 }`
- **THEN** the server SHALL insert a document into the `pcs` MongoDB collection with a generated `id`, `owner_user_id` set to the requesting user, and return HTTP 200 with the saved PC

#### Scenario: Owner updates their own PC
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` for a PC whose `owner_user_id` matches the requesting user
- **THEN** the server SHALL overwrite the existing document's editable fields and return HTTP 200

#### Scenario: Update rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` for a PC whose `owner_user_id` does not match the requesting user
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: PC creation rejected with invalid payload
- **WHEN** a client sends `POST /api/pcs` with `max_hp` ≤ 0 or an empty `name`
- **THEN** the server SHALL return HTTP 400 and make no change to the collection

#### Scenario: PC creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/pcs`
- **THEN** the server SHALL respond with HTTP 401 and make no change

### Requirement: A user can create companion profiles linked to their own PC

A companion profile SHALL be saved with `type: "companion"`, a `parent_pc_id` linking it to a PC, and a `shares_initiative` boolean. The requesting user MUST own the PC referenced by `parent_pc_id`.

#### Scenario: Companion profile is created with shared initiative
- **WHEN** an authenticated client sends `POST /api/pcs/:parent_id/companions` with `{ "name": "Rex", "max_hp": 8, "shares_initiative": true }` and the client owns the PC at `:parent_id`
- **THEN** the server SHALL create the companion document linked via `parent_pc_id` and return HTTP 200

#### Scenario: Companion profile is created without shared initiative
- **WHEN** an authenticated client sends the same request with `"shares_initiative": false`
- **THEN** the server SHALL save the document with `shares_initiative: false` and return HTTP 200

#### Scenario: Companion creation rejected for a PC the user does not own
- **WHEN** an authenticated client attempts to create a companion under `parent_pc_id` belonging to a different user
- **THEN** the server SHALL respond with HTTP 403 and make no change

### Requirement: A user can fetch their own PCs and companions

The system SHALL provide `GET /api/pcs/:id`, scoped to the requesting user's ownership, returning the PC and an array of all companion documents whose `parent_pc_id` matches. A user's full PC list (without per-PC companion detail) SHALL also be available via `GET /api/me` (see `user-accounts`).

#### Scenario: Own PC fetched with companions
- **WHEN** an authenticated client sends `GET /api/pcs/:id` for a PC they own that has linked companions
- **THEN** the server SHALL return HTTP 200 with `{ "pc": { ... }, "companions": [ ... ] }`

#### Scenario: Fetch rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `GET /api/pcs/:id` for a PC owned by a different user
- **THEN** the server SHALL respond with HTTP 403 or HTTP 404 (not revealing whether the PC exists)

#### Scenario: PC not found
- **WHEN** an authenticated client sends `GET /api/pcs/:id` for an `id` that does not exist
- **THEN** the server SHALL respond with HTTP 404

### Requirement: Player can refresh their entity from profile while in an active room

A player SHALL be able to send a `refresh_from_profile` WebSocket message to re-fetch their PC (identified by the `pc_id` established at connection time) from MongoDB and update their entity's `max_hp` (and all their companions' `max_hp`) in the current room only. Other rooms are unaffected.

#### Scenario: Player refreshes and max_hp increases
- **WHEN** a player sends `{ "type": "refresh_from_profile" }` and their PC's stored `max_hp` is higher than their current entity's
- **THEN** the server SHALL update the entity's `max_hp` to the new value and broadcast the updated `RoomState`

#### Scenario: Player refreshes and max_hp decreases — current_hp is capped
- **WHEN** a player sends `refresh_from_profile` and the new `max_hp` is lower than the entity's `current_hp`
- **THEN** the server SHALL set `max_hp` to the new value AND reduce `current_hp` to match the new `max_hp`, then broadcast

#### Scenario: Refresh also updates companion max_hp values
- **WHEN** a player sends `refresh_from_profile` and the player has companion entities in the room
- **THEN** the server SHALL also update each companion's `max_hp` (looked up via `parent_pc_id`) and cap `current_hp` if needed, then broadcast

#### Scenario: Refresh fails when the PC no longer exists
- **WHEN** a player sends `refresh_from_profile` but their `pc_id` no longer resolves to a PC document in MongoDB
- **THEN** the server SHALL send no broadcast and make no state change
