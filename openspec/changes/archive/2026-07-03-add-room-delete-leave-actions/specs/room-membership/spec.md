## ADDED Requirements

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
