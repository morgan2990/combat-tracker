# Spec: Room Membership

## Purpose

Defines how the system tracks which rooms a user has joined as a player, purely as a recency record surfaced on the dashboard as "recent rooms" — it grants no permission.

## Requirements

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

### Requirement: A user can delete their own room membership (leave a room)

The system SHALL provide a `DELETE /api/rooms/{room_id}/membership` endpoint, available to any authenticated user, that deletes the calling user's own `RoomMembership` record for that `room_id`, identified from their session (not a path or body parameter). This SHALL have no effect on the room itself, on other users' memberships for the same room, or on the room's ownership.

#### Scenario: User leaves a room they previously joined
- **WHEN** an authenticated user with a `RoomMembership` for `room_id` sends `DELETE /api/rooms/{room_id}/membership`
- **THEN** the server SHALL delete that user's `RoomMembership` record, and their subsequent `GET /api/me` response SHALL NOT include that room in recent rooms

#### Scenario: Leaving does not affect other users' memberships
- **WHEN** one user leaves a room that another user also has a `RoomMembership` for
- **THEN** the other user's `RoomMembership` record for that room SHALL be unaffected

#### Scenario: Leaving a room with no existing membership is a no-op
- **WHEN** an authenticated user with no `RoomMembership` for `room_id` sends `DELETE /api/rooms/{room_id}/membership`
- **THEN** the server SHALL respond with a success status without error

#### Scenario: Unauthenticated request is rejected
- **WHEN** a client without a valid session sends `DELETE /api/rooms/{room_id}/membership`
- **THEN** the server SHALL respond with HTTP 401

### Requirement: Player can leave a room from the Dashboard

The Player Dashboard's "Recent Rooms" list SHALL present a "Leave" action per room, guarded by a confirmation prompt, that calls the leave endpoint and removes that room from the displayed list on success.

#### Scenario: Player leaves a room from the list
- **WHEN** the player confirms the "Leave" action for a room in "Recent Rooms"
- **THEN** the client SHALL call `DELETE /api/rooms/{room_id}/membership` and, on success, remove that room from the displayed "Recent Rooms" list without a page reload

#### Scenario: Player cancels the confirmation
- **WHEN** the player activates "Leave" for a room but does not confirm the prompt
- **THEN** the client SHALL NOT call the leave endpoint and the room SHALL remain in the list
