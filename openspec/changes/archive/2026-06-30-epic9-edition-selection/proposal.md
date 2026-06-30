## Why

With Epic 8's monster scrubber now populating MongoDB with edition-aware creatures, the app needs a way to know which edition a room is running so it can filter the correct monster set and prepare the autocomplete contract that Epic 12 (Typesense) will fulfill. Without room-level edition context, every monster lookup is ambiguous and the search endpoint has nothing to filter by.

## What Changes

- `RoomState` gains an `edition` field (`"5e"` | `"5.5e"`)
- `POST /api/rooms` accepts an optional `edition` body field; defaults to `"5e"` if omitted
- The DM room creation UI presents an edition selector before the room is created
- New endpoint `GET /api/search/monsters?q=&edition=` performs an exact-name MongoDB lookup filtered by edition — establishing the contract Epic 12 will later fulfil with Typesense
- The DM Combat Panel search bar is wired to the new endpoint, passing `room.edition` from WebSocket state

## Capabilities

### New Capabilities
- `monster-search`: Edition-filtered monster search endpoint (`GET /api/search/monsters?q=&edition=`)

### Modified Capabilities
- `room-creation`: `POST /api/rooms` now accepts an optional `edition` field; the DM UI presents an edition selector on the creation screen
- `room-state`: `RoomState` data model gains an `edition` field broadcast to all clients

## Impact

- **`room/room.go`** — `RoomState` struct gains `Edition string`; `CreateRoom` accepts edition parameter
- **`api/handler.go`** — `CreateRoom` handler reads `edition` from request body, defaults to `"5e"`
- **`main.go`** — new route `GET /api/search/monsters` wired to new handler
- **`store/monster.go`** — new `SearchMonsters(name, edition string)` method
- **Frontend** — edition selector on join/create screen; DM panel search bar uses `room.edition`
- **Downstream** — Epic 12 replaces the MongoDB query in `monster-search` with Typesense; no frontend or route changes needed
