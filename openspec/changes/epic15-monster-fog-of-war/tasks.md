## 1. PlayerView staging filter

- [x] 1.1 In `frontend/src/components/PlayerView.tsx`, derive `visibleEntities` from `entities` and `is_started` (all entities when started, `type !== 'creature'` otherwise) above the existing initiative ladder render
- [x] 1.2 Replace the `entities.map(...)` in the initiative ladder with `visibleEntities.map(...)`
- [x] 1.3 Update the empty-state check: keep "No combatants yet." when `entities.length === 0`; add a new staging placeholder message when `visibleEntities.length === 0 && entities.length > 0`

## 2. Verification

- [x] 2.1 `npx tsc --noEmit` in `frontend/` to confirm no type errors
- [ ] 2.2 Manual check via the `/run` skill or dev server: DM adds creatures pre-combat, confirm player view hides them and shows the staging placeholder; confirm DM Panel still shows them; start combat and confirm player view instantly reveals the full list

## 3. Spec sync

- [ ] 3.1 After merge, sync the `room-state` delta spec into `openspec/specs/room-state/spec.md` (via `openspec-sync-specs` or archive flow)
