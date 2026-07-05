## MODIFIED Requirements

### Requirement: A user can create or update their own PC

An authenticated user SHALL be able to create a PC by sending a `POST /api/pcs` request with a `name` and `max_hp`; the server SHALL generate an `id`, set `owner_user_id` from the session, initialize `items` as an empty array, and initialize `currency` with all five denominations (`pp`, `gp`, `ep`, `sp`, `cp`) set to `0`. The user SHALL be able to update a document they own (a top-level PC or a companion — both are rows in the same collection, distinguished by `type`) via `PUT /api/pcs/{id}`, including its `items`, `currency`, and `shares_initiative` fields. `name` is a display label only — it is NOT required to be unique, globally or per-user.

#### Scenario: New PC is created
- **WHEN** an authenticated client sends `POST /api/pcs` with `{ "name": "Aria", "max_hp": 16 }`
- **THEN** the server SHALL insert a document into the `pcs` MongoDB collection with a generated `id`, `owner_user_id` set to the requesting user, `items: []`, `currency` fully zeroed, and return HTTP 200 with the saved PC

#### Scenario: Owner updates their own PC
- **WHEN** an authenticated client sends `PUT /api/pcs/{id}` for a PC whose `owner_user_id` matches the requesting user
- **THEN** the server SHALL overwrite the existing document's editable fields, including `items` and `currency`, and return HTTP 200

#### Scenario: Owner updates a companion's editable fields
- **WHEN** an authenticated client sends `PUT /api/pcs/{id}` for a companion document (`type: "companion"`) whose `owner_user_id` matches the requesting user, with `{ "name": ..., "max_hp": ..., "shares_initiative": ... }`
- **THEN** the server SHALL persist all three fields, including `shares_initiative`, and return HTTP 200

#### Scenario: Update rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `PUT /api/pcs/{id}` for a PC whose `owner_user_id` does not match the requesting user
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: PC creation rejected with invalid payload
- **WHEN** a client sends `POST /api/pcs` with `max_hp` ≤ 0 or an empty `name`
- **THEN** the server SHALL return HTTP 400 and make no change to the collection

#### Scenario: PC creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/pcs`
- **THEN** the server SHALL respond with HTTP 401 and make no change

### Requirement: Player can refresh their entity from profile while in an active room
A player SHALL be able to send a `refresh_from_profile` WebSocket message to re-fetch their PC (identified by the `pc_id` established at connection time) from MongoDB and update their entity's `max_hp` (and all their companions' `max_hp`, `name`, and `shares_initiative`) in the current room only. Other rooms are unaffected.

#### Scenario: Player refreshes and max_hp increases
- **WHEN** a player sends `{ "type": "refresh_from_profile" }` and their PC's stored `max_hp` is higher than their current entity's
- **THEN** the server SHALL update the entity's `max_hp` to the new value and broadcast the updated `RoomState`

#### Scenario: Player refreshes and max_hp decreases — current_hp is capped
- **WHEN** a player sends `refresh_from_profile` and the new `max_hp` is lower than the entity's `current_hp`
- **THEN** the server SHALL set `max_hp` to the new value AND reduce `current_hp` to match the new `max_hp`, then broadcast

#### Scenario: Refresh also updates companion max_hp values
- **WHEN** a player sends `refresh_from_profile` and the player has companion entities in the room
- **THEN** the server SHALL also update each companion's `max_hp` (looked up via `parent_pc_id`) and cap `current_hp` if needed, then broadcast

#### Scenario: Refresh also updates companion name and shares_initiative
- **WHEN** a player sends `refresh_from_profile` and a companion's stored `name` or `shares_initiative` differs from its live room entity
- **THEN** the server SHALL update that companion entity's `Name` and `SharesInitiative` to match the stored profile values, then broadcast

#### Scenario: Refresh fails when the PC no longer exists
- **WHEN** a player sends `refresh_from_profile` but their `pc_id` no longer resolves to a PC document in MongoDB
- **THEN** the server SHALL send no broadcast and make no state change

## ADDED Requirements

### Requirement: A user can delete their own PC or companion
The system SHALL provide `DELETE /api/pcs/{id}`, available to the authenticated owner of the document (`owner_user_id` matches the requester), for either a top-level PC or a companion document. Deleting a `type: "pc"` document SHALL cascade to delete every companion document whose `parent_pc_id` matches it. Deletion SHALL be unconditional — it SHALL NOT be blocked by the document being currently instantiated as a live entity in an active room.

#### Scenario: Owner deletes their own companion
- **WHEN** the owner of a companion document sends `DELETE /api/pcs/{id}` for that companion's `id`
- **THEN** the server SHALL delete the document and return HTTP 200 (or 204)

#### Scenario: Owner deletes a PC, cascading to its companions
- **WHEN** the owner of a PC sends `DELETE /api/pcs/{id}` for that PC's `id`, and one or more companion documents have `parent_pc_id` equal to that `id`
- **THEN** the server SHALL delete the PC document and all matching companion documents in the same operation

#### Scenario: Delete rejected for a document owned by someone else
- **WHEN** an authenticated client sends `DELETE /api/pcs/{id}` for a PC or companion whose `owner_user_id` does not match the requesting user
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Delete rejected when not authenticated
- **WHEN** a client without a valid session sends `DELETE /api/pcs/{id}`
- **THEN** the server SHALL respond with HTTP 401 and make no change

#### Scenario: Delete succeeds even when the document is live in an active room
- **WHEN** a PC or companion document being deleted currently has a corresponding live `Entity` in an active room's in-memory state
- **THEN** the server SHALL still delete the document; the live entity SHALL remain in the room unchanged (no cleanup, no broadcast) until that room ends or its state is otherwise refreshed

#### Scenario: Not found
- **WHEN** an authenticated client sends `DELETE /api/pcs/{id}` for an `id` that does not exist
- **THEN** the server SHALL respond with HTTP 404
