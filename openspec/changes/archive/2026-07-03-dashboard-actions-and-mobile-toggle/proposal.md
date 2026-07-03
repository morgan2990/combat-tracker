## Why

The Dashboard's per-section "create new" header actions look inconsistent — "+ New Room" is a filled button while "+ New Monster", "+ New Encounter", and "+ New Character" are plain text links — with no functional reason for the difference. Separately, on phone-width viewports the "As DM" and "As Player" columns stack into one long scroll (each holding 3-4 collapsible sections), which today looks fine but will get worse as more sections are added.

## What Changes

- The "+ New Monster", "+ New Encounter", and "+ New Character" header actions render as filled buttons, matching "+ New Room"'s style. "+ New Party" is unaffected — it's paired with a name input in the section body, not a bare header action.
- At phone-tier viewports, the Dashboard renders a toggle (segmented control) showing one panel — "As DM" or "As Player" — at a time, using the existing `useLayoutTier()` hook. Tablet and desktop tiers render both panels side by side.

## Capabilities

### New Capabilities
- `dashboard-responsive-layout`: Defines how the Dashboard's DM/Player panels adapt across viewport widths — phone tier renders a toggle showing one panel at a time; tablet/desktop tiers render both panels side by side (unchanged from today). Mirrors the existing `dmview-responsive-layout` capability's tiering approach.

### Modified Capabilities
- `dashboard-collapsible-sections`: The create-new action requirement for Characters/Monsters/Encounters/DM Rooms specifies a button (not a link) for each section.

## Impact

- `frontend/src/components/Dashboard.tsx`: header action styling for three sections; phone-tier toggle rendering for the DM/Player panels.
- `openspec/specs/dashboard-collapsible-sections/spec.md`: create-new action requirement specifies a button for all four sections.
- No backend changes.
