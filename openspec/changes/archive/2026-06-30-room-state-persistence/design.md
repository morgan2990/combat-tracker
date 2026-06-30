## Context

`room.Registry` holds all room state (`room.Room{State, DMToken, Clients}`) purely in process memory; `store/mongo.go` only persists `entities` (player/companion profiles) and `monsters`. A server restart wipes every active room, even though `JoinScreen.tsx` already exposes a DM-rejoin flow (room ID + dm_token) implying sessions are expected to survive reconnects. This change adds a MongoDB-backed snapshot of room state and a restore path, while keeping the live WebSocket combat loop (Go memory, `r.mu`-guarded) as the fast path it already is.

## Goals / Non-Goals

**Goals:**
- Survive a server restart/crash without losing in-progress combat (round, active turn, HP, conditions, entities) beyond a small, bounded window.
- Restore a room transparently on the first WebSocket connection (or REST lookup) that references a room_id no longer in memory.
- Keep `room.go`'s combat-turn logic (index-based active turn, mutex discipline, existing tests) untouched.

**Non-Goals:**
- No TTL/eviction or cleanup story for old rooms (acceptable at this project's personal/low-volume scale).
- No offline-player UI indicator — connection status is backend/Mongo bookkeeping only, not broadcast to clients.
- No strict write ordering/serialization for persistence writes — occasional stale-write clobber is an accepted risk, self-healed by the next sweep.
- No frontend wiring of the new `GET /api/rooms/{room_id}` endpoint — it must work standalone; pre-flight UX is a future follow-on.

## Decisions

### 1. Persistence stays out of `room.go`'s mutation methods
`Room` gains a `dirty bool` (guarded by the existing `r.mu`) and two methods: `MarkDirty()` and `PersistNow(st *store.Store)`. Combat-turn methods (`StartCombat`, `NextTurn`, `AddCreature`, etc.) are untouched — they don't know persistence exists. The calling layer (`ws/handler.go`, right next to each existing `rm.BroadcastState()` call) decides whether to call `MarkDirty()` (deferred) or `PersistNow()` (immediate, fire-and-forget goroutine).
**Alternative considered**: thread a `*store.Store` into every `Room` method (matching the existing `RefreshFromProfile` pattern). Rejected — it would force every combat-turn unit test to deal with a store dependency, for a concern (when to persist) that's really about WS event types, not state-mutation semantics.

### 2. Two-tier write timing: immediate vs. swept
AC3 in Epic 11 names five "must write whenever" trigger events (join, leave, combat start, combat end, turn advance) plus a 30s catch-all ticker "if changes occurred." These map to two call patterns:
- **Immediate** (`PersistNow`, fire-and-forget goroutine): the five named trigger events. Each takes its own fresh `RoomState` snapshot at write time.
- **Deferred** (`MarkDirty` only): everything else (HP/condition edits, add/remove creature or companion, DM overrides, initiative changes). A single background sweeper goroutine (started in `main.go` after `store.Init()`) ticks every 30s, scans `room.Global.rooms`, and calls `PersistNow` on any room with `dirty == true`, then clears the flag.

No per-room write-serialization is added: `PersistNow` reads `RoomState` fresh under `r.RLock()` at the moment it's called and writes outside the lock, same pattern as `BroadcastState()`. Two near-simultaneous `PersistNow` calls for the same room could theoretically complete out of order and let a stale write clobber a fresher one — accepted as a rare, self-healing race (next sweep tick corrects it), not worth a dedicated write-mutex at this project's scale.

### 3. `active_turn_entity_id` is a serialization-only concept
Runtime active-turn tracking stays `State.ActiveIndex` (int) — no change to `NextTurn`/`StartCombat`/`RemoveEntity`/`RemoveDeadCreatures`/`DMUpdateEntity`. Two small helpers live at the persistence boundary:
- `activeEntityID() *string` — save time: nil if `!IsStarted`, else the ID of `Entities[ActiveIndex]`.
- `resolveActiveIndex(id *string) int` — restore time: linear scan for matching ID (same shape as the existing ID-preservation block inside `sortEntities()`); falls back to `0` if `id` is nil or unresolved.
**Alternative considered**: switch the runtime model to ID-based and drop `ActiveIndex` entirely. Rejected — touches index arithmetic in several already-tested methods for a benefit (no translation layer) that's mostly aesthetic.

### 4. Connection status is computed only at snapshot time, not stored on `Entity`
`room.Entity` and the WebSocket `RoomState` broadcast are unchanged. The Mongo snapshot document has its own `entities[].connected` field, computed when `PersistNow` builds the snapshot: `true` if the entity's `SessionID` is a live key in `r.Clients`, `false`/not-applicable for companions and creatures (no `SessionID`). This avoids adding a field to every WS broadcast for a concern only the persistence layer (and Epic 11's AC2) actually needs.

