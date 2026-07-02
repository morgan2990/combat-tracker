## MODIFIED Requirements

### Requirement: A user can create or update their own PC

An authenticated user SHALL be able to create a PC by sending a `POST /api/pcs` request with a `name` and `max_hp`; the server SHALL generate an `id`, set `owner_user_id` from the session, initialize `items` as an empty array, and initialize `currency` with all five denominations (`pp`, `gp`, `ep`, `sp`, `cp`) set to `0`. The user SHALL be able to update a PC they own via `PUT /api/pcs/:id`, including its `items` and `currency` fields. `name` is a display label only — it is NOT required to be unique, globally or per-user.

#### Scenario: New PC is created
- **WHEN** an authenticated client sends `POST /api/pcs` with `{ "name": "Aria", "max_hp": 16 }`
- **THEN** the server SHALL insert a document into the `pcs` MongoDB collection with a generated `id`, `owner_user_id` set to the requesting user, `items: []`, `currency` fully zeroed, and return HTTP 200 with the saved PC

#### Scenario: Owner updates their own PC
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` for a PC whose `owner_user_id` matches the requesting user
- **THEN** the server SHALL overwrite the existing document's editable fields, including `items` and `currency`, and return HTTP 200

#### Scenario: Update rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` for a PC whose `owner_user_id` does not match the requesting user
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: PC creation rejected with invalid payload
- **WHEN** a client sends `POST /api/pcs` with `max_hp` ≤ 0 or an empty `name`
- **THEN** the server SHALL return HTTP 400 and make no change to the collection

#### Scenario: PC creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/pcs`
- **THEN** the server SHALL respond with HTTP 401 and make no change
