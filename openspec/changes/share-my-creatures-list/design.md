## Context

`DMView.tsx` already defines viewport-width tier detection (`useLayoutTier`, phone < 768px / tablet 768-1300px / desktop >= 1300px) and uses it to swap its in-room "My Creatures" quick-pick between an inline pill row (phone) and `DMNavColumn`'s row-list (tablet/desktop). Both the hook and the row-list markup live only inside `DMView.tsx` / `DMNavColumn.tsx` today — neither is exported or reusable.

`EncounterForm.tsx` (the standalone Encounter Builder screen at `/encounters/new` and `/encounters/:id/edit`) has no tier logic at all and always renders its own separate pill markup for the same underlying data (`GET /api/custom-monsters?edition=...`, `CustomMonster[]`), regardless of viewport width.

## Goals / Non-Goals

**Goals:**
- Encounter Builder's "My Creatures" quick-pick matches DMView's tablet/desktop row-list presentation at the same widths, while keeping the existing pill presentation on phone.
- Tier-detection logic and row-list markup are defined once and reused by both `DMNavColumn` and `EncounterForm`, rather than being copied.

**Non-Goals:**
- No change to DMView's in-room "Add Creature" panel behavior — it already does the right thing.
- No change to the tier breakpoints themselves (768px / 1300px stay as defined in `dmview-responsive-layout`).
- No change to the "My Creatures" data source, fetch behavior, or the add-to-staging-list behavior — only the presentation of the list.
- Encounter Builder does not otherwise become a multi-column responsive layout; it remains a single-column form. Only the quick-pick section's internal rendering is tier-aware.

## Decisions

**Extract `useLayoutTier` into a shared hook module** (e.g. `frontend/src/hooks/useLayoutTier.ts`), exporting the hook plus the `LayoutTier` type. `DMView.tsx` imports it instead of defining it locally; `EncounterForm.tsx` imports the same hook.
- Alternative considered: re-implement a second, independent media-query check in `EncounterForm`. Rejected — duplicating breakpoint constants risks the two screens drifting out of sync over time.

**Extract the "My Creatures" row-list into a shared component** (e.g. `frontend/src/components/CustomMonsterList.tsx`), taking `monsters: CustomMonster[]` and `onSelect: (m: CustomMonster) => void` as props, rendering the existing heading, empty-state text, and `navItem`-styled rows. `DMNavColumn` renders it in place of its inlined block; `EncounterForm` renders it in place of its pill markup when the tier is tablet or desktop.
- Alternative considered: keep `DMNavColumn` untouched and copy its row styling into `EncounterForm` independently. Rejected — a shared component keeps a single source of truth for the row style and empty-state copy.
- The container/heading chrome differs between the two call sites (`DMNavColumn` wraps it in a fixed-height scrolling sidebar section; `EncounterForm` wraps it in a form field like its current pill section), so only the row-list itself (heading, empty state, rows) is shared — surrounding layout stays local to each component.

**Tier gating in `EncounterForm`**: reuse the same breakpoint split as `DMView` — phone (< 768px) keeps the existing pill buttons unchanged; tablet and desktop (>= 768px) render the shared row-list component. This mirrors `DMView`'s own phone-vs-rest split exactly, using the same 768px phone cutoff.

## Risks / Trade-offs

- [Extracting `useLayoutTier` out of `DMView.tsx` could subtly change its behavior if the extraction is not a pure move] → Keep the extraction mechanical (move the function bodies and constants verbatim into the new module); `DMView.tsx`'s own rendering and tests should be unaffected since the hook's return value and update behavior are unchanged.
- [Two call sites for the shared row-list component means its props/API needs to fit both a sidebar and a form-field context] → Keep the shared component narrowly scoped to just the list/rows (not the surrounding panel chrome), so each caller's layout differences stay outside the shared component.

## Migration Plan

Frontend-only, no data migration. Ship as a single change: extract hook, extract list component, wire `DMNavColumn` to the extracted component (behavior-neutral), then make `EncounterForm` tier-aware. No feature flag needed — this is a pure presentation change with no backend dependency.

## Open Questions

None outstanding.
