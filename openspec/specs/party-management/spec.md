# Spec: Party Management

## Purpose

Defines Parties — cross-user groupings of member PCs with a pooled currency purse — and the Dashboard UI used to create parties, manage membership, and edit pooled currency.

## Requirements

### Requirement: Any authenticated user can create a Party
An authenticated user SHALL be able to create a Party by sending `POST /api/parties` with a `name`; the server SHALL generate an `id`, a `join_code` (5 characters, drawn from the alphabet `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, unique among stored parties), store an empty `member_pc_ids` array, and initialize `currency` with all five denominations (`pp`, `gp`, `ep`, `sp`, `cp`) set to `0`. A Party is not owned by a single user and is not scoped to a Room.

#### Scenario: New party is created
- **WHEN** an authenticated client sends `POST /api/parties` with `{ "name": "The Silver Hand" }`
- **THEN** the server SHALL insert a document into a `parties` MongoDB collection with a generated `id`, a 5-character `join_code` unique among existing parties, `member_pc_ids: []`, and `currency` fully zeroed, returning HTTP 200 with the saved party

#### Scenario: Party creation rejected with invalid payload
- **WHEN** an authenticated client sends `POST /api/parties` with an empty `name`
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Party creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/parties`
- **THEN** the server SHALL respond with HTTP 401 and make no change

#### Scenario: Join code generation retries on collision
- **WHEN** a randomly generated `join_code` matches one already stored on another party
- **THEN** the server SHALL generate a new candidate code and retry, without returning a duplicate code or failing the request under normal conditions

### Requirement: Any user owning a member PC can manage party membership
A user SHALL be able to add or remove PCs from a Party's `member_pc_ids` by sending `PUT /api/parties/:id` with an updated `member_pc_ids` array. Any authenticated user MAY add any PC (including PCs they do not own) to a party's membership. Editing membership requires either that the party currently has no members, or that the requester owns at least one PC already in `member_pc_ids`.

#### Scenario: First member added to a new party
- **WHEN** an authenticated client sends `PUT /api/parties/:id` for a party with an empty `member_pc_ids`, setting `member_pc_ids` to include one PC
- **THEN** the server SHALL save the updated membership and return HTTP 200

#### Scenario: Existing member's owner adds another user's PC
- **WHEN** an authenticated client who owns a PC already listed in `member_pc_ids` sends `PUT /api/parties/:id` adding a PC owned by a different user
- **THEN** the server SHALL save the updated membership and return HTTP 200

#### Scenario: Membership edit rejected for a non-member's owner
- **WHEN** an authenticated client who owns none of the PCs currently in `member_pc_ids` sends `PUT /api/parties/:id` to change membership on a party that already has members
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Membership edit rejected for a party that does not exist
- **WHEN** an authenticated client sends `PUT /api/parties/:id` for an `id` that does not exist
- **THEN** the server SHALL respond with HTTP 404

### Requirement: Any user owning a member PC can edit the party's pooled currency
A user who owns at least one PC listed in a Party's `member_pc_ids` SHALL be able to update the party's pooled `currency` (denominations `pp`, `gp`, `ep`, `sp`, `cp`) via `PUT /api/parties/:id`. The pooled currency is independent of any member PC's personal `currency`.

#### Scenario: Member's owner adjusts pooled currency
- **WHEN** an authenticated client who owns a PC in `member_pc_ids` sends `PUT /api/parties/:id` with an updated `currency` value
- **THEN** the server SHALL overwrite the party's stored `currency` and return HTTP 200

#### Scenario: Currency edit rejected for a non-member's owner
- **WHEN** an authenticated client who owns none of the party's member PCs sends `PUT /api/parties/:id` with a `currency` change
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Negative pooled currency values are rejected
- **WHEN** an authenticated client sends `PUT /api/parties/:id` with any `currency` field less than `0`
- **THEN** the server SHALL return HTTP 400 and make no change to the stored party

### Requirement: A user can fetch parties relevant to their PCs
The system SHALL provide `GET /api/parties/:id`, returning the party document to any authenticated user, and SHALL include a user's party memberships (parties containing at least one PC they own) in `GET /api/me` (see `user-accounts`).

#### Scenario: Party fetched by id
- **WHEN** an authenticated client sends `GET /api/parties/:id` for an existing party
- **THEN** the server SHALL return HTTP 200 with the party document

#### Scenario: Party not found
- **WHEN** an authenticated client sends `GET /api/parties/:id` for an `id` that does not exist
- **THEN** the server SHALL respond with HTTP 404

#### Scenario: User's own parties listed via /api/me
- **WHEN** an authenticated client sends `GET /api/me` and owns a PC that is a member of one or more parties
- **THEN** the response SHALL include those parties (or a summary of them) alongside the user's PC list

### Requirement: Party membership and pooled currency are managed from a Dashboard "Parties" section
Dashboard SHALL include a "Parties" section listing the authenticated user's parties, allowing them to create a party, add/remove member PCs, and view/edit the pooled currency.

#### Scenario: User creates a party from Dashboard
- **WHEN** a user submits the "create party" form in Dashboard's Parties section
- **THEN** the client SHALL send `POST /api/parties` and display the newly created party in the list

#### Scenario: User edits pooled currency from Dashboard
- **WHEN** a user who owns a member PC updates the pooled currency fields in the Parties section and saves
- **THEN** the client SHALL send `PUT /api/parties/:id` with the updated `currency` and reflect the saved value on success

### Requirement: A Party has a join code that any user can present to self-join
Every Party SHALL have a `join_code`, included in the party document returned by `GET /api/parties/:id` and by the create/update endpoints. Any authenticated user SHALL be able to add one of their own PCs to a party by sending `POST /api/parties/join` with `{ join_code, pc_id }`. Authorization requires only that the requester owns `pc_id` — unlike `PUT /api/parties/:id`, this path does not require the requester to already own a PC in the party's `member_pc_ids`.

#### Scenario: User self-joins a party with a valid code
- **WHEN** an authenticated user sends `POST /api/parties/join` with a valid `join_code` and the `pc_id` of a PC they own
- **THEN** the server SHALL add that `pc_id` to the party's `member_pc_ids` and return HTTP 200 with the updated party

#### Scenario: Self-join rejected for an unowned PC
- **WHEN** an authenticated user sends `POST /api/parties/join` with a `pc_id` they do not own
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Self-join rejected for an unknown code
- **WHEN** an authenticated user sends `POST /api/parties/join` with a `join_code` that matches no party
- **THEN** the server SHALL respond with HTTP 404 and make no change

#### Scenario: Self-join is idempotent for an already-present PC
- **WHEN** an authenticated user sends `POST /api/parties/join` with a valid `join_code` and a `pc_id` already present in that party's `member_pc_ids`
- **THEN** the server SHALL return HTTP 200 with the unchanged party, making no duplicate entry

### Requirement: A party's join code is visible to Dashboard users who can view that party
Dashboard's Parties section SHALL display each listed party's `join_code`, and SHALL provide a "join a party" control accepting a join code and one of the user's own PCs, submitting via `POST /api/parties/join`.

#### Scenario: Join code shown on a party card
- **WHEN** a user views a party they are a member of in Dashboard's Parties section
- **THEN** the party's `join_code` SHALL be displayed on that party's card

#### Scenario: User joins a party from Dashboard
- **WHEN** a user enters a join code and selects one of their own PCs in the "join a party" control and submits
- **THEN** the client SHALL send `POST /api/parties/join` with that code and PC, and on success SHALL display the joined party in the user's party list

### Requirement: Dashboard's pooled-currency editor offers the same delta-entry modal as InventoryPanel

Dashboard's pooled-currency editor SHALL offer the same delta-entry modal specified for `CurrencyEditor` in `pc-inventory`.

#### Scenario: User applies a delta to pooled currency from Dashboard
- **WHEN** a user who owns a member PC opens the delta modal on the party's pooled currency in Dashboard's Parties section, enters one or more deltas, and clicks Apply
- **THEN** the pooled currency's editable state SHALL reflect the applied deltas, ready to be saved via the existing `PUT /api/parties/:id` save action