### 4a. Snapshot entities carry full fidelity, not just AC2's literal field list
Epic 11's AC2 lists a minimal field set ("ID, name, current_hp, max_hp, initiative, conditions, and connection status") for illustration, but a literal reading breaks restoration: `Type` is required to distinguish player/creature/companion (used by `EndCombat`, `RemoveDeadCreatures`, reconnect matching), `OwnerID` is required for companion ownership, `Dead` must survive a restart, and so on. `store.RoomEntitySnapshot` therefore mirrors every field of `room.Entity` (minus `SessionID`'s liveness, which is harmless to carry as inert data since `Clients` is rebuilt empty and reconnect overwrites it) plus the persistence-only `Connected` field. `store` cannot import `room` (an import cycle, since `room` already imports `store`), so `RoomEntitySnapshot` is an independent mirror struct, following the same pattern as `store.Profile` already being independent of `room.Entity`.

### 5. Single shared restore path used by both REST and WebSocket
`Registry.GetOrRestoreRoom(roomID string) (*Room, error)` replaces direct use of `GetRoom` at both call sites:
1. `RLock`, check `rooms[roomID]` — hit → return immediately (fast path, no Mongo round-trip on every connect).
2. Miss → query the new `rooms` Mongo collection by `room_id`.
3. Found → inflate a `*Room{State: <decoded snapshot>, DMToken: <decoded>, Clients: map[string]*Client{}}` (empty `Clients` — no WebSocket connection survives a process restart; clients re-populate it as they reconnect through the existing `ValidateAndRegister` flow, which already matches player names back to entities by `SessionID`/name).
4. Register the inflated room into `Registry.rooms`, return it.
5. Not found anywhere → return a sentinel "not found" error.

`GET /api/rooms/{room_id}` and `ws.Handler`'s existing `GetRoom` call both go through this one method — no duplicated lookup/restore logic. `ws.Handler`'s existing post-`ValidateAndRegister` call to `rm.BroadcastState()` already satisfies AC4 of US11.2 (push full state down the pipe on connect) once the room is correctly inflated beforehand.

### 6. Mongo `rooms` collection follows the existing `monsters` init pattern
`store.Init()` gains a `roomsCol := db.Collection("rooms")` plus a unique index on `room_id`, mirroring `ensureMonsterIndex`. A new `GlobalRooms RoomStore` (or similar) exposes `Save(snapshot)` (upsert by `room_id`) and `GetByRoomID(id)`.

## Risks / Trade-offs

- **[Risk]** Fire-and-forget `PersistNow` writes can race and let a stale snapshot overwrite a fresher one under concurrent triggers for the same room. → **Mitigation**: accepted; the 30s sweeper re-writes current state shortly after, bounding the staleness window. Revisit with a per-room write-mutex only if real-world clobbering is observed.
- **[Risk]** A crash between two trigger events loses any state changed only via `MarkDirty` (no immediate write) since the last successful sweep — up to ~30s of HP/condition edits. → **Mitigation**: explicitly acceptable per Epic 11 AC3's own design (periodic ticker is the documented catch-all, not a guarantee of zero data loss).
- **[Risk]** Restoring a room re-creates an empty `Clients` map; until players reconnect, `BroadcastState()` for a freshly-restored room has zero recipients and the restoring client is the only one who gets the pushed state. → **Mitigation**: matches existing reconnect behavior (`room-connection` spec's "Player reconnects with name matching existing entity" scenario already handles re-binding `session_id` on reconnect); no new behavior needed here, just confirming restored rooms hit that same path.

## Migration Plan

No data migration needed — this is new persistence, not a change to existing stored shapes. Rollout is a single deploy: `store.Init()` creates the `rooms` collection/index on first run if absent (same pattern as `ensureMonsterIndex`). No rollback hazard beyond reverting the deploy; existing in-memory-only behavior resumes if reverted (Mongo `rooms` collection simply goes unread).

## Open Questions

- Should `GET /api/rooms/{room_id}` ever get wired into `JoinScreen.tsx`'s DM-rejoin form as a pre-flight check? Deferred — flagged as optional in tasks.md.
