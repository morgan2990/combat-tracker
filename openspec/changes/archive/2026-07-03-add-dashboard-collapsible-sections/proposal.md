## Why

The Dashboard's two columns ("As DM" and "As Player") each render as one undifferentiated block: sections like "My Characters," "My Parties," and "Recent Rooms" are distinguished only by a small muted label with no border or divider, while nested Party cards *do* get their own bordered box — making the layout feel visually inconsistent and hard to scan. Nothing on the Dashboard is collapsible, so as a user's characters, parties, and rooms accumulate, each column becomes one long uninterrupted scroll with no way to focus on just one grouping.

## What Changes

- A new shared `CollapsibleSection` component: a header row (title, a count badge, an optional primary action such as "+ New Room" that stays visible even when the section is collapsed, and a collapse/expand chevron) followed by a divider and the section's collapsible body.
- Applied to all six existing Dashboard groupings — Player column: Characters, Parties, Rooms; DM column: Rooms, Monsters, Encounters — replacing today's plain muted-label headers.
- Collapse/expand state persists per section via `localStorage`, keyed independently per section, defaulting to expanded so nothing changes visually for an existing user until they actively collapse something.
- The "+ New Encounter" action moves from below the encounters list into the Encounters section's header row, so it stays reachable even when that section is collapsed — the same treatment as the other five sections' primary actions.
- The DM column's edition selector (5e / 5.5e), which today sits inline with "+ New Room" inside what becomes the Rooms section, moves out to a standalone control above the DM column's sections — so it stays visible and usable regardless of whether the Rooms section is collapsed, and reads as DM-column-level configuration rather than one Rooms-section control among several.
- No other existing Dashboard behavior changes: data fetching, and party/room/character/encounter/monster creation, editing, and deletion all work exactly as they do today. This is a structural and presentational regrouping of already-existing content, not a behavior change to any of it.

## Capabilities

### New Capabilities
- `dashboard-collapsible-sections`: the shared `CollapsibleSection` component (header with title/count/action/chevron, persisted collapse state), and which six Dashboard groupings use it.

### Modified Capabilities
- `encounter-builder`: the "Dashboard Encounters List" requirement's "+ New Encounter link ... SHALL appear below the list" changes to reflect the link living in the Encounters section's header instead.

## Impact

- **Frontend**: `Dashboard.tsx` is restructured to wrap its six existing groupings in the new component instead of plain labeled `<div>`s; a new `CollapsibleSection.tsx` component.
- **No backend changes**: this is presentation-only; no new API endpoints, no schema changes.
- **No new dependencies**: collapse persistence uses the browser's existing `localStorage`, nothing new to install.
