## Context

`Dashboard.tsx` renders two flex columns ("As DM," "As Player"), each a single bordered `panelStyle` box containing several groupings — My Rooms, My Monsters, My Encounters on the DM side; My Characters, My Parties, Recent Rooms/Join a new room on the Player side. Each grouping is introduced only by a small muted-color label (`fontSize:13, color:color.textSecondary`) with no border or divider, while the individual `PartyCard`s nested inside "My Parties" *do* get their own bordered box — the one place in either column that looks like a "card." Nothing on the Dashboard is collapsible.

A consistent section-header convention already exists elsewhere in the app (`sectionHeading` in `CustomMonsterList.tsx`, reused by `DMNavColumn.tsx`; equivalently named `sectionLabel` in `InventoryPanel.tsx`/`PartyInventoryPanel.tsx`): uppercase, letter-spaced, bold, secondary-colored text. None of these existing instances collapse — they're static labels in an already-scrollable side panel. This change extends that established typographic language with a collapse affordance rather than introducing a new visual system.

## Goals / Non-Goals

**Goals:**
- Make each of the six existing Dashboard groupings (Characters, Parties, Rooms on the Player side; Rooms, Monsters, Encounters on the DM side) visually distinct via a real divider and a stronger header, consistent with the app's existing section-header typography.
- Let each of the six sections collapse/expand independently, persisted across reloads.
- Keep each section's primary single-click "create new" action reachable even when that section is collapsed.
- Change nothing about how any of the underlying data (characters, parties, rooms, monsters, encounters) is fetched, created, edited, or deleted.

**Non-Goals:**
- No change to `Dashboard.tsx`'s overall two-column structure, its `maxWidth: 760` container, or making it tier-aware via `useLayoutTier` — that's a separate, larger concern (noted as a possible future change, not this one).
- No server-side persistence of collapse state (no new user-preference field, no API call) — `localStorage` only.
- No change to `PartyCard`'s own per-party bordered styling — parties remain individually-bordered cards *inside* the collapsible "Parties" section, same as today.

## Decisions

**1. One new shared component, `CollapsibleSection`, used six times.**
Rather than hand-rolling collapse state and a header in `Dashboard.tsx` six times over, one component takes `id` (a stable slug for the persistence key), `title`, `count`, an optional `headerAction` (rendered next to the chevron, visible whether expanded or collapsed), and `children` (the collapsible body). This mirrors the codebase's existing pattern of small focused components (`ItemNameSearchField`, `CurrencyEditor`) rather than duplicating markup per section.

**2. The header action slot is reserved for simple, single-interaction actions only — not forms.**
Of the six sections' "create new" affordances, three are a bare link/button with no required accompanying input: "+ New Character," "+ New Monster," "+ New Encounter" (and, once Decision 4 below extracts the edition selector, "+ New Room" joins this group). These move into the header via `headerAction` and stay reachable when collapsed. The other two — "+ New Party" (requires a name typed into an adjacent input first) and "Join a new room" / "Join a party" (each a small multi-field form: PC select + code input + submit) — are not compact enough for a header row and stay inside the collapsible body like everything else in that section. Collapsing "Parties" or "Rooms" (Player side) hides those forms same as it hides the lists; this is an accepted, consistent trade-off rather than special-casing which forms get a header shortcut.

**3. Collapse state persists via `localStorage`, one boolean per section, keyed `dashboard.section.<column>.<section>`.**
E.g. `dashboard.section.player.characters`, `dashboard.section.dm.rooms`. Read once on mount via `useState`'s lazy initializer (`localStorage.getItem(key) !== 'false'`, so anything except the literal string `"false"` — including a missing key — defaults to expanded); written on every toggle. This is the Dashboard's first use of `localStorage` anywhere in the app, but it's a plain browser built-in — no new dependency, no server round-trip, and per-section keys mean one section's state can never clobber another's.

**4. The DM column's edition selector (5e/5.5e) is extracted out of the Rooms section entirely, to a standalone control above the DM column's sections.**
Today it sits inline with "+ New Room" (the two are visually paired but not semantically position-locked by any spec requirement — `room-creation`'s requirement only says the Dashboard "SHALL present an edition selector ... and a '+ New Room' action," with no adjacency requirement). Moving it out means it stays visible and usable regardless of whether Rooms is collapsed, and reads more accurately as DM-column-level configuration (it's `dmEdition` state already lifted to `Dashboard`, used only by `handleCreateRoom`) rather than one control bundled into a single section among several. `handleCreateRoom` and the now-header-only "+ New Room" button both continue reading the same lifted `dmEdition` state — no behavior change to room creation itself.

**5. `encounter-builder`'s "+ New Encounter ... SHALL appear below the list" requirement is updated, not preserved as a contradiction.**
Since Decision 2 moves that link into the Encounters section's header, the existing spec text is factually wrong under the new layout. The delta spec updates this requirement's wording to describe the header placement instead of introducing a second, conflicting position.

**6. Count badges are derived from data already fetched for each section — no new requests.**
Characters: `me.pcs.length`. Parties: `parties.length`. DM Rooms: `me.rooms.length`. Monsters: `myMonsters.length`. Encounters: `myEncounters.length`. Player Rooms: `me.recent_rooms.length`. All are already in component state/props today; the badge is a pure render-time count, not a new fetch.

## Risks / Trade-offs

- **[Risk] A returning user's `localStorage` is empty (first visit after this ships), so every section defaults expanded** — meaning nothing visually changes for existing habits, only the option to collapse is added. → **Mitigation**: this is intentional (see Goals) — a silent behavior change on first load (e.g. auto-collapsing rarely-used sections) would be more surprising than defaulting open.
- **[Risk] Hiding "Join a new room"/"Join a party"/"+ New Party" behind a collapsed section reduces their discoverability** compared to always-visible controls today. → **Mitigation**: these are inherently occasional actions (joining a specific room/party by code is not a habitual, every-visit action the way viewing your characters or opening a room is), and the section defaults to expanded — a user only loses quick access after deliberately collapsing that specific section themselves.
- **[Risk] `localStorage` can be unavailable or throw** (private browsing modes in some older browsers, storage quota exceeded — vanishingly unlikely for a handful of boolean keys, but not impossible). → **Mitigation**: wrap reads/writes in `try/catch`; a failure just means collapse state doesn't persist for that session, falling back to the default-expanded behavior — never a hard error.

## Migration Plan

1. Ship `CollapsibleSection` and the restructured `Dashboard.tsx` together — there's no partial-rollout concern since this is a single client-rendered page with no server dependency.
2. No data migration: `localStorage` starts empty for every user, which is exactly the "default expanded" state described above.
3. Rollback: revert the deploy. No persisted server data is touched; any `localStorage` keys already written by users are simply ignored by the reverted code and orphaned harmlessly in their browsers.

## Open Questions

None outstanding.
