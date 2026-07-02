## ADDED Requirements

### Requirement: Any authenticated user can create a Party
An authenticated user SHALL be able to create a Party by sending `POST /api/parties` with a `name`; the server SHALL generate an `id`, store an empty `member_pc_ids` array, and initialize `currency` with all five denominations (`pp`, `gp`, `ep`, `sp`, `cp`) set to `0`. A Party is not owned by a single user and is not scoped to a Room.

#### Scenario: New party is created
- **WHEN** an authenticated client sends `POST /api/parties` with `{ "name": "The Silver Hand" }`
- **THEN** the server SHALL insert a document into a `parties` MongoDB collection with a generated `id`, `member_pc_ids: []`, and `currency` fully zeroed, returning HTTP 200 with the saved party

#### Scenario: Party creation rejected with invalid payload
- **WHEN** an authenticated client sends `POST /api/parties` with an empty `name`
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Party creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/parties`
- **THEN** the server SHALL respond with HTTP 401 and make no change

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
