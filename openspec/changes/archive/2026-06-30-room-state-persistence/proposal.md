## Why

Room state today lives only in the Go process's memory (`room.Registry.rooms`). A server restart or crash destroys every active combat session with no recovery path, even though the DM-rejoin UI already implies sessions should survive a reconnect. Epic 11 closes that gap: mirror room state to MongoDB so sessions survive restarts and can be restored when a client reconnects.

## What Changes

- Add a `rooms` MongoDB collection storing a snapshot of each room's combat state (`room_id`, `dm_token`, `is_combat_active`, `current_round`, `active_turn_entity_id`, `edition`, `entities`).
- Add a dirty-flag + 30s background sweeper to `room.Room`/`room.Registry` that snapshots and persists rooms whose state changed since the last save.
- Add immediate (fire-and-forget) persistence at specific trigger events: player join, player leave, combat started, combat ended, turn advances.
- Add a `Registry.GetOrRestoreRoom` lookup path: check in-memory registry first, fall back to MongoDB, and inflate a restored `*Room` back into the registry on a hit.
- Add a new `GET /api/rooms/{room_id}` REST endpoint backed by `GetOrRestoreRoom`, returning room metadata on success or `404` if the room exists nowhere.
- Update `ws.Handler` to use `GetOrRestoreRoom` instead of the current memory-only `GetRoom`, so a WebSocket connection against a Mongo-only room also triggers restoration.
- **BREAKING (spec-level)**: `room-creation`'s "Room state is stored in memory only" / "lost on restart" requirement no longer holds — state now survives a restart via MongoDB.
- **BREAKING (spec-level)**: `room-connection`'s "Invalid room ID" scenario (reject any room_id not in the in-memory registry) is superseded — a room_id absent from memory but present in MongoDB is now a valid, restorable connection, not a 4004 rejection.

Out of scope (explicitly deferred): TTL/eviction of old rooms, an offline-player UI indicator, strict write ordering/serialization for persistence writes, frontend pre-flight wiring to the new GET endpoint.

## Capabilities

### New Capabilities
- `room-persistence`: MongoDB-backed snapshotting of room state (dirty-tracking sweeper, triggered immediate writes, restore-on-lookup, the `GET /api/rooms/{room_id}` endpoint, and the persisted document shape).

### Modified Capabilities
- `room-creation`: the "Room state is stored in memory only" requirement changes — room state is no longer lost on restart; it is recoverable from MongoDB.
- `room-connection`: the "Invalid room ID" scenario changes — a room_id not found in memory is no longer immediately rejected; the server SHALL check MongoDB before deciding the room doesn't exist.

## Impact

- `room/room.go`: new dirty flag + `MarkDirty`/`PersistNow` methods on `Room`; new `GetOrRestoreRoom` on `Registry`; index→ID / ID→index translation helpers for `active_turn_entity_id`.
- `store/mongo.go`: new `rooms` collection, index on `room_id`, snapshot read/write functions, following the existing `monsters` collection init pattern.
- `ws/handler.go`: call `MarkDirty`/`PersistNow` alongside existing `BroadcastState()` calls; switch `GetRoom` → `GetOrRestoreRoom`.
- `api/handler.go` + `main.go`: new `GET /api/rooms/{room_id}` handler; start the background sweeper goroutine after `store.Init()`.
- No frontend changes required for the core behavior (connection-status persistence is backend-only; frontend wiring to the new GET endpoint is an optional follow-on, not required by this change).
