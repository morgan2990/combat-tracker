## ADDED Requirements

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
- **THEN** the frontend SHALL send `DELETE /api/monsters/custom/:id` and remove the row from the list on success

#### Scenario: DM cancels the delete confirmation
- **WHEN** a DM clicks Delete on a row and then declines/cancels the confirmation step
- **THEN** the frontend SHALL NOT send a delete request and the row SHALL remain in the list
