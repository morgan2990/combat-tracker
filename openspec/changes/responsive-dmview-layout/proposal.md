## Why

DMView is a single centered column capped at 720px, so wide screens (tablets, laptops, monitors) sit mostly empty while two useful in-room actions — browsing saved encounters and browsing custom monsters — stay hidden behind a dropdown and a small inline form section. DMs on larger screens want persistent access to both lists, and a statblock panel that doesn't visually overlap the tracker.

## What Changes

- Add a responsive, width-based 3-tier layout for **DMView only** (phone / tablet / desktop). Breakpoints are pure viewport-width checks (CSS media queries / `ResizeObserver`) — no device or user-agent detection, since device class doesn't reliably predict available pixels.
- At the tablet and desktop tiers, add a persistent left "DM Nav" column listing the DM's saved encounters (click to inject) and custom monsters (click to add) — the same click behaviors as today's Encounter Templates dropdown and inline "My Creatures" quick-pick, just always visible instead of collapsed.
- At the desktop tier only, promote the statblock panel from a `position: fixed` viewport overlay (which can currently visually collide with the centered tracker on ~1000-1200px screens) into a real third grid column. Phone and tablet tiers keep today's overlay drawer behavior.
- Show a placeholder image in the statblock column when no creature is selected, so the column doesn't read as empty/broken (monster portraits are a possible future enhancement — out of scope here).
- Reduce the tracker column's max-width from 720px to ~580px so the desktop tier is reachable on common ~1366px-wide laptops, not just large monitors.
- Each column scrolls independently at the tablet/desktop tiers; the overall 3-column layout has a capped total width and centers with gutters on ultra-wide viewports rather than growing to fill available space.
- Phone tier (<768px) is functionally unchanged from today.

## Capabilities

### New Capabilities
- `dmview-responsive-layout`: The breakpoint tiers, column structure, independent per-column scroll, width caps/centering, and the statblock column's empty-state placeholder for DMView.

### Modified Capabilities
- `encounter-injection`: The "DM Panel Encounter Templates Dropdown" requirement gains tier-dependent placement — persistent list in the DM Nav column at tablet/desktop tiers, existing dropdown retained at the phone tier. Injection click behavior is unchanged.
- `initiative-ui`: The "Add Creature Form — My Creatures Quick-Pick" requirement gains tier-dependent placement — persistent list in the DM Nav column at tablet/desktop tiers, existing inline-in-form list retained at the phone tier. Selection behavior is unchanged.
- `statblock-drawer`: Gains tier-dependent presentation — real grid column at the desktop tier, existing fixed-position overlay drawer retained at phone/tablet tiers. Open/close and content-rendering behavior are unchanged; only the container changes.

## Impact

- `frontend/src/components/DMView.tsx` — main layout restructure, extraction of DM Nav column, tier-aware placement of `EncounterTemplatesControl` and the "My Creatures" quick-pick.
- `frontend/src/components/StatblockDrawer.tsx` — adapted to render as a grid column at the desktop tier instead of always rendering as a fixed overlay.
- New responsive/breakpoint handling (CSS and/or a small `ResizeObserver`/hook) — no existing responsive infrastructure in the codebase to build on.
- No backend or API changes — reuses existing `GET /api/encounters` and `GET /api/custom-monsters` endpoints and the existing `inject_encounter`/`add_creature` WS messages.
- No changes to Dashboard, PlayerView, or any route other than the live-room DM screen.
