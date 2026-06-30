## Context

The app is a Go + React combat tracker. All state today is in-memory: rooms, entities, and combat progress live only for the lifetime of the server process. Players re-enter their character name, max HP, and companion details every session.

Epic 6 introduces the first persistent layer: a MongoDB collection storing player and companion profiles keyed by character name. The join flow changes from "enter your stats" to "look up your saved profile." Initiative remains session-only (not persisted) and is now nullable in the runtime entity.

The backend is a single Go binary with four files (`main.go`, `api/handler.go`, `ws/handler.go`, `room/room.go`). The frontend is a React SPA with no routing library.

## Goals / Non-Goals

**Goals:**
- Store player/companion profiles in MongoDB (name, type, max_hp, parent_pc_name, shares_initiative)
- Require a saved profile to join a room (no manual fallback)
- Auto-load profile stats and companions on join; initiative entered separately
- Block `start_combat` until all players and companions have initiative set
- Propagate shared initiative from player to flagged companions automatically
- Allow in-room "Refresh from profile" to update max_hp from MongoDB without affecting other rooms
- Add React Router and a `/characters/new` route for profile creation/editing

**Non-Goals:**
- User accounts or namespaced profiles (deferred)
- Initiative persistence across sessions
- Real-time profile sync to active rooms (player-triggered refresh only)
- Offline / MongoDB-down graceful fallback (hard block on failure)
- Companion creation outside of the character creation screen during a session (manual `add_companion` WS is retained but companions are expected to come from profiles)

## Decisions

### D1: New `store/` package for all MongoDB access

A dedicated `store/` package (`store/mongo.go`) holds the MongoDB client and exposes three functions: `UpsertEntity`, `GetEntityByName`, `GetCompanionsByParent`. The `api` and `ws` packages import `store/` — `store/` never imports them.

**Why not embed MongoDB calls directly in `api/handler.go`?** Keeping DB access isolated makes it mockable in tests and keeps the handler thin. The project is small but will grow (user accounts, room persistence) — a store layer is the right seam.

### D2: Profile fetch via REST, not WebSocket

`GET /api/entities/:name` is a standard HTTP endpoint called by the frontend before the WS connection is opened. The WS path remains stateless with respect to MongoDB.

**Why not query MongoDB during WS upgrade?** A slow or failed DB call during the WS handshake produces a confusing half-open connection. Separating the concerns keeps the WS path fast and simple: by the time the player connects via WS, the frontend already has the profile data.

### D3: `room.Entity.Initiative` becomes `*int`

The Go `int` zero value is indistinguishable from "initiative = 0". A nil pointer cleanly represents "not yet set." The JSON tag uses `omitempty` so unset initiative serializes as `null` to the frontend.

**Why not a sentinel value like -1?** Negative initiative is technically valid in some game systems. A pointer is semantically correct and avoids magic numbers.

### D4: Shared initiative propagated at setup time, not at combat start

When a player sends `set_initiative` (the new WS message that replaces the initiative field in `setup_character`), the server immediately checks all companions in the room whose `SharesInitiative` flag is true and whose `OwnerID` matches the player's entity ID, and sets their initiative to match.

**Why at setup time and not at `start_combat`?** Players set initiative after joining but before combat starts. Propagating immediately keeps the tracker view accurate in real time and removes the need for a separate "copy initiative" step at combat start.

### D5: Companions auto-loaded via WS message sequence, not a special join path

After the player's WS connection is established and the frontend has the profile, the frontend sends `setup_character` (with max_hp from profile, no initiative yet), followed immediately by one `add_companion` message per companion in the profile. This reuses the existing WS dispatch logic with no new server-side join path.

**Why not inflate companions server-side at join time?** Server-side inflation would require passing profile data into `ValidateAndRegister` or a new join hook. The frontend already has the profile (fetched via REST before WS connect); delegating the add_companion messages to the frontend keeps the server stateless and avoids a new coupling between `ws/handler.go` and `store/`.

**Companion initiative on auto-load:** Companions load with `initiative = nil`. If `shares_initiative = true`, initiative is set automatically when the player later sends `set_initiative`. If `shares_initiative = false`, the companion's initiative field stays nil and appears in the tracker as "pending" until the player explicitly sets it.

### D6: `setup_character` no longer accepts `max_hp`

The client sends `{ "type": "setup_character" }` with no max_hp field. The server reads max_hp from the profile stored in the room join context — specifically, it is passed during `ValidateAndRegister` or a new pre-join REST call that the WS handler references. 

**Alternative considered:** keep max_hp in the WS message but validate it against the profile. Rejected — it adds a round-trip validation concern and tempts clients to send arbitrary values.

**Implementation:** On profile-based join, the frontend passes max_hp as a query param on the WS URL (`/ws?...&max_hp=N`). The WS handler stores it on the `Client` struct and uses it when processing `setup_character`. This avoids a new in-memory pre-join state store.

### D7: React Router added with two routes

`/` — the existing join screen  
`/characters/new` — character creation form  
`/room` — the in-game view (player or DM), reached after a successful WS connect

The in-game view currently renders based on app state; it becomes a proper route so the browser back button works and accidental refreshes don't silently drop players.

### D8: MongoDB URI via environment variable

`MONGODB_URI` env var, defaulting to `mongodb://localhost:27017` for local dev. Connection is established at startup; fatal if unreachable. Database name: `combatapp`, collection: `entities`.

## Risks / Trade-offs

- **MongoDB single point of failure** → Players cannot join if MongoDB is down. Acceptable given the small-group use case; document the dependency clearly.
- **Global name namespace** → Two players with the same character name share a profile. Mitigated by the single-friend-group constraint; deferred to user account epic.
- **`*int` initiative change is a schema break** → All existing clients receive `"initiative": null` for new entities. Frontend must handle `null` as "not yet set." Any hardcoded `initiative > 0` checks need updating.
- **WS max_hp via query param is visible in logs** → Low risk for a private app; not a secret value. Would need moving to a header or body if the app becomes public.
- **Companion initiative UX** → Players with non-shared companions must remember to set each companion's initiative separately. The tracker will show a visible "pending" state to remind them.

## Migration Plan

1. Deploy MongoDB alongside the Go binary (Docker or local install)
2. Set `MONGODB_URI` in the environment
3. Existing in-flight rooms are unaffected (purely in-memory, no migration needed)
4. Players create profiles via `/characters/new` before their next session
5. No rollback needed for the DB schema — the collection is append/upsert only; reverting the binary simply stops reading from it

## Open Questions

- Should the "Refresh from profile" also update companion max_hp values, or only the player's own max_hp? (Assumed: yes, also update companions — capture decision in spec.)
- Should the `/characters/new` form allow editing an existing profile (load by name, modify, re-save) or only create new ones? (Assumed: same form — if the name already exists, it upserts.)
