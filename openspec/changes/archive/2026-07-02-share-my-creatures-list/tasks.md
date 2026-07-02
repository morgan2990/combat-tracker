## 1. Extract shared layout-tier hook

- [x] 1.1 Create `frontend/src/hooks/useLayoutTier.ts`, moving `LayoutTier`, `TABLET_QUERY`, `DESKTOP_QUERY`, `computeLayoutTier`, and `useLayoutTier` out of `DMView.tsx` verbatim.
- [x] 1.2 Update `DMView.tsx` to import `useLayoutTier` (and `LayoutTier` if referenced) from the new module and remove the local definitions.
- [x] 1.3 Verify `DMView.tsx`'s tier-dependent rendering (phone/tablet/desktop columns, `showMyCreaturesInline`) is unchanged after the move.

## 2. Extract shared custom-monster row-list component

- [x] 2.1 Create `frontend/src/components/CustomMonsterList.tsx` exporting a component that takes `monsters: CustomMonster[]` and `onSelect: (monster: CustomMonster) => void`, rendering the existing heading, empty-state text, and `navItem`-styled rows currently inlined in `DMNavColumn.tsx`.
- [x] 2.2 Update `DMNavColumn.tsx` to render `CustomMonsterList` in place of its inlined "My Creatures" block, passing `onSelectCustomMonster` through as `onSelect`.
- [x] 2.3 Verify `DMNavColumn`'s rendered output and click behavior are unchanged after the extraction.

## 3. Make EncounterForm's quick-pick tier-aware

- [x] 3.1 Import `useLayoutTier` into `EncounterForm.tsx` and read the current tier.
- [x] 3.2 At the phone tier, keep rendering the existing pill-button markup for `myCreatures` unchanged.
- [x] 3.3 At the tablet and desktop tiers, render `CustomMonsterList` with `monsters={myCreatures}` and `onSelect={addCustomMonster}` in place of the pill markup.
- [x] 3.4 Confirm selecting a custom monster still appends it to the staging list the same way at every tier (no additional fetch, same staged fields).

## 4. Verify

- [x] 4.1 Manually test the Encounter Builder at phone width (<768px): pills render and clicking one adds it to the staging list.
- [x] 4.2 Manually test the Encounter Builder at tablet/desktop width (>=768px): row-list renders and clicking a row adds it to the staging list.
- [x] 4.3 Manually test DMView's in-room panel at phone and tablet/desktop widths to confirm no regression from the hook/component extraction.
- [x] 4.4 Run the frontend's existing lint/type-check/test commands and confirm they pass.
