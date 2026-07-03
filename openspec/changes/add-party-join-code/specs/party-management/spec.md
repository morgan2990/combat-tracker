## MODIFIED Requirements

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

## ADDED Requirements

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
