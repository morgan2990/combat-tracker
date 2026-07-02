## Context

DMView (`frontend/src/components/DMView.tsx`) currently renders as a single `<div style={{ maxWidth: 720, margin: '0 auto' }}>` — header, combat controls, Add Creature form (with an inline "My Creatures" quick-pick), and the initiative tracker, all stacked vertically. There is no responsive infrastructure in the codebase: the only `@media` queries in `App.css` are unused Vite scaffolding.

Two DM actions are already wired up but visually collapsed:
- `EncounterTemplatesControl` (DMView.tsx ~line 537): a button that opens a small absolutely-positioned popover listing saved encounters; clicking one sends `inject_encounter`.
- The "My Creatures" quick-pick inside `AddCreatureForm` (DMView.tsx ~line 424): an inline searchable list of the DM's custom monsters; clicking one populates the add-creature form.

`StatblockDrawer` (`frontend/src/components/StatblockDrawer.tsx`) is `position: fixed`, docked to the *viewport's* right edge at 420px wide (95vw on narrow screens). Because the main column is independently centered, the two aren't coordinated — on real laptop widths (~1000-1200px) the drawer can overlap the tracker.

This design adds a width-based responsive layout to DMView that turns the two collapsed interactions into a persistent column and gives the statblock panel a real slot in the layout at wide-enough widths.

## Goals / Non-Goals

**Goals:**
- Give DMs on tablet/desktop-width screens persistent, always-visible access to their saved encounters and custom monsters, without changing what clicking them does.
- Give the statblock panel a real layout slot at desktop widths so it no longer overlaps the tracker.
- Reach the desktop tier on common ~1366px-wide laptops, not just large monitors.
- Keep phone-width behavior (the current implementation) completely unchanged.

**Non-Goals:**
- Dashboard or PlayerView responsiveness — out of scope, may be a future change.
- Any change to click/selection semantics for encounters or custom monsters.
- Monster portrait images in the statblock empty state — placeholder only for now.
- Device/input-type detection (touch vs. mouse) — DMView currently has no hover-only interactions, so this isn't needed alongside width-based tiers.

## Decisions

### Width-based breakpoints, not device detection
Tiers are determined purely by viewport width (CSS media queries and/or a `ResizeObserver`-backed hook), never by `navigator.userAgent` or touch-capability sniffing.

Rationale: device class doesn't reliably predict available pixels — a resized desktop browser window can be phone-narrow, and a tablet can have desktop-class resolution (the motivating case: a 1920×1080 tablet should get the full desktop layout, which a UA check keyed to "tablet" would get wrong). UA strings are also an increasingly unreliable signal as browsers reduce/freeze them. Width answers the only question that actually matters for a column layout: does the content fit.

Alternative considered: `pointer`/`hover` media features for touch-specific affordances. Rejected for this change — grepping DMView found no hover-only interactions today, so there's nothing for an input-type check to protect against yet. Can be added later if a hover-gated interaction is introduced.

### Three tiers, progressive column count
- **Phone (`<768px`)**: unchanged today's layout — single column, statblock overlay drawer, Encounter Templates dropdown, inline My Creatures list.
- **Tablet (`768px`–`~1320px`)**: two columns — DM Nav | Tracker. Statblock remains an overlay drawer; not enough width for a third column without cramping.
- **Desktop (`>~1320px`)**: three columns — DM Nav | Tracker | Statblock. Drawer overlay retired in favor of a real grid column.

Alternative considered: a hard two-tier cutoff (today's layout below a threshold, full 3-column above it). Rejected because it gives tablet-width screens — which do have meaningfully more room than a phone — no improvement at all, even though a persistent nav column fits comfortably by itself before there's room for a third column.

### Desktop threshold tuned for laptops, not just monitors
Approximate column widths: DM Nav ~240px, Tracker ~580px (see below), Statblock ~420px, plus gaps/padding ≈ 1300-1350px. The threshold is set here, rather than higher (~1450px+), specifically so that ~1366px-wide laptops — one of the most common laptop resolutions — reach the desktop tier instead of being stuck at two columns.

### Tracker max-width reduced from 720px to ~580px
Direct consequence of the above: fitting three real columns at ~1320px instead of ~1450px+ requires shrinking the tracker's cap. Entity rows (name/HP/AC/initiative) don't need 720px to stay readable.

### Statblock column: real grid column at desktop, unchanged overlay below it
Rather than making `StatblockDrawer` a grid column at all tiers, it keeps its current fixed-overlay behavior at phone/tablet and only becomes a true column at desktop width. This avoids reserving ~420px of a two-column tablet layout for a panel that would rarely have enough surrounding room to justify losing that space from the tracker/nav columns.

Empty state: when no creature's statblock is open at the desktop tier, the column shows a placeholder image rather than collapsing to zero width (which would make the layout jump between 2 and 3 columns as the DM opens/closes statblocks) or sitting blank (which reads as broken). Portraits are a plausible future replacement for the placeholder but are not built here.

### Independent per-column scroll; capped total width
Each column (DM Nav, Tracker, Statblock) scrolls on its own axis within a fixed-height shell at the tablet/desktop tiers, so a long initiative tracker doesn't push the DM Nav list or an open statblock out of view. This requires DMView's tablet/desktop container to establish a bounded height (e.g. `height: 100vh` minus header chrome) rather than letting the whole page scroll as it does today.

The three-column block has a capped total width and centers with gutters on ultra-wide viewports (e.g. the 1920px tablet) rather than growing to fill available space — avoids stretching entity rows or the nav list to uncomfortable widths on very wide screens. Phone-tier scrolling is unaffected (whole-page scroll, as today).

## Risks / Trade-offs

- **Fixed-height shell changes scroll ergonomics at tablet/desktop.** → Mitigate by keeping phone tier (the primary, best-tested experience today) on the existing whole-page-scroll behavior; only tablet/desktop opt into the bounded-height layout.
- **Narrower tracker (580px vs. 720px) could feel cramped if more columns/data are added to entity rows later.** → Acceptable trade-off now; revisit the cap if a future change adds tracker row content.
- **Two places (DM Nav column at tablet/desktop, existing dropdown/inline list at phone) now render the same encounter/monster lists.** → Both read from the same existing endpoints and click handlers; no new state duplication, just conditional placement, so behavior can't drift between tiers.
- **Breakpoint thresholds (768px, ~1320px) are estimates from approximate column widths, not measured against real content.** → Treat as a starting point; adjust during implementation if actual rendered column widths (longest encounter/monster names, real entity row content) push the numbers.

