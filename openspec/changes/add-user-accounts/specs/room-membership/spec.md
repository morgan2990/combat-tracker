## ADDED Requirements

### Requirement: Joining a room as a player records a RoomMembership

Whenever an authenticated user successfully joins a room with `role=player`, the system SHALL upsert a `RoomMembership` document keyed by `(user_id, room_id)`, storing the `pc_id` used and the current time as `last_joined_at`. This record grants no permission by itself — it is purely a recency record.

#### Scenario: First join creates a membership
- **WHEN** a user joins a room as a player for the first time, using a given PC
- **THEN** the server SHALL create a `RoomMembership` document with that `user_id`, `room_id`, `pc_id` as `last_pc_id`, and `last_joined_at` set to the current time

#### Scenario: Repeat join updates the existing membership
- **WHEN** a user who already has a `RoomMembership` for a room joins that room again, possibly with a different PC
- **THEN** the server SHALL update the existing document's `last_pc_id` and `last_joined_at` rather than creating a duplicate

#### Scenario: DM connections do not create a membership
- **WHEN** a user connects to a room with `role=dm`
- **THEN** the server SHALL NOT create or update a `RoomMembership` document

### Requirement: A user can fetch their recent room memberships

The system SHALL include the authenticated user's `RoomMembership` records, ordered by `last_joined_at` descending, in the `GET /api/me` response.

#### Scenario: Recent rooms returned in recency order
- **WHEN** an authenticated user with two or more `RoomMembership` records sends `GET /api/me`
- **THEN** the response SHALL include their memberships ordered from most-recently-joined to least-recently-joined

#### Scenario: User with no memberships sees an empty list
- **WHEN** an authenticated user with no `RoomMembership` records sends `GET /api/me`
- **THEN** the response SHALL include an empty list for recent rooms
