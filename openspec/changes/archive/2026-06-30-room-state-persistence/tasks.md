## 1. MongoDB `rooms` collection

- [x] 1.1 Define a `RoomSnapshot` struct in a new `store/room.go` matching the persisted document shape: `room_id`, `dm_token`, `is_combat_active`, `current_round`, `active_turn_entity_id` (nullable), `edition`, `entities`. Each `RoomEntitySnapshot` mirrors every field of `room.Entity` (id, name, type, owner_id, max_hp, current_hp, temp_hp, initiative, shares_initiative, conditions, dead, source_type, reference_url, pdf_object_key, initiative_modifier, initiative_roll) plus `connected` — full fidelity is required for restore to preserve combat-relevant behavior (see design.md §4a).
- [x] 1.2 In `store.Init()`, create the `rooms` collection and a unique index on `room_id`, following the existing `ensureMonsterIndex` pattern.
- [x] 1.3 Add `SaveRoomSnapshot(snapshot RoomSnapshot) error` (upsert by `room_id`) and `GetRoomSnapshot(roomID string) (*RoomSnapshot, error)` (nil if not found) to the store layer.

## 2. Dirty-tracking and snapshot translation on `Room`

- [x] 2.1 Add a `dirty bool` field to `room.Room`, guarded by the existing `r.mu`.
- [x] 2.2 Add `(r *Room) MarkDirty()` — sets `dirty = true` under lock.
- [x] 2.3 Add `(r *Room) activeEntityID() *string` — returns nil if `!State.IsStarted`, else the ID of `State.Entities[State.ActiveIndex]`.
- [x] 2.4 Add `(r *Room) resolveActiveIndex(id *string) int` — linear scan for matching entity ID; returns `0` if `id` is nil or no match is found.
- [x] 2.5 Add `(r *Room) snapshot() store.RoomSnapshot` — builds a `RoomSnapshot` from current `RoomState` + `DMToken`, computing each entity's `connected` field from whether its `SessionID` is a live key in `r.Clients` (only meaningful for `type == "player"`).
- [x] 2.6 Add `(r *Room) PersistNow(st *store.RoomStore)` — snapshots state and clears `dirty` under `Lock`, then calls `st.SaveRoomSnapshot` outside the lock; re-marks dirty on write failure so the next sweep retries. Takes `*store.RoomStore` (not `*store.Store`, which is the unrelated entity-profile store) — callers spawn it as `go rm.PersistNow(...)`.

## 3. Wire immediate and deferred persistence into the WS layer

- [x] 3.1 In `ws/handler.go`, call `rm.PersistNow` (as a goroutine) alongside the existing `rm.BroadcastState()` calls for: `start_combat`, `end_combat`, `next_turn`, and the join/leave paths in `Handler`/`serve()`'s defer.
- [x] 3.2 For all other dispatch cases that currently call `rm.BroadcastState()` on success (`update_entity`, `dm_update_entity`, `add_creature`, `add_companion`, `remove_entity`, `remove_dead_creatures`, `set_initiative`, `setup_character`, `refresh_from_profile`), additionally call `rm.MarkDirty()`.

## 4. Background sweeper

- [x] 4.1 Add a `Registry.SweepDirty(st *store.RoomStore)` method that snapshots the current list of rooms under `RLock`, then for each room with `dirty == true` calls `PersistNow`.
- [x] 4.2 In `main.go`, after `store.Init()`, start a goroutine that ticks every 30 seconds and calls `room.Global.SweepDirty(&store.GlobalRooms)` for the lifetime of the process.

## 5. Restore path

- [x] 5.1 Add `Registry.GetOrRestoreRoom(roomID string, st *store.RoomStore) (*Room, bool)` — checks `rooms[roomID]` first; on miss, calls `st.GetRoomSnapshot`; if found, inflates a `*Room` (`State` decoded from the snapshot including `ActiveIndex` resolved via `resolveActiveIndex`, `DMToken` from the snapshot, `Clients` as an empty map), registers it into `rooms`, and returns it.
- [x] 5.2 Update `ws.Handler` to call `room.Global.GetOrRestoreRoom(roomID, &store.GlobalRooms)` instead of `room.Global.GetRoom(roomID)`.

## 6. REST restore endpoint

- [x] 6.1 Add `GetRoom(w http.ResponseWriter, r *http.Request)` to `api/handler.go`, calling `GetOrRestoreRoom` and responding `200` with `{room_id, edition, is_combat_active}` on success or `404` if not found. Reads room metadata via a new `Room.Summary()` accessor (lock-safe) rather than touching `rm.State` fields directly from outside the package.
- [x] 6.2 Register `GET /api/rooms/{room_id}` in `main.go`'s mux.

## 7. Tests

- [x] 7.1 Unit tests for `activeEntityID`/`resolveActiveIndex` (nil when not started, correct ID/index round-trip, fallback to 0 on unresolved ID).
- [x] 7.2 Unit tests for `snapshot()`'s `connected` computation (player with live session → true, player with no session → false, companion/creature → not derived from session).
- [ ] 7.3 Integration-style test for `GetOrRestoreRoom`: room in memory (no Mongo call), room only in Mongo (inflated and registered), room in neither (not found). **Deferred** — this codebase has no Mongo mocking/interface layer and no separate test database; the only reachable MongoDB is the same instance the running app uses. Revisit with a proper test-DB or mock if this becomes worth the investment. The "found in memory" branch (no Mongo dependency) is implicitly covered by existing `room_test.go` patterns; the Mongo-fallback branch is unverified by automated tests.
- [ ] 7.4 Test `GET /api/rooms/{room_id}` returns 200 for an existing/restorable room and 404 otherwise. **Deferred** for the same reason as 7.3 — relies on the same untested `GetOrRestoreRoom` Mongo-fallback path. Verify manually when running the app.
