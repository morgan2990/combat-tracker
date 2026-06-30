## Why

Players currently re-enter their character stats (name, max HP, companions) from scratch every combat session. This is repetitive and error-prone for regular players whose characters don't change between sessions. Persistent profiles let players save their character once and load it automatically on every join.

## What Changes

- **New**: MongoDB integration in the Go backend for persistent entity storage
- **New**: `store/` package with `UpsertEntity`, `GetEntityByName`, `GetCompanionsByParent` operations
- **New**: `POST /api/entities` — upsert a player or companion profile by name
- **New**: `GET /api/entities/:name` — fetch a player profile and their companions
- **New**: `/characters/new` route (requires adding React Router) with a character creation form
- **New**: `shares_initiative` field on companion profiles — if true, companion copies owner's initiative automatically when the owner sets theirs
- **Modified**: `room.Entity.Initiative` changes from `int` to `*int` (pointer, nullable) — initiative is runtime-only and starts unset after a profile-based join
- **Modified**: Player join flow — players must have a saved profile; manual stat entry is removed; max_hp is loaded from profile; initiative is set separately after joining
- **Modified**: `start_combat` gains a pre-flight check — all player and companion entities must have a non-nil initiative before combat can begin
- **Modified**: `setup_character` WS message — max_hp is no longer provided by the client; it is loaded from the profile on join; only initiative is submitted
- **Modified**: `add_companion` WS message — companions auto-load from the player's profile on join; the existing manual `add_companion` flow is retained for adding companions not in the profile
- **New**: "Refresh from profile" button in PlayerView — re-fetches the player's MongoDB profile and updates max_hp in the current room (capping current_hp if needed), without affecting other rooms

## Capabilities

### New Capabilities

- `player-profile-management`: Persistent storage of player and companion profiles in MongoDB. Covers the character creation screen (`/characters/new`), the upsert endpoint (`POST /api/entities`), the fetch endpoint (`GET /api/entities/:name`), and the "Refresh from profile" in-room action.
- `profile-based-join`: Profile-required player join flow. Covers the "Find my character" button on the join screen, profile fetch and validation, auto-population of max_hp, and auto-loading of companion entities linked to the player profile.

### Modified Capabilities

- `player-character-setup`: Initiative is now nullable (`*int`); `setup_character` no longer accepts `max_hp` (loaded from profile); initiative is set as a separate step post-join.
- `companion-management`: Companions gain a `shares_initiative` profile field; companions linked to a player profile are auto-loaded into the room on join alongside the player entity.
- `combat-turn-flow`: `start_combat` is blocked until every player and companion entity in the room has a non-nil initiative value.

## Impact

- **Backend**: New MongoDB dependency (`go.mongodb.org/mongo-driver`); new `store/` package; two new HTTP endpoints; `room.Entity` struct change (`Initiative *int`); `StartCombat` logic update; `SetupCharacter` signature change
- **Frontend**: React Router added; new `/characters/new` route; `JoinScreen` player tab revised; `PlayerView` gains "Refresh from profile" button; `types.ts` `Entity.initiative` becomes `number | null`
- **Breaking**: `setup_character` WS message no longer accepts `max_hp`; clients sending the old message shape will have max_hp ignored (loaded from profile instead)
- **Infrastructure**: MongoDB instance required at runtime; connection URI configured via environment variable (`MONGODB_URI`)
