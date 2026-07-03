## MODIFIED Requirements

### Requirement: A section's primary create-new action stays reachable when collapsed
For the Characters, Monsters, Encounters, and DM Rooms sections — whose primary "create new" affordance is a single-interaction action requiring no accompanying input — that action SHALL render as a filled button and SHALL remain visible and usable in the section's header regardless of whether the section is expanded or collapsed. The Parties section's "create a party" control (requires a party name) and the Rooms sections' "join" controls (require a PC selection plus a code) are not single-interaction actions; they SHALL render as part of each section's collapsible content and SHALL be hidden when that section is collapsed.

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

#### Scenario: Create-new actions render as filled buttons
- **WHEN** the Dashboard renders the Characters, Monsters, Encounters, or DM Rooms section header
- **THEN** that section's "+ New …" action SHALL render as a filled button, matching the visual style of the other three sections' create actions

#### Scenario: Create-a-party control hidden while its section is collapsed
- **WHEN** the Parties section is collapsed
- **THEN** the "create a party" name field and button SHALL NOT be visible

#### Scenario: Join-a-room control hidden while its section is collapsed
- **WHEN** the player-side Rooms section is collapsed
- **THEN** the "join a new room" controls SHALL NOT be visible
