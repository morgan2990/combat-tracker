## 1. Backend entity model

- [x] 1.1 Add `DisplayName string \`json:"display_name,omitempty"\`` to the `Entity` struct in `room/room.go`
- [x] 1.2 Add `DisplayName string` to `addCreatureMsg` in `ws/handler.go` (json tag `display_name`)
- [x] 1.3 Update `AddCreature` in `room/room.go` to accept a `displayName string` param; set `Entity.DisplayName`, auto-numbering it per instance (`"{alias} {n}"`) when `quantity > 1` and `displayName != ""`, mirroring the existing `entityName` numbering logic
- [x] 1.4 Update the `add_creature` case in `ws/handler.go` to pass `msg.DisplayName` through to `rm.AddCreature`

## 2. Backend persistence

- [x] 2.1 Add `DisplayName string \`bson:"display_name,omitempty" json:"display_name,omitempty"\`` to `RoomEntitySnapshot` in `store/room.go`
- [x] 2.2 Update `snapshot()` in `room/room.go` to copy `e.DisplayName` into the `RoomEntitySnapshot`
- [x] 2.3 Update `inflateRoom()` in `room/room.go` to copy `DisplayName` back from the snapshot into the restored `Entity`

## 3. Backend live editing

- [x] 3.1 Add `DisplayName string` to `dmUpdateEntityMsg` in `ws/handler.go` (json tag `display_name`)
- [x] 3.2 Update `DMUpdateEntity` in `room/room.go` to accept `displayName string` and set `e.DisplayName = displayName` unconditionally for `type == "creature"` entities (no blank-check, since clearing to empty is a valid intentional action, unlike the existing `name` field's guard)
- [x] 3.3 Update the `dm_update_entity` case in `ws/handler.go` to pass `msg.DisplayName` through

## 4. Frontend types

- [x] 4.1 Add `display_name?: string` to the `Entity` interface in `frontend/src/types.ts`

## 5. Frontend — DM Panel

- [x] 5.1 In `AddCreatureForm` (`DMView.tsx`), add an optional "Custom Display Name / Alias (Optional)" text input alongside Name/Max HP/Qty; include `display_name` in the `add_creature` WS message only when non-blank (empty string otherwise, matching the backend's blank-means-no-alias handling)
- [x] 5.2 In `EntityRow`'s main row render, replace the bare `{entity.name}` span with a conditional: `"{entity.display_name} ({entity.name})"` when `entity.display_name` is non-empty, else `entity.name` alone
- [x] 5.3 In `EntityRow`'s expanded edit panel, add an "Alias" input next to the existing "Name" input (creature-only, matching the existing conditional), wired to a new `applyDisplayName` handler that calls `sendUpdate({ display_name: aliasInput.trim() })`
- [x] 5.4 Add `display_name: entity.display_name ?? ''` to the default field set in `EntityRow`'s `sendUpdate` function so unrelated updates (HP, conditions, etc.) don't clobber the current alias

## 6. Frontend — Player View

- [x] 6.1 In `PlayerView.tsx`'s initiative ladder row render, replace `{entity.name}` with `{entity.display_name || entity.name}`

## 7. Verification

- [x] 7.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [x] 7.2 `go build ./...` and `go test ./...` from the repo root to confirm the backend compiles and existing room tests still pass (`TestSnapshotConnectedStatus` fails identically on `master` before this change — pre-existing, uses `Type: "player"` instead of the real `"pc"` type — unrelated to this change, not fixed here)
- [x] 7.3 Manual check via dev server: add a batch of 3 creatures with an alias, confirm DM Panel shows numbered dual labels and Player View shows only numbered aliases; edit an alias post-creation and confirm it updates live for both views; clear an alias and confirm Player View falls back to the base name

## 8. Spec sync

- [ ] 8.1 After merge, sync the `entity-schema`, `room-persistence`, and `room-state` delta specs into `openspec/specs/` (via `openspec-sync-specs` or archive flow)
