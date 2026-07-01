## ADDED Requirements

### Requirement: Dashboard "As DM" Panel Provides the Monster Creation Entry Point

The main Dashboard SHALL include a `+ New Monster` link inside the "As DM" panel, styled and positioned consistently with the `+ New Character` link in the "As Player" panel. Activating it SHALL navigate to `/monsters/new`.

#### Scenario: DM opens monster creation from the dashboard
- **WHEN** an authenticated DM clicks `+ New Monster` in the "As DM" panel on the main dashboard
- **THEN** the frontend SHALL navigate to `/monsters/new` and render `MonsterForm`

#### Scenario: Entry point is visible without an active room
- **WHEN** a DM has zero rooms and views the main dashboard
- **THEN** the `+ New Monster` link SHALL still be visible and usable in the "As DM" panel

### Requirement: In-Room Monster Creation Entry Point Is Removed

The combat room DM panel (`DMView.tsx`) SHALL NOT provide a button or other control that opens monster creation. Monster creation SHALL only be reachable from the main dashboard.

#### Scenario: DM panel no longer offers monster creation
- **WHEN** a DM opens an active combat room
- **THEN** the room header SHALL NOT contain a `+ Monster` (or equivalent) control that navigates to `/monsters/new`

### Requirement: Post-Save Screen Supports Batch Creation and Returns to Dashboard

After a monster is successfully saved, `MonsterForm` SHALL present two actions: one that resets the form to create another monster without leaving the page, and one that returns the user to the main dashboard.

#### Scenario: DM creates multiple monsters in one sitting
- **WHEN** a DM saves a monster and then clicks "Add Another"
- **THEN** the form SHALL reset to its empty state and remain on `/monsters/new`, without navigating away

#### Scenario: DM finishes monster prep and returns to the dashboard
- **WHEN** a DM saves a monster and then clicks "Back to Dashboard"
- **THEN** the frontend SHALL navigate to `/`

#### Scenario: No confirmation toast is shown
- **WHEN** a monster is successfully saved
- **THEN** the frontend SHALL rely on the save-confirmation screen's own buttons for feedback and SHALL NOT display a separate toast or timed notification
