## 1. Backend — edition filter on custom monster listing

- [x] 1.1 Update `ListCustomMonstersByOwner` in `store/custom_monster.go` to accept an additional `edition string` parameter, adding `edition` to the Mongo filter only when non-empty (mirroring `ListEncountersByOwner`'s pattern)
- [x] 1.2 Update `ListMyCustomMonsters` in `api/handler.go` to read `r.URL.Query().Get("edition")` and pass it through

## 2. Frontend — Add Creature Form quick-pick

- [x] 2.1 In `DMView.tsx`'s `AddCreatureForm`, add a `myCreatures` state array and a `useEffect` (keyed on `edition`) that fetches `GET /api/custom-monsters?edition=<edition>` on mount/edition-change
- [x] 2.2 Add a `selectCustomMonster(m: CustomMonster)` handler that sets `name`, `maxHP`, and `monsterRef` directly from the fetched document (no follow-up fetch, unlike `selectMonster`)
- [x] 2.3 Render a "My Creatures" section (chip list: name + HP, styled consistently with the search dropdown) above the search input, calling `selectCustomMonster` on click

## 3. Frontend — Encounter Builder quick-pick

- [x] 3.1 In `EncounterForm.tsx`, add a `myCreatures` state array and a `useEffect` (keyed on `edition`) that fetches `GET /api/custom-monsters?edition=<edition>`
- [x] 3.2 Add an `addCustomMonster(m: CustomMonster)` handler that appends a staged group `{name: m.name, monster_id: m.id, is_custom: true, quantity: 1, display_name: ''}` directly from the fetched document
- [x] 3.3 Render the same "My Creatures" section above the search input, calling `addCustomMonster` on click

## 4. Verification

- [x] 4.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [x] 4.2 `go build ./...` from the repo root to confirm the backend compiles (also `go test ./...`: only the same pre-existing unrelated `TestSnapshotConnectedStatus` failure from prior sessions)
- [ ] 4.3 Manual check via dev server: create 2 custom monsters in different editions, confirm the Add Creature form and Encounter Builder each show only the current-edition one in "My Creatures", confirm clicking one populates/stages it correctly, and confirm search still works unaffected

## 5. Spec sync

- [ ] 5.1 After merge, sync the `monster-repository`, `initiative-ui`, and `encounter-builder` delta specs into `openspec/specs/` (via `openspec-sync-specs` or archive flow)
