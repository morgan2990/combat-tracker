## ADDED Requirements

### Requirement: Dashboard groupings render as collapsible sections
The Dashboard SHALL render each of its six existing groupings — Player column: "Characters," "Parties," "Rooms"; DM column: "Rooms," "Monsters," "Encounters" — as a section with a header (title, a count of the items in that section, and a collapse/expand toggle) followed by a divider and the section's content. Each section's expanded/collapsed state SHALL be independent of the other five sections.

#### Scenario: Section header shows an item count
- **WHEN** the Dashboard renders the Characters section for a user with 2 PCs
- **THEN** the section's header SHALL display a count reflecting 2 items

#### Scenario: Collapsing one section does not affect others
- **WHEN** a user collapses the Parties section
- **THEN** the Characters, Rooms, Monsters, and Encounters sections SHALL remain in their own current expanded/collapsed states, unaffected

#### Scenario: Sections default to expanded
- **WHEN** a user with no previously stored collapse preference for any section loads the Dashboard
- **THEN** all six sections SHALL render expanded

### Requirement: Section collapse state persists across page reloads
Each section's expanded/collapsed state SHALL be persisted in the browser's `localStorage`, keyed independently per section, and SHALL be restored the next time the Dashboard renders, including after a full page reload. A section with no persisted state SHALL default to expanded.

#### Scenario: Collapsed section stays collapsed after reload
- **WHEN** a user collapses the Monsters section and then reloads the page
- **THEN** the Monsters section SHALL render collapsed

#### Scenario: Section with no stored state defaults to expanded
- **WHEN** a user has a stored collapse preference for the Parties section but none for the Encounters section
- **THEN** the Encounters section SHALL render expanded regardless of the Parties section's stored state

### Requirement: A section's primary create-new action stays reachable when collapsed
For the Characters, Monsters, Encounters, and DM Rooms sections — whose primary "create new" affordance is a single-interaction link or button requiring no accompanying input — that action SHALL remain visible and usable in the section's header regardless of whether the section is expanded or collapsed. The Parties section's "create a party" control (requires a party name) and the Rooms sections' "join" controls (require a PC selection plus a code) are not single-interaction actions; they SHALL render as part of each section's collapsible content and SHALL be hidden when that section is collapsed.

#### Scenario: New Character action usable while its section is collapsed
- **WHEN** the Characters section is collapsed
- **THEN** the "+ New Character" action SHALL still be visible and usable

#### Scenario: New Room action usable while its section is collapsed
- **WHEN** the DM Rooms section is collapsed
- **THEN** the "+ New Room" action SHALL still be visible and usable

#### Scenario: New Monster action usable while its section is collapsed
- **WHEN** the Monsters section is collapsed
- **THEN** the "+ New Monster" action SHALL still be visible and usable

#### Scenario: New Encounter action usable while its section is collapsed
- **WHEN** the Encounters section is collapsed
- **THEN** the "+ New Encounter" action SHALL still be visible and usable

#### Scenario: Create-a-party control hidden while its section is collapsed
- **WHEN** the Parties section is collapsed
- **THEN** the "create a party" name field and button SHALL NOT be visible

#### Scenario: Join-a-room control hidden while its section is collapsed
- **WHEN** the player-side Rooms section is collapsed
- **THEN** the "join a new room" controls SHALL NOT be visible

### Requirement: DM column's edition selector is independent of the Rooms section
The DM column's edition selector (`"5e"` / `"5.5e"`) SHALL render as a standalone control above the DM column's collapsible sections, not as part of the Rooms section's content, so it remains visible and usable regardless of whether the Rooms section is expanded or collapsed.

#### Scenario: Edition selector visible while Rooms section is collapsed
- **WHEN** the DM Rooms section is collapsed
- **THEN** the edition selector SHALL still be visible and usable

#### Scenario: Edition selection still governs room creation
- **WHEN** the DM selects an edition via the standalone selector and then activates "+ New Room"
- **THEN** the created room SHALL use the selected edition, the same behavior as before this change
