# Spec: Dashboard Monster Creation

## Purpose

Defines how DMs create monsters from the main Dashboard rather than from inside an active combat room: the "As DM" panel provides the entry point, the in-room control is removed, and the post-save screen supports creating multiple monsters in one sitting before returning to the dashboard.

## Requirements

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

### Requirement: Dashboard "As DM" Panel Lists the DM's Own Custom Monsters

The Dashboard's "As DM" panel SHALL include a "My Monsters" list showing only the requesting DM's own custom monsters (`owner_id` matching the authenticated user), regardless of their `private` value. The list SHALL be populated from the owner-scoped listing endpoint (see `monster-repository`).

#### Scenario: DM sees only their own monsters
- **WHEN** a DM who has created two custom monsters (one public, one private) views their dashboard, and another DM has also created custom monsters
- **THEN** the "My Monsters" list SHALL show only the requesting DM's two monsters, not the other DM's

#### Scenario: DM with no custom monsters sees an empty state
- **WHEN** a DM with zero custom monsters views the dashboard
- **THEN** the "My Monsters" list SHALL render an empty/no-monsters state rather than an error

### Requirement: DM Can Edit a Custom Monster from the Dashboard List

Each row in the "My Monsters" list SHALL include an Edit action that navigates to `/monsters/custom/:id/edit` for that monster.

#### Scenario: DM opens edit from the dashboard list
- **WHEN** a DM clicks Edit on a row in "My Monsters"
- **THEN** the frontend SHALL navigate to `/monsters/custom/:id/edit` for that monster's id

### Requirement: DM Can Delete a Custom Monster from the Dashboard List, With Confirmation

Each row in the "My Monsters" list SHALL include a Delete action. Activating it SHALL require an explicit confirm step before the delete request is sent — this is the first delete-of-persisted-data action in the application, and MUST NOT execute on a single, unconfirmed click.

#### Scenario: DM deletes a monster after confirming
- **WHEN** a DM clicks Delete on a row and confirms the action
- **THEN** the frontend SHALL send `DELETE /api/custom-monsters/:id` and remove the row from the list on success

#### Scenario: DM cancels the delete confirmation
- **WHEN** a DM clicks Delete on a row and then declines/cancels the confirmation step
- **THEN** the frontend SHALL NOT send a delete request and the row SHALL remain in the list
