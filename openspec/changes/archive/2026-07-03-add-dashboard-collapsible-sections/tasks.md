## 1. Shared `CollapsibleSection` component

- [x] 1.1 Create `frontend/src/components/CollapsibleSection.tsx` accepting `id` (persistence key slug), `title`, `count`, an optional `headerAction` (rendered next to the toggle, always visible), and `children` (collapsible body)
- [x] 1.2 Implement expand/collapse state via `useState`'s lazy initializer reading `localStorage.getItem(`dashboard.section.${id}`)`, defaulting to expanded when absent or not the literal string `"false"`, wrapped in `try/catch` so a `localStorage` failure falls back to expanded rather than throwing
- [x] 1.3 Persist state on every toggle via `localStorage.setItem`, also wrapped in `try/catch`
- [x] 1.4 Render the header as a real `<button>` (not a clickable `<div>`) with `aria-expanded`, extending the existing `sectionHeading`/`sectionLabel` typographic convention (`CustomMonsterList.tsx`, `InventoryPanel.tsx`) with a count badge and a chevron that reflects expanded/collapsed state
- [x] 1.5 Render a divider between the header and the body, and render `children` only when expanded

## 2. Restructure the DM column

- [x] 2.1 Extract the edition selector (5e/5.5e) out of the Rooms grouping to a standalone control rendered above the DM column's three sections, still backed by the same `dmEdition` state and still read by `handleCreateRoom`
- [x] 2.2 Wrap "My Rooms" in a `CollapsibleSection` (`id="dm.rooms"`, count = `me.rooms.length`, `headerAction` = "+ New Room")
- [x] 2.3 Wrap "My Monsters" in a `CollapsibleSection` (`id="dm.monsters"`, count = `myMonsters.length`, `headerAction` = "+ New Monster" link)
- [x] 2.4 Wrap "My Encounters" in a `CollapsibleSection` (`id="dm.encounters"`, count = `myEncounters.length`, `headerAction` = "+ New Encounter" link)

## 3. Restructure the Player column

- [x] 3.1 Wrap "My Characters" in a `CollapsibleSection` (`id="player.characters"`, count = `me.pcs.length`, `headerAction` = "+ New Character" link)
- [x] 3.2 Wrap "My Parties" (the party cards, and the create-a-party name input + button, and the join-a-party controls) in a `CollapsibleSection` (`id="player.parties"`, count = `parties.length`, no `headerAction` — its create-a-party form requires a name input first, so it stays in the collapsible body like the other multi-field controls)
- [x] 3.3 Wrap "Recent Rooms" + "Join a new room" in a `CollapsibleSection` (`id="player.rooms"`, count = `me.recent_rooms.length`, no `headerAction`)

## 4. Verification

- [x] 4.1 Manually verify each of the six sections collapses/expands independently and the others are unaffected
- [x] 4.2 Manually verify collapse state survives a full page reload for a collapsed section, and that a section never collapsed defaults to expanded
- [x] 4.3 Manually verify "+ New Character," "+ New Room," "+ New Monster," and "+ New Encounter" remain clickable and functional while their section is collapsed
- [x] 4.4 Manually verify the create-a-party and join-a-party/join-a-room controls are hidden when their section is collapsed and reappear on expand, with no data loss to in-progress input
- [x] 4.5 Manually verify the DM edition selector stays visible and still governs "+ New Room" room creation while the Rooms section is collapsed
- [x] 4.6 Confirm no regression to existing Dashboard behavior: creating/editing/deleting characters, parties, rooms, monsters, and encounters, and joining a party/room by code, all still work exactly as before
