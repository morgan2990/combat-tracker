## Why

DMs currently have to rebuild the same monster group from scratch every time they run a recurring encounter (a goblin ambush, a room full of skeletons) — searching, setting quantity, and aliasing each group live during a session instead of prepping it calmly beforehand. Epic 19 adds saveable "Encounter" blueprints a DM can build ahead of time and inject into any live room with one click.

## What Changes

- Add a new `encounters` MongoDB collection with ownership-scoped CRUD (`Create`, `List` (owner-filtered), `Get`, `Update`, `Delete`), mirroring the existing `CustomMonster` pattern exactly (`owner_id` set on create, checked on update/delete, `ListByOwner` for listing).
- Each encounter stores `name`, `owner_id`, `edition`, and a `monsters` array of `{ name | monster_id, is_custom, quantity, display_name? }` groups. **Deviates from AC2's literal single `monster_id` field**: official monsters are referenced by `name` (reusing the existing `GetMonsterByName` lookup, since no by-ID lookup exists for official monsters today) while custom monsters are referenced by their real Mongo `id`; `is_custom` discriminates which key a given entry holds. This avoids adding a new `GetMonsterByID` code path for a lookup the app doesn't otherwise need.
- **Scope addition beyond the epic's literal ACs**: full update support — `PUT /api/encounters/:id` and an `/encounters/:id/edit` screen, mirroring `MonsterForm`'s create/edit split — since AC3 already says `POST /api/encounters` is "Create/Update" but no edit screen was actually specified anywhere in the epic.
- Add a Dashboard "My Encounters" list (mirroring the existing "My Monsters" list exactly: row per encounter, Edit link, Delete button, "+ New Encounter" link) and a `/encounters/new` builder screen: name field, edition selector, Typesense-backed monster search, and a staging list of `{monster, quantity, alias}` groups.
- Add an in-room DM Panel "Encounter Templates" dropdown that fetches `GET /api/encounters?edition=<room's edition>` (server-side filtered, matching the existing `GET /api/search/monsters?edition=` convention) and, on selection, sends a new `inject_encounter` WS message with the encounter's ID.
- Add `InjectEncounter(sessionID, groups) error` to `room.Room`: resolves each monster group's full stats (already done in `ws/handler.go` before calling in, using the same by-name/by-ID split as storage), then loops the same per-instance entity-creation logic `AddCreature` already has — batch quantity expansion, per-instance d20 rolls gated on `IsStarted`, alias auto-numbering — under a single lock and a single sort/broadcast for the whole encounter, not one broadcast per monster group.
- **Deviates from AC3's silence on failure handling**: if a monster group's reference no longer resolves (the custom monster was deleted since the encounter was saved), that group is skipped and the rest of the encounter still injects — consistent with the app's existing tolerance for dangling references elsewhere (creature entities, companions) rather than aborting the whole injection.

## Capabilities

### New Capabilities
- `encounter-repository`: the `encounters` Mongo schema and ownership-scoped CRUD endpoints (mirrors `monster-repository`'s scope).
- `encounter-builder`: the Dashboard "My Encounters" list and the `/encounters/new` / `/encounters/:id/edit` frontend screens (mirrors `monster-form`'s scope).
- `encounter-injection`: the `inject_encounter` WS message, the DM Panel's "Encounter Templates" dropdown, and `InjectEncounter`'s reuse of `AddCreature`'s per-instance logic.

### Modified Capabilities
(none — injection is purely additive; it calls existing `AddCreature`-equivalent logic without changing any existing struct, message, or requirement)

## Impact

- `store/encounter.go` (new): `Encounter`/`EncounterMonster` structs, `EncounterStore` (Create/List/Get/Update/Delete).
- `api/handler.go`: `CreateEncounter`, `ListMyEncounters`, `GetEncounter`, `UpdateEncounter`, `DeleteEncounter`.
- `main.go`: route registrations under `/api/encounters`.
- `room/room.go`: `InjectEncounter` method.
- `ws/handler.go`: `inject_encounter` WS message struct and dispatch case (includes the Mongo lookups needed to resolve each group before calling into `room`).
- `frontend/src/types.ts`: `Encounter`/`EncounterMonster` types.
- `frontend/src/components/EncounterForm.tsx` (new): create/edit screen, mirroring `MonsterForm.tsx`'s structure.
- `frontend/src/components/Dashboard.tsx`: "My Encounters" list section.
- `frontend/src/components/DMView.tsx`: "Encounter Templates" dropdown.
- `frontend/src/App.tsx`: `/encounters/new` and `/encounters/:id/edit` routes.
- No changes to `room-state`, `room-persistence`, or `entity-schema` — injected creatures are ordinary `creature`-type entities created through the same path manual `add_creature` already uses.
