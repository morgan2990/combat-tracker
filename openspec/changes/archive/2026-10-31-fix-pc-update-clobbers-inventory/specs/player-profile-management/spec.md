## MODIFIED Requirements

### Requirement: A user can create or update their own PC

An authenticated user SHALL be able to create a PC by sending a `POST /api/pcs` request with a `name` and `max_hp`; the server SHALL generate an `id`, set `owner_user_id` from the session, initialize `items` as an empty array, and initialize `currency` with all five denominations (`pp`, `gp`, `ep`, `sp`, `cp`) set to `0`. The user SHALL be able to update a document they own (a top-level PC or a companion — both are rows in the same collection, distinguished by `type`) via `PUT /api/pcs/{id}`. `name` and `max_hp` SHALL be required on every update. `items`, `currency`, and `shares_initiative` SHALL be updated only when present in the request; an omitted or `null` field SHALL keep its stored value. `name` is a display label only — it is NOT required to be unique, globally or per-user.

#### Scenario: New PC is created
- **WHEN** an authenticated client sends `POST /api/pcs` with `{ "name": "Aria", "max_hp": 16 }`
- **THEN** the server SHALL insert a document into the `pcs` MongoDB collection with a generated `id`, `owner_user_id` set to the requesting user, `items: []`, `currency` fully zeroed, and return HTTP 200 with the saved PC

#### Scenario: Owner updates their own PC
- **WHEN** an authenticated client sends `PUT /api/pcs/{id}` for a PC whose `owner_user_id` matches the requesting user, with `name`, `max_hp`, `items`, and `currency`
- **THEN** the server SHALL overwrite those fields and return HTTP 200

#### Scenario: Update that omits inventory keeps it
- **WHEN** the owner of a PC with stored `items` and `currency` sends `PUT /api/pcs/{id}` with only `{ "name": "Strider", "max_hp": 50 }`
- **THEN** the server SHALL update `name` and `max_hp`, SHALL leave `items`, `currency`, and `shares_initiative` unchanged, and return HTTP 200

#### Scenario: An explicit empty item list clears the items
- **WHEN** the owner of a PC with stored `items` sends `PUT /api/pcs/{id}` with `"items": []`
- **THEN** the server SHALL store `items` as an empty array

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
