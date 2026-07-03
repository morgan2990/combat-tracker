## Context

`Dashboard.tsx` renders two columns — "As DM" and "As Player" — each built from `CollapsibleSection`s. Three of those sections' header actions (`+ New Monster`, `+ New Encounter`, `+ New Character`) are plain `<Link>`s styled via `headerActionLink` (text, no background), while `+ New Room` uses `primaryBtn` (filled, bold, `color.success` green). The columns lay out with native `flex-wrap` and a `minWidth: 300` per column, which happens to reflow to a stacked column on narrow viewports today, but isn't an intentional tier-based decision the way `DMView`'s layout is.

The app already has a viewport-width tiering hook, `useLayoutTier()` (`phone` / `tablet` / `desktop`, thresholds at 768px / 1300px), used by `DMView` and `EncounterForm`. `DMView`'s phone tier renders a single stacked column — there is no existing precedent in this app for a phone-tier toggle/tab pattern.

## Goals / Non-Goals

**Goals:**
- All four bare single-click "create new X" header actions render as visually identical buttons.
- At phone-tier width, the DM/Player panels render one at a time behind a toggle, cutting scroll length.
- Tablet/desktop layout is pixel-for-pixel unchanged.

**Non-Goals:**
- Not touching `+ New Party`'s inline body button/input pairing.
- Not changing collapsible-section behavior (expand/collapse persistence, count display) beyond the header action's visual form.
- Not introducing the toggle pattern anywhere outside `Dashboard.tsx`.

## Decisions

- **Shared button style, not a copy of `primaryBtn`.** `primaryBtn(disabled)` carries `handleCreateRoom`'s async "Creating…" / disabled-dimming behavior, which doesn't apply to the Monster/Encounter/Character actions — those are plain route navigations via `<Link>`, not fetches. Introduce a `headerActionButton` style (same visual weight, padding, and `color.success` fill as `primaryBtn`'s enabled state) applied to those three `<Link>`s, so they read identically to `+ New Room` without inheriting create-in-flight state that doesn't exist for them.
- **Reuse the file's own segmented-toggle visual language for the DM/Player toggle.** The DM edition selector (`5e` / `5.5e`) already implements a two-option segmented toggle in this same file (`dmEdition` state, active/inactive button styling). The new "As DM" / "As Player" phone-tier toggle reuses that same visual pattern rather than introducing a new tab component, keeping the file internally consistent.
- **Tier gating via `useLayoutTier()`, `tier === 'phone'` only.** Tablet and desktop keep today's `flex-wrap` two-column render untouched; only the phone branch changes to render the toggle plus the single active panel.
- **Default active panel: "As DM", not persisted.** Matches the current visual order (DM column first) and avoids adding another `localStorage` key. Unlike per-section collapse state (which persists because losing it is annoying across many sections), this is a single binary toggle a user can flip in one tap — low cost to default it back to DM on every load.

## Risks / Trade-offs

- **New interaction pattern, no in-app precedent** → Scoped tightly to `Dashboard.tsx`'s phone tier only; if it feels inconsistent with `DMView`'s stacking approach once built, it's an isolated component to revert or reconcile later.
- **Losing at-a-glance visibility of both roles on phone** (e.g., glancing at Recent Rooms while primarily looking at DM tools) → toggle control stays persistently visible above the active panel, so switching is always one tap away, not buried in a menu.

## Open Questions

- Should the active panel persist across reloads (like section collapse state) if usage shows people always flip to the same side? Deferred — ship without persistence first.
