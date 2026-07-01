## ADDED Requirements

### Requirement: Dashboard Encounters List

The Dashboard's "As DM" card SHALL render a "My Encounters" list, fetched from `GET /api/encounters` on mount, following the same row layout as the existing "My Monsters" list: each row shows the encounter's `name`, an "Edit" link to `/encounters/:id/edit`, and a "Delete" button. A "+ New Encounter" link to `/encounters/new` SHALL appear below the list.

#### Scenario: DM sees their saved encounters
- **WHEN** the Dashboard loads for a DM with 2 saved encounters
- **THEN** both encounters render as rows with Edit and Delete controls

#### Scenario: DM with no encounters sees an empty-state message
- **WHEN** the Dashboard loads for a DM with zero saved encounters
- **THEN** a message equivalent to "No custom monsters yet." (e.g. "No encounters yet.") renders in place of the list

#### Scenario: DM deletes an encounter from the Dashboard
- **WHEN** the DM clicks Delete on an encounter row and confirms
- **THEN** the frontend SHALL send `DELETE /api/encounters/:id` and remove the row from the list on success

### Requirement: Encounter Builder Screen

The system SHALL provide an encounter builder screen reachable at `/encounters/new` (create mode) and `/encounters/:id/edit` (edit mode, mirroring `MonsterForm.tsx`'s create/edit split). The screen SHALL include:
- A text field for the encounter's `name`.
- An edition selector (`"5e"` / `"5.5e"`).
- A monster search input backed by `GET /api/search/monsters`, filtered by the selected edition.
- A staging list of monster groups the DM has added, each with an editable `quantity` and an optional "Custom Display Name / Alias" field.

Selecting a monster from search results SHALL add it to the staging list with the same `is_custom` discriminator the search hit already carries. Submitting the form SHALL send `POST /api/encounters` (create mode) or `PUT /api/encounters/:id` (edit mode) with the encounter's `name`, `edition`, and the staged `monsters` array, then redirect to the Dashboard.

#### Scenario: DM adds a monster group to a new encounter
- **WHEN** the DM searches for and selects "Goblin" (an official monster), sets quantity to 3, and enters "Ambush Party" as the alias
- **THEN** the staging list shows one group `{ name: "Goblin", is_custom: false, quantity: 3, display_name: "Ambush Party" }`

#### Scenario: DM adds a custom monster group
- **WHEN** the DM selects a custom monster from search results
- **THEN** the staged group's `is_custom` is `true` and `monster_id` is set to that custom monster's id

#### Scenario: Submitting a new encounter
- **WHEN** the DM fills in a name, edition, and at least one monster group, then submits
- **THEN** the frontend SHALL send `POST /api/encounters` with the full payload and redirect to the Dashboard on success

#### Scenario: Editing an existing encounter loads its current state
- **WHEN** the DM opens `/encounters/:id/edit` for a previously saved encounter
- **THEN** the form SHALL populate the name, edition selector, and staging list from `GET /api/encounters/:id`

#### Scenario: Submitting in edit mode updates the existing document
- **WHEN** the DM changes the staging list on the edit screen and submits
- **THEN** the frontend SHALL send `PUT /api/encounters/:id` with the updated values, not create a new document
