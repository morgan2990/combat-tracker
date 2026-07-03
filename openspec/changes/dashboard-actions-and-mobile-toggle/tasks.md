## 1. Header action button consistency

- [x] 1.1 Add a `headerActionButton` style constant in `Dashboard.tsx` matching `primaryBtn`'s enabled visual (fill, padding, font weight) without the create-in-flight/disabled logic.
- [x] 1.2 Apply `headerActionButton` to the "+ New Monster" `<Link>` (My Monsters section header).
- [x] 1.3 Apply `headerActionButton` to the "+ New Encounter" `<Link>` (My Encounters section header).
- [x] 1.4 Apply `headerActionButton` to the "+ New Character" `<Link>` (My Characters section header).
- [x] 1.5 Confirm "+ New Room" and "+ New Party" are visually unchanged.

## 2. Phone-tier DM/Player toggle

- [x] 2.1 Import and call `useLayoutTier()` in `Dashboard.tsx`.
- [x] 2.2 Add local state for the selected panel (`'dm' | 'player'`), defaulting to `'dm'`, not persisted.
- [x] 2.3 Build the toggle control reusing the file's existing segmented-toggle visual pattern (same shape as the `dmEdition` 5e/5.5e selector).
- [x] 2.4 At `tier === 'phone'`, render the toggle control followed by only the selected panel (DM or Player), replacing the stacked two-column render for this tier.
- [x] 2.5 At `tier === 'tablet'` or `'desktop'`, keep the existing side-by-side `flex-wrap` two-column render, with no toggle control.
- [x] 2.6 Verify the toggle stays visible and usable in both selected states, and that switching is a single tap with no scroll required to reach it.

## 3. Spec sync

- [x] 3.1 Verify `dashboard-collapsible-sections`'s create-new action requirement specifies a button for all four sections, as implemented.
- [x] 3.2 Verify `dashboard-responsive-layout` scenarios (phone toggle, tablet/desktop unaffected) pass manually at each viewport tier.

## 4. Manual verification

- [x] 4.1 At each viewport tier (phone <768px, tablet 768-1300px, desktop ≥1300px), confirm layout matches spec.
- [x] 4.2 Confirm all four header create actions (Room, Monster, Encounter, Character) are visually identical buttons.
- [x] 4.3 Confirm collapsing a section still leaves its header button visible and usable (regression check against existing `dashboard-collapsible-sections` behavior).
