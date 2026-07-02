## 1. Breakpoint infrastructure

- [x] 1.1 Add a width-based tier hook/utility (e.g. `useLayoutTier` via `ResizeObserver` or `matchMedia`) returning `'phone' | 'tablet' | 'desktop'` for `<768px`, `768-~1320px`, `>=~1320px` — no `navigator.userAgent` or touch-capability checks
- [x] 1.2 Wire the tier into `DMView.tsx` and confirm it updates live on browser window resize (dev-tool check, not just initial load)

## 2. DM Nav column

- [x] 2.1 Extract a `DMNavColumn` component that fetches `GET /api/encounters?edition=<room's edition>` and `GET /api/custom-monsters?edition=<room's edition>` on mount and renders both lists persistently (no toggle/click-to-open)
- [x] 2.2 Wire encounter selection in `DMNavColumn` to send `inject_encounter` exactly as `EncounterTemplatesControl` does today
- [x] 2.3 Wire custom monster selection in `DMNavColumn` to populate the Add Creature form's `name`, `max_hp`, and statblock-reference fields exactly as the existing inline "My Creatures" quick-pick does today
- [x] 2.4 Render `DMNavColumn` only at tablet/desktop tiers; leave phone tier's existing `EncounterTemplatesControl` dropdown and inline "My Creatures" section in `AddCreatureForm` untouched

## 3. Tracker column

- [x] 3.1 Reduce the Tracker column's max-width from 720px to ~580px, applied only at tablet/desktop tiers (phone tier keeps its current single-column width behavior)
- [x] 3.2 Confirm entity rows (name/HP/AC/initiative) remain readable at the narrower width; adjust the ~580px figure if real content forces it

## 4. Statblock column (desktop tier)

- [x] 4.1 Add a placeholder image asset for the Statblock column's empty state
- [x] 4.2 Adapt `StatblockDrawer` (or add a sibling component) so that at the desktop tier the open creature's statblock renders as the Statblock column's content instead of a fixed-overlay drawer, while preserving lazy-loading (no fetch until opened) and single-open-at-a-time behavior
- [x] 4.3 At phone/tablet tiers, keep `StatblockDrawer` rendering exactly as today (fixed-position overlay, 420px/95vw)
- [x] 4.4 Show the placeholder image in the Statblock column when no creature's statblock is open at desktop tier; swap to statblock content on open and back to placeholder on close

## 5. Layout shell, scroll, and width capping

- [x] 5.1 Build the tablet-tier two-column (DM Nav | Tracker) and desktop-tier three-column (DM Nav | Tracker | Statblock) grid/flex containers in `DMView.tsx`
- [x] 5.2 Give the tablet/desktop container a bounded height (e.g. `100vh` minus header chrome) so each column can scroll independently; verify phone tier still scrolls as a single page
- [x] 5.3 Cap the overall multi-column layout's total width and center it with gutters at very wide viewports (e.g. 1920px), rather than letting columns grow to fill space
- [x] 5.4 Verify no visual regression to the phone-tier layout (should be pixel-equivalent to today's behavior)

## 6. Verification

- [x] 6.1 Manually test at representative widths: <768px (phone), ~900px (tablet), ~1366px (common laptop, should reach desktop tier), ~1920px (large/tablet monitor)
- [x] 6.2 Manually test resizing a browser window live across each threshold to confirm no layout breakage or state loss (e.g. an open statblock surviving a tier crossing)
- [x] 6.3 Confirm encounter injection and custom-monster quick-add produce identical results whether triggered from the phone-tier controls or the tablet/desktop DM Nav column
