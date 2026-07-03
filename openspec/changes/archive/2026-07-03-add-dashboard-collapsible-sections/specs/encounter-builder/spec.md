## MODIFIED Requirements

### Requirement: Dashboard Encounters List

The Dashboard's "As DM" card SHALL render a "My Encounters" list, fetched from `GET /api/encounters` on mount, following the same row layout as the existing "My Monsters" list: each row shows the encounter's `name`, an "Edit" link to `/encounters/:id/edit`, and a "Delete" button. A "+ New Encounter" link to `/encounters/new` SHALL appear in the Encounters section's header, alongside the section's collapse/expand toggle.

#### Scenario: DM sees their saved encounters
- **WHEN** the Dashboard loads for a DM with 2 saved encounters
- **THEN** both encounters render as rows with Edit and Delete controls

#### Scenario: DM with no encounters sees an empty-state message
- **WHEN** the Dashboard loads for a DM with zero saved encounters
- **THEN** a message equivalent to "No custom monsters yet." (e.g. "No encounters yet.") renders in place of the list

#### Scenario: DM deletes an encounter from the Dashboard
- **WHEN** the DM clicks Delete on an encounter row and confirms
- **THEN** the frontend SHALL send `DELETE /api/encounters/:id` and remove the row from the list on success

#### Scenario: New Encounter link is reachable from the section header
- **WHEN** the DM views the Encounters section, expanded or collapsed
- **THEN** the "+ New Encounter" link SHALL be visible in the section's header and SHALL navigate to `/encounters/new` when activated
