# Spec: Player Profile Management

## Purpose

Defines how players create, update, and fetch persistent character and companion profiles stored in MongoDB, and how those profiles can be used to refresh entity state in an active room.

## Requirements

### Requirement: Player can create or update a character profile
A player SHALL be able to save a character profile by sending a `POST /api/entities` request. If a profile with the same name already exists it SHALL be overwritten (upsert by name). The `name` field is the unique key across all profiles globally.

#### Scenario: New player profile is created
- **WHEN** a client sends `POST /api/entities` with `{ "name": "Aria", "type": "player", "max_hp": 16 }`
- **THEN** the server SHALL insert the document into the `entities` MongoDB collection and return HTTP 200 with the saved profile

#### Scenario: Existing profile is updated (upsert)
- **WHEN** a client sends `POST /api/entities` with a `name` that already exists in the collection
- **THEN** the server SHALL overwrite the existing document with the new field values and return HTTP 200

#### Scenario: Profile creation rejected with invalid payload
- **WHEN** a client sends `POST /api/entities` with `max_hp` ≤ 0, an empty `name`, or a `type` other than `"player"` or `"companion"`
- **THEN** the server SHALL return HTTP 400 and make no change to the collection

### Requirement: Player can create companion profiles linked to a character
A companion profile SHALL be saved with `type: "companion"`, a `parent_pc_name` linking it to a player profile, and a `shares_initiative` boolean. Companion profiles are upserted by name, independent of the linked player profile.

#### Scenario: Companion profile is created with shared initiative
- **WHEN** a client sends `POST /api/entities` with `{ "name": "Rex", "type": "companion", "max_hp": 8, "parent_pc_name": "Aria", "shares_initiative": true }`
- **THEN** the server SHALL upsert the document into the `entities` collection and return HTTP 200

#### Scenario: Companion profile is created without shared initiative
- **WHEN** a client sends `POST /api/entities` with `"shares_initiative": false`
- **THEN** the server SHALL save the document with `shares_initiative: false` and return HTTP 200

#### Scenario: Companion profile missing parent_pc_name is rejected
- **WHEN** a client sends `POST /api/entities` with `type: "companion"` and no `parent_pc_name` (or empty string)
- **THEN** the server SHALL return HTTP 400 and make no change

### Requirement: Player can fetch a profile and its companions by name
`GET /api/entities/:name` SHALL return the player profile matching `name` and an array of all companion documents whose `parent_pc_name` matches that name.

#### Scenario: Profile found with companions
- **WHEN** a client sends `GET /api/entities/Aria` and "Aria" exists in the collection with linked companions
- **THEN** the server SHALL return HTTP 200 with `{ "profile": { ... }, "companions": [ ... ] }`

#### Scenario: Profile found with no companions
- **WHEN** a client sends `GET /api/entities/Aria` and "Aria" exists but has no linked companions
- **THEN** the server SHALL return HTTP 200 with `{ "profile": { ... }, "companions": [] }`

#### Scenario: Profile not found
- **WHEN** a client sends `GET /api/entities/Unknown` and no document with that name exists
- **THEN** the server SHALL return HTTP 404

### Requirement: Player can refresh their entity from profile while in an active room
A player SHALL be able to send a `refresh_from_profile` WebSocket message to re-fetch their profile from MongoDB and update their entity's `max_hp` (and all their companions' `max_hp`) in the current room only. Other rooms are unaffected.

#### Scenario: Player refreshes and max_hp increases
- **WHEN** a player sends `{ "type": "refresh_from_profile" }` and their MongoDB profile has a higher `max_hp` than their current entity
- **THEN** the server SHALL update the entity's `max_hp` to the new value and broadcast the updated `RoomState`

#### Scenario: Player refreshes and max_hp decreases — current_hp is capped
- **WHEN** a player sends `refresh_from_profile` and the new `max_hp` is lower than the entity's `current_hp`
- **THEN** the server SHALL set `max_hp` to the new value AND reduce `current_hp` to match the new `max_hp`, then broadcast

#### Scenario: Refresh also updates companion max_hp values
- **WHEN** a player sends `refresh_from_profile` and the player has companion entities in the room
- **THEN** the server SHALL also update each companion's `max_hp` (and cap `current_hp` if needed) from the companion's stored profile, then broadcast

#### Scenario: Refresh fails when profile not found
- **WHEN** a player sends `refresh_from_profile` but their name no longer exists in MongoDB
- **THEN** the server SHALL send no broadcast and make no state change
