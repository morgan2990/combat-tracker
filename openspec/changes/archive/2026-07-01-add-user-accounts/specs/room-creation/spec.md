## MODIFIED Requirements

### Requirement: DM can create a combat room

The system SHALL provide a `POST /api/rooms` endpoint, available only to authenticated users, that creates a new combat room owned by the requesting user and returns a unique room ID. The request body MAY include an optional `edition` field (`"5e"` or `"5.5e"`); if omitted or invalid, the server SHALL default to `"5e"`.

#### Scenario: Successful room creation with edition
- **WHEN** an authenticated client sends `POST /api/rooms` with body `{ "edition": "5.5e" }`
- **THEN** the server SHALL respond with HTTP 201 and a JSON body containing a unique `room_id` and the resolved `edition: "5.5e"`; the created room's `owner_user_id` SHALL be set to the requesting user's id

#### Scenario: Successful room creation without edition defaults to 5e
- **WHEN** an authenticated client sends `POST /api/rooms` with no body
- **THEN** the server SHALL respond with HTTP 201 and `edition: "5e"` in the response

#### Scenario: Room creation rejected when not authenticated
- **WHEN** a client without a valid session sends `POST /api/rooms`
- **THEN** the server SHALL respond with HTTP 401 and SHALL NOT create a room

#### Scenario: Room ID is unique among active rooms
- **WHEN** a room is created and a room with the generated ID already exists in memory
- **THEN** the server SHALL regenerate the ID until a unique one is found before responding

#### Scenario: Room initialized with empty state
- **WHEN** a room is created
- **THEN** the room's combat state SHALL contain no entities, `is_started` set to false, `round` set to 0, `active_index` set to 0, and `edition` set to the resolved value

### Requirement: DM can create a room from the browser UI

The DM Dashboard SHALL present an edition selector (5e / 5.5e) and a "+ New Room" action that creates a room owned by the logged-in user and immediately opens a WebSocket connection to it as DM — no token is returned, copied, or re-entered.

#### Scenario: DM creates a room with edition selected
- **WHEN** the logged-in DM selects "5.5e" and clicks "+ New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5.5e" }`, receive `room_id` and `edition` from the response, and immediately open a WebSocket connection to that room with `role=dm`

#### Scenario: DM creates a room without changing the default edition
- **WHEN** the logged-in DM does not change the edition selector (default: "5e") and clicks "+ New Room"
- **THEN** the client SHALL call `POST /api/rooms` with `{ "edition": "5e" }`

#### Scenario: Room creation API failure is surfaced to the DM
- **WHEN** `POST /api/rooms` returns a non-2xx status
- **THEN** the client SHALL display an error message and SHALL NOT attempt a WebSocket connection

#### Scenario: Owned rooms appear on the dashboard without re-entering credentials
- **WHEN** a logged-in user has previously created one or more rooms
- **THEN** the Dashboard's "My Rooms" list SHALL show each room with an action to open it directly, with no room code or token required

Note: the "Rejoin Existing Room" form (room code + `dm_token`) scenario from the prior version of this requirement is dropped — `dm_token` no longer exists, and a DM's rooms are now listed directly on their Dashboard instead of requiring manual rejoin-by-token.
