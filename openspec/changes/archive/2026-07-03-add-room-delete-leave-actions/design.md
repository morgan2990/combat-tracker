## Context

Rooms today have a create path (`POST /api/rooms`) and a read/restore path (`GET /api/rooms/{room_id}`, backed by `room.Global` in-memory registry + `store.GlobalRooms` Mongo mirror), but no delete path. `room.Registry` (`room/room.go`) has no method to remove a room from its map, and `Room` has no method to force-close its connected clients — `RemoveClient` only removes one session on disconnect, it doesn't proactively disconnect anyone. `store.RoomStore` has `Save`/`Get` but no `Delete`. `store.MembershipStore` has `Upsert`/`ListByUser` but no delete of any kind — deletion (both the cascade and the player's own "leave") is entirely new surface on that store too.

`RoomMembership` is explicitly documented as "purely a recency record, grants no permission by itself" (`store/membership.go`), which is why "leave" is architecturally simple: deleting a `(user_id, room_id)` row has zero side effects on the room or other users.

## Goals / Non-Goals

**Goals:**
- DM can delete an owned room unconditionally (no combat-state or connected-player gating), and it's fully gone: no live session, no Mongo snapshot, no dangling membership rows.
- A retried/duplicate delete request is safe (idempotent from the caller's perspective).
- Player can remove their own room from Recent Rooms without affecting the room or other players.

**Non-Goals:**
- Not building any confirmation/warning UI beyond the existing `window.confirm` pattern already used for monster/encounter deletes.
- Not adding a "DM kicks a specific player" feature — this is whole-room deletion only.
- Not changing how a room is created, restored, or persisted during normal operation.

## Decisions

- **Room deletion order: in-memory teardown first, then Mongo deletes, synchronously in the request handler.** Remove the room from `room.Global`'s map and close every connected client's WebSocket connection first (so no new state mutations land mid-delete), then delete the Mongo room snapshot, then delete membership rows referencing that `room_id`. Unlike routine state persistence (`PersistNow`, fire-and-forget via goroutine), this handler waits for both Mongo deletes to complete before responding, since a destructive action should confirm completion rather than report success optimistically.
- **Idempotent by construction, not by a special-cased check.** If the Mongo snapshot delete fails after in-memory teardown, a retried `DELETE` re-enters through `requireOwnedRoom` → `GetOrRestoreRoom`, which (finding nothing in memory) restores the room from the still-present Mongo snapshot, then deletion proceeds again. No explicit "already deleted" state needs to be tracked.
- **New `Registry.RemoveRoom(roomID string)` method, not exposed via `Room` itself.** Mirrors the existing shape where `Registry.CreateRoom`/`GetRoom` own the map; `Room` doesn't manage its own lifecycle in the registry today, and deletion shouldn't be the first exception. It closes each `Client.Conn` in the room's `Clients` map, then deletes the map entry under the registry's lock.
- **Membership cascade is a new `MembershipStore.DeleteByRoom(roomID string)`, separate from `MembershipStore.DeleteOne(userID, roomID string)` used by "leave."** Same collection, two distinct deletion shapes (by room vs. by user+room) — keeping them as separate methods matches the existing store's narrow, single-purpose method style (`Upsert`, `ListByUser`) rather than one parameterized delete.
- **Leave endpoint path: `DELETE /api/rooms/{room_id}/membership`, user resolved from session.** No `user_id` in the path — a user can only ever delete their own membership, so it's implicit from `requireUser`, consistent with how other user-scoped endpoints in this codebase avoid taking a redundant user id.

## Risks / Trade-offs

- **Connected players get an abrupt WebSocket close with no advance warning** → Acceptable per the "always allow, no gating" decision; closing the connection is the intended signal. A reconnect attempt will hit `GetOrRestoreRoom` and get a clean not-found (both memory and Mongo snapshot gone), which should surface through whatever generic "room not found" handling the client already has for a bad `room_id` — verify this manually in tasks rather than building new client-side error UI, since no such UI is in scope.
- **Partial failure between the two Mongo deletes** (snapshot deleted, membership cascade fails, or vice versa) → Non-atomic across two Mongo collections; worst case is a temporarily orphaned membership row surviving one extra request cycle. A player hitting a stale "Recent Room" entry gets a 404 on `GetRoom`/join, same as any other missing room today — not a new failure mode, just a delayed cleanup. Not worth a distributed transaction for a two-collection best-effort cascade.

## Open Questions

- None outstanding — this design uses existing patterns (owner-check helper, confirm-guarded delete UX, store method style) throughout.
