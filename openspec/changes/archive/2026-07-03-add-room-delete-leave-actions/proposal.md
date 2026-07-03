## Why

The Dashboard has no way to remove a room from either side: a DM's "My Rooms" list only offers "Open," and a player's "Recent Rooms" list only offers "Go." Rooms accumulate indefinitely with no cleanup path, and there's currently no backend endpoint to delete a room at all.

## What Changes

- Add `DELETE /api/rooms/{room_id}`, owner-only. Deletion is always allowed (no blocking on active combat or connected players) — it tears down the live in-memory room (force-disconnecting any connected WebSocket clients), deletes the persisted MongoDB snapshot, and cascades to delete every `RoomMembership` row referencing that `room_id` so players don't retain a dead "Recent Room" entry.
- Add `DELETE /api/rooms/{room_id}/membership`, deleting only the calling user's own `RoomMembership` row for that room (a "leave" — removes the room from that user's own Recent Rooms list; has no effect on the room itself or other users' memberships).
- Add a `Delete` method to `store.RoomStore` and delete-by-room / delete-by-user-and-room methods to `store.MembershipStore`.
- Dashboard frontend: add a confirm-guarded "Delete" button to each row in the DM's "My Rooms" list, and a confirm-guarded "Leave" button to each row in the Player's "Recent Rooms" list — same `window.confirm` pattern already used for deleting monsters/encounters.

## Capabilities

### New Capabilities
- `room-deletion`: A room's owner can permanently delete it — tearing down the live session, removing the persisted snapshot, and cascading to remove player membership records that reference it.

### Modified Capabilities
- `room-membership`: A user can delete their own `RoomMembership` record (leave a room), independent of the room-deletion cascade.

## Impact

- `api/rooms.go`: new `DeleteRoom` and `LeaveRoom` (or similarly named) handlers, reusing the existing `requireOwnedRoom` / `requireUser` helpers.
- `room/room.go`: a registry method to remove a room and disconnect its clients.
- `store/room.go`: new `RoomStore.Delete` method.
- `store/membership.go`: new `MembershipStore` methods to delete by room and by (user, room).
- `frontend/src/components/Dashboard.tsx`: "Delete" action on DM room rows, "Leave" action on player recent-room rows.
