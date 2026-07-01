## 1. Backend — entity creation and sorting

- [x] 1.1 Add `AddLairAction(sessionID string) error` to `room/room.go`, mirroring `RemoveDeadCreatures`'s no-payload/ownership-check pattern: append an `Entity{Name: "Lair Action", Type: "lair_action", Initiative: <ptr to 20>, MaxHP: 0, CurrentHP: 0, Conditions: []string{}, IsHidden: true}`, then call `r.sortEntities()`
- [x] 1.2 In `sortEntities()`'s comparator in `room/room.go`, add the tie-break rule ahead of the numeric comparison: when `*a == *b` and exactly one of the two entities has `Type == "lair_action"`, that entity sorts after the other

## 2. Backend — WS dispatch

- [x] 2.1 Add an `add_lair_action` case to the WS message dispatch in `ws/handler.go` (no message struct/unmarshal needed) that calls `rm.AddLairAction(c.SessionID)`, then `rm.BroadcastState()` and `rm.MarkDirty()` on success

## 3. Frontend types

- [x] 3.1 Widen `Entity.type` in `frontend/src/types.ts` from `'pc' | 'creature' | 'companion'` to `'pc' | 'creature' | 'companion' | 'lair_action'`

## 4. Frontend — DM Panel

- [x] 4.1 Add a `+ Add Lair Action` button to the combat-controls row in `DMView.tsx` (outside the `is_started` conditional branch, so it renders in both states), dispatching `sendMessage({ type: 'add_lair_action' })`
- [x] 4.2 In `EntityRow`, widen the visibility-toggle condition from `entity.type === 'creature'` to `(entity.type === 'creature' || entity.type === 'lair_action')`
- [x] 4.3 In `EntityRow`'s expanded panel, widen the Name editor's condition (`entity.type === 'creature'`) to also include `lair_action`
- [x] 4.4 In `EntityRow`'s expanded panel, widen the Alias editor's condition (`entity.type === 'creature'`) to also include `lair_action`
- [x] 4.5 In `EntityRow`, compute `vitalState` as a fixed non-dead/non-unconscious value for `type === 'lair_action'` (short-circuit before calling `entityVitalState`, mirroring how HP masking already special-cases entity type elsewhere), so no Dead/Unconscious badge is possible for a HP-less entity
- [x] 4.6 In `EntityRow`'s main row, wrap the `{entity.current_hp}/{entity.max_hp} HP` + temp_hp display in `entity.type !== 'lair_action' &&`
- [x] 4.7 In `EntityRow`'s expanded panel, wrap the HP smart-input editor block in `entity.type !== 'lair_action' &&`
- [x] 4.8 In `EntityRow`'s expanded panel, wrap the condition-toggle row in `entity.type !== 'lair_action' &&`
- [x] 4.9 In `EntityRow`'s expanded panel action buttons, wrap the Kill/Revive button in `entity.type !== 'lair_action' &&` (Remove stays unconditional)

## 5. Frontend — Player View

- [x] 5.1 In `PlayerView.tsx`'s row renderer, extend the existing `isCreature`-based vital-state short-circuit (`const vitalState = isCreature ? 'alive' : entityVitalState(...)`) to also cover `type === 'lair_action'`
- [x] 5.2 In `PlayerView.tsx`'s row renderer, wrap the HP display block (the `isCreature ? hpLabel(...) : <span>{current_hp}/{max_hp} HP</span>` block) so a `lair_action` row renders neither branch — no HP text at all
- [x] 5.3 In `PlayerView.tsx`'s row renderer, ensure the conditions line (`entity.conditions.length > 0 && ...`) does not render for `type === 'lair_action'` (conditions are always empty for this type per task 1.1, so this is naturally a no-op, but confirm no dead code path renders an empty div) — confirmed: `myEntity`/`myCompanions` lookups only ever match `pc`/`companion` types, so `EntityEditor` (the only other conditions-rendering path) never receives a `lair_action` entity

## 6. Verification

- [x] 6.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [x] 6.2 `go build ./...` from the repo root to confirm the backend compiles (also ran `go test ./...`: same pre-existing `TestSnapshotConnectedStatus` failure as prior sessions, unrelated to this change)
- [x] 6.3 Manual check via dev server: add a lair action, confirm it's invisible to a connected player until the DM toggles it visible; confirm its row in the DM Panel has no HP/condition/Kill-Revive UI but does have Remove, initiative, name/alias editors, and the visibility toggle; add a creature that also rolls/sets initiative 20 and confirm the lair action sorts after it regardless of which was added first

## 7. Spec sync

- [ ] 7.1 After merge, sync the new `lair-actions` capability and the `room-state` delta spec into `openspec/specs/` (via `openspec-sync-specs` or archive flow)
