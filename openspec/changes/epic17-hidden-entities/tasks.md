## 1. Backend entity model

- [x] 1.1 Add `IsHidden bool \`json:"is_hidden"\`` to the `Entity` struct in `room/room.go` (no `omitempty` — always serialized)
- [x] 1.2 Add a `toggleEntityVisibilityMsg` struct to `ws/handler.go`, mirroring `removeEntityMsg`'s `{EntityID string \`json:"entity_id"\`}` shape
- [x] 1.3 Add a `ToggleEntityVisibility(sessionID, entityID string) error` method to `room/room.go`, mirroring `RemoveEntity`'s ownership-check/entity-lookup pattern, flipping `IsHidden` on the matched entity
- [x] 1.4 Add a `toggle_entity_visibility` case to the WS dispatch in `ws/handler.go` that calls `rm.ToggleEntityVisibility`, then `rm.BroadcastState()` and `rm.MarkDirty()` on success (matching the `remove_entity` case's pattern — dirty-mark only, not an immediate persist)

## 2. Backend persistence

- [x] 2.1 Add `IsHidden bool \`bson:"is_hidden" json:"is_hidden"\`` to `RoomEntitySnapshot` in `store/room.go` (no `omitempty` — always serialized)
- [x] 2.2 Update `snapshot()` in `room/room.go` to copy `e.IsHidden` into the `RoomEntitySnapshot`
- [x] 2.3 Update `inflateRoom()` in `room/room.go` to copy `IsHidden` back from the snapshot into the restored `Entity`

## 3. Frontend types

- [x] 3.1 Add `is_hidden: boolean` to the `Entity` interface in `frontend/src/types.ts`

## 4. Frontend — DM Panel

- [x] 4.1 In `EntityRow` (`DMView.tsx`), add a 👁/🙈 toggle button on creature rows (only), positioned near the existing 📋 statblock button, dispatching `sendMessage({ type: 'toggle_entity_visibility', entity_id: entity.id })` on click
- [x] 4.2 Apply `opacity: 0.5` (additive to the existing `vitalState`-driven `rowBg`/`textColor` logic, not a replacement) to the row wrapper when `entity.is_hidden` is true

## 5. Frontend — Player View

- [x] 5.1 In `PlayerView.tsx`, extend the existing `visibleEntities` filter to `entities.filter(e => (is_started || e.type !== 'creature') && !e.is_hidden)`

## 6. Verification

- [x] 6.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [x] 6.2 `go build ./...` from the repo root to confirm the backend compiles
- [ ] 6.3 Manual check via dev server: hide a creature mid-combat and confirm it disappears from Player View but stays visible (dimmed) in DM Panel; reveal it and confirm it reappears in Player View; confirm hiding a creature pre-combat has no additional visible effect (already hidden by the Epic 15 staging filter)

## 7. Spec sync

- [ ] 7.1 After merge, sync the `entity-schema`, `room-persistence`, and `room-state` delta specs into `openspec/specs/` (via `openspec-sync-specs` or archive flow)
