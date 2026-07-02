## 1. Extract EntityRow

- [x] 1.1 Create `frontend/src/components/EntityRow.tsx` with the `EntityRow` component, its `EntityRowProps` interface, `parseHP` helper, and `actionBtn` style constant, moved verbatim.
- [x] 1.2 Import `EntityRow` into `DMView.tsx`, remove the inline definition.

## 2. Extract AddCreatureForm

- [x] 2.1 Create `frontend/src/components/AddCreatureForm.tsx` with the `AddCreatureForm` `forwardRef` component, `AddCreatureFormProps`/`MonsterRef` interfaces, the exported `AddCreatureFormHandle` interface, `SEARCH_MIN_CHARS`/`SEARCH_DEBOUNCE_MS` constants, and `fieldStyle`, moved verbatim.
- [x] 2.2 Import `AddCreatureForm` and the `AddCreatureFormHandle` type into `DMView.tsx`, remove the inline definition.

## 3. Extract EncounterTemplatesControl

- [x] 3.1 Create `frontend/src/components/EncounterTemplatesControl.tsx` with the component and its props interface, moved verbatim — including its existing fetch logic, unchanged (its retry-on-network-error-only behavior is tracked separately by GitHub issue #7, not touched here).
- [x] 3.2 Import `EncounterTemplatesControl` into `DMView.tsx`, remove the inline definition.

## 4. Verify

- [x] 4.1 `DMView.tsx` reduced to the layout shell only (imports + `DMViewProps` + column-sizing constants + the `DMView` function); confirmed no other file imports `AddCreatureFormHandle` (or anything else) from `DMView.tsx` directly.
- [x] 4.2 `tsc -b` and `oxlint` clean (same pre-existing warnings as before, no new ones).
- [x] 4.3 Manually verified via Playwright, desktop and phone tiers: added a creature via search (`AddCreatureForm`), selected a custom monster from the `DMNavColumn` row-list and submitted it (`AddCreatureForm`'s `selectCustomMonster` via ref), expanded an `EntityRow`, toggled a condition, killed it (dead-state row rendered correctly), and opened `EncounterTemplatesControl`'s dropdown on phone tier (listed a saved encounter template correctly).
