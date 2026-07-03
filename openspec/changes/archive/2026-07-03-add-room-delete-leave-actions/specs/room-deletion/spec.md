## ADDED Requirements

### Requirement: Room owner can delete a room
The system SHALL provide a `DELETE /api/rooms/{room_id}` endpoint, available only to the room's owner (same ownership check as other owner-only room endpoints). Deletion SHALL be unconditional — it SHALL NOT be blocked by an active combat state or by any currently connected client.

#### Scenario: Owner deletes their room
- **WHEN** the room's owner sends `DELETE /api/rooms/{room_id}` for a room they own
- **THEN** the server SHALL respond with a success status and the room SHALL no longer be resolvable by `room_id`, either in memory or via restoration from MongoDB

#### Scenario: Non-owner cannot delete a room
- **WHEN** an authenticated user who does not own the room sends `DELETE /api/rooms/{room_id}`
- **THEN** the server SHALL respond with HTTP 403 and SHALL NOT delete the room

#### Scenario: Unauthenticated request is rejected
- **WHEN** a client without a valid session sends `DELETE /api/rooms/{room_id}`
- **THEN** the server SHALL respond with HTTP 401 and SHALL NOT delete the room

#### Scenario: Deleting a room mid-combat is allowed
- **WHEN** the room's owner deletes a room whose combat is currently active
- **THEN** the server SHALL delete the room without requiring combat to be ended first

#### Scenario: Deleting a nonexistent room
- **WHEN** a request targets a `room_id` that does not exist in memory or MongoDB
- **THEN** the server SHALL respond with HTTP 404

### Requirement: Deletion tears down the live session and disconnects connected clients
When a room is deleted, the system SHALL remove it from the in-memory room registry and close the WebSocket connection of every currently connected client (DM or player) in that room.

#### Scenario: Connected player is disconnected on room deletion
- **WHEN** a player is connected via WebSocket to a room that its owner then deletes
- **THEN** that player's WebSocket connection SHALL be closed by the server as part of the deletion

#### Scenario: Deleted room is absent from the in-memory registry
- **WHEN** a room has been deleted
- **THEN** a subsequent in-memory-only lookup for that `room_id` SHALL report not-found

### Requirement: Deletion removes the persisted snapshot
When a room is deleted, the system SHALL delete its persisted snapshot document from the MongoDB `rooms` collection, so the room cannot be restored after deletion.

#### Scenario: Deleted room does not restore from MongoDB
- **WHEN** a room has been deleted and a subsequent lookup for its `room_id` is performed
- **THEN** the lookup SHALL find no in-memory room and no MongoDB snapshot, and SHALL report not-found

### Requirement: Deletion cascades to remove player membership records
When a room is deleted, the system SHALL delete every `RoomMembership` record referencing that room's `room_id`, so no player retains a "Recent Room" entry pointing at a deleted room.

#### Scenario: Deleting a room removes players' membership records
- **WHEN** a room with one or more players' `RoomMembership` records is deleted
- **THEN** every `RoomMembership` record with that `room_id` SHALL be deleted, and those players' subsequent `GET /api/me` responses SHALL NOT include that room

### Requirement: DM can delete a room from the Dashboard
The DM Dashboard's "My Rooms" list SHALL present a "Delete" action per room, guarded by a confirmation prompt, that calls the delete endpoint and removes the room from the displayed list on success.

#### Scenario: DM deletes a room from the list
- **WHEN** the DM confirms the "Delete" action for a room in "My Rooms"
- **THEN** the client SHALL call `DELETE /api/rooms/{room_id}` and, on success, remove that room from the displayed "My Rooms" list without a page reload

#### Scenario: DM cancels the confirmation
- **WHEN** the DM activates "Delete" for a room but does not confirm the prompt
- **THEN** the client SHALL NOT call the delete endpoint and the room SHALL remain in the list
