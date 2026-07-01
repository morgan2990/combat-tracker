## 1. Backend — encounter storage

- [x] 1.1 Create `store/encounter.go` with `Encounter` and `EncounterMonster` structs (per the `encounter-repository` spec), an `EncounterStore` wrapping a Mongo collection, and a package-level `GlobalEncounters EncounterStore`
- [x] 1.2 Implement `CreateEncounter(e Encounter) (Encounter, error)` — generates `ID` via `newID()` if unset, inserts, mirrors `CustomMonsterStore.CreateCustomMonster`'s shape (no Typesense mirroring needed — encounters aren't searched)
- [x] 1.3 Implement `GetEncounterByID(id string) (*Encounter, error)` — mirrors `GetCustomMonsterByID`
- [x] 1.4 Implement `UpdateEncounter(id string, e Encounter) (Encounter, error)` — full-document replace, mirrors `UpdateCustomMonster`
- [x] 1.5 Implement `DeleteEncounter(id string) error` — mirrors `DeleteCustomMonster`
- [x] 1.6 Implement `ListEncountersByOwner(ownerID string, edition string) ([]Encounter, error)` — filters by `owner_id`, and additionally by `edition` when non-empty
- [x] 1.7 Register the `encounters` collection index setup (unique index on `id`) in the store init path, mirroring how `custom_monsters` is indexed

## 2. Backend — HTTP handlers and routes

- [x] 2.1 Add `CreateEncounter`, `GetEncounter`, `UpdateEncounter`, `DeleteEncounter`, `ListMyEncounters` to `api/handler.go`, mirroring the corresponding `*CustomMonster` handlers' auth/ownership/validation shape exactly (edition validation, `owner_id` set server-side, 401/403/404 handling)
- [x] 2.2 Register routes in `main.go`: `POST /api/encounters`, `GET /api/encounters`, `GET /api/encounters/{id}`, `PUT /api/encounters/{id}`, `DELETE /api/encounters/{id}` (a distinct top-level path, no ServeMux collision risk)
- [x] 2.3 `ListMyEncounters` reads an optional `edition` query parameter and passes it through to `ListEncountersByOwner`

## 3. Backend — injection pipeline

- [x] 3.1 Add `InjectEncounter(sessionID string, groups []MonsterGroup) error` to `room/room.go` — under one `r.mu.Lock()`, loop the same per-group entity-creation logic `AddCreature`'s inner loop performs (batch numbering, per-instance auto-roll gated on `IsStarted`), then one `r.sortEntities()` call. Refactored the shared loop body out of `AddCreature` into a private `appendCreatureGroup` helper both methods call.
- [x] 3.2 Add an `inject_encounter` WS message struct (`EncounterID string`) and dispatch case in `ws/handler.go`: fetch the encounter via `store.GlobalEncounters.GetEncounterByID`, verify `OwnerID` matches the connection's authenticated user, resolve each `EncounterMonster` (by `GetMonsterByName` or `GetCustomMonsterByID` per `IsCustom`), skip unresolvable groups, call `rm.InjectEncounter(...)`, then `rm.BroadcastState()` and `rm.MarkDirty()` once on success

## 4. Frontend types

- [x] 4.1 Add `Encounter` and `EncounterMonster` interfaces to `frontend/src/types.ts`, matching the backend JSON shape

## 5. Frontend — Encounter Builder screen

- [x] 5.1 Create `frontend/src/components/EncounterForm.tsx`, mirroring `MonsterForm.tsx`'s create/edit pattern: `useParams<{id?: string}>()`, `editing = Boolean(id)`, a `useEffect` that loads `GET /api/encounters/:id` in edit mode and populates state
- [x] 5.2 Add name field, edition selector, and a monster search input (reusing the debounced `GET /api/search/monsters?q=&edition=` pattern already in `DMView.tsx`'s `AddCreatureForm`)
- [x] 5.3 Add a staging list: selecting a search result appends a `{name, monster_id, is_custom, quantity, display_name}` group; each staged row has editable quantity and alias inputs, plus a remove control
- [x] 5.4 On submit, send `POST /api/encounters` (create) or `PUT /api/encounters/:id` (edit) with `{name, edition, monsters}`, then `navigate('/')`
- [x] 5.5 Add routes to `frontend/src/App.tsx`: `/encounters/new` and `/encounters/:id/edit`, both rendering `<EncounterForm />`

## 6. Frontend — Dashboard list

- [x] 6.1 In `Dashboard.tsx`, fetch `GET /api/encounters` on mount into a `myEncounters` state array, mirroring the existing `myMonsters` fetch
- [x] 6.2 Render a "My Encounters" list section (mirroring the "My Monsters" list: row per encounter with an Edit link to `/encounters/:id/edit` and a Delete button calling `DELETE /api/encounters/:id`), plus a "+ New Encounter" link to `/encounters/new`

## 7. Frontend — DM Panel injection

- [x] 7.1 In `DMView.tsx`, add an "Encounter Templates" control that fetches `GET /api/encounters?edition=<roomState.edition>` when opened
- [x] 7.2 Selecting an encounter sends `sendMessage({ type: 'inject_encounter', encounter_id: <id> })`

## 8. Verification

- [x] 8.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [x] 8.2 `go build ./...` from the repo root to confirm the backend compiles (also `go vet ./...` clean, and `go test ./...` shows only the same pre-existing unrelated `TestSnapshotConnectedStatus` failure from prior sessions)
- [x] 8.3 Manual check via dev server: create an encounter with an official + a custom monster group, verify it appears on the Dashboard, edit it, then open a room of the matching edition and inject it — confirm the right quantities/aliases spawn; delete the referenced custom monster and re-inject the same encounter, confirming that group is skipped and the rest still spawns

## 9. Spec sync

- [ ] 9.1 After merge, sync the new `encounter-repository`, `encounter-builder`, and `encounter-injection` capability specs into `openspec/specs/` (via `openspec-sync-specs` or archive flow)
