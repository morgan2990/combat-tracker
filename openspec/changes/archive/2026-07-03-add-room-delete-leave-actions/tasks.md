## 1. Store layer

- [x] 1.1 Add `RoomStore.Delete(roomID string) error` to `store/room.go`, deleting the document by `room_id`.
- [x] 1.2 Add `MembershipStore.DeleteByRoom(roomID string) error` to `store/membership.go`, deleting all documents matching `room_id`.
- [x] 1.3 Add `MembershipStore.DeleteOne(userID, roomID string) error` to `store/membership.go`, deleting the single document matching `user_id` + `room_id` (no-op, no error, if none exists).

## 2. Room registry layer

- [x] 2.1 Add `Registry.RemoveRoom(roomID string)` to `room/room.go`: under the registry lock, look up the room, close every `Client.Conn` in its `Clients` map, then delete the room from the registry's map.

## 3. API handlers

- [x] 3.1 Add `DeleteRoom` handler in `api/rooms.go`: `requireUser` → `requireOwnedRoom` → `Registry.RemoveRoom` → `store.GlobalRooms.Delete` → `store.GlobalMemberships.DeleteByRoom` → respond success.
- [x] 3.2 Add `LeaveRoom` handler in `api/rooms.go`: `requireUser` → `store.GlobalMemberships.DeleteOne(userID, roomID)` → respond success.
- [x] 3.3 Register `DELETE /api/rooms/{room_id}` and `DELETE /api/rooms/{room_id}/membership` routes (wherever routes are wired, alongside the existing room routes).

## 4. Frontend

- [x] 4.1 In `Dashboard.tsx`, add a `handleDeleteRoom(roomId)` function: `window.confirm` guard, `DELETE /api/rooms/{room_id}`, on success filter the room out of `me.rooms`' rendered list (mirroring `handleDeleteMonster`/`handleDeleteEncounter`'s optimistic-update pattern).
- [x] 4.2 Add a "Delete" button (styled with the existing `deleteBtn` style) to each row in the DM's "My Rooms" list, next to "Open."
- [x] 4.3 In `Dashboard.tsx`, add a `handleLeaveRoom(roomId)` function: `window.confirm` guard, `DELETE /api/rooms/{room_id}/membership`, on success filter the room out of the rendered "Recent Rooms" list.
- [x] 4.4 Add a "Leave" button (styled with the existing `deleteBtn` style) to each row in the Player's "Recent Rooms" list, next to "Go."

## 5. Manual verification

- [x] 5.1 As DM, delete a room with no connected players; confirm it disappears from "My Rooms" and `GET /api/rooms/{room_id}` 404s afterward.
- [x] 5.2 As DM, delete a room while a player is connected via WebSocket in another session; confirm that player's connection closes and a reconnect attempt fails cleanly (not found).
- [x] 5.3 As DM, delete a room mid-combat; confirm no blocking/error occurs.
- [x] 5.4 As a player who joined that deleted room, confirm it no longer appears in "Recent Rooms" after refreshing `/api/me`.
- [x] 5.5 As a player, use "Leave" on a Recent Room; confirm it disappears from the list and the room itself (and its DM) is unaffected.
- [x] 5.6 Confirm a non-owner calling the delete endpoint directly gets a 403 and the room is untouched.
