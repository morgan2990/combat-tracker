## MODIFIED Requirements

### Requirement: Encounter Builder Screen

The system SHALL provide an encounter builder screen reachable at `/encounters/new` (create mode) and `/encounters/:id/edit` (edit mode, mirroring `MonsterForm.tsx`'s create/edit split). The screen SHALL include:
- A text field for the encounter's `name`.
- An edition selector (`"5e"` / `"5.5e"`).
- A monster search input backed by `GET /api/search/monsters`, filtered by the selected edition.
- A "My Creatures" quick-pick section listing the DM's own custom monsters for the selected edition (fetched via `GET /api/custom-monsters?edition=<selected edition>`), alongside the search input.
- A staging list of monster groups the DM has added, each with an editable `quantity` and an optional "Custom Display Name / Alias" field.

The "My Creatures" quick-pick section SHALL render tier-aware based on viewport width, using the same phone/tablet/desktop tier detection and phone breakpoint (below 768px) as the `dmview-responsive-layout` capability:
- At the phone tier (viewport width below 768px), the section SHALL render each custom monster as a pill-style button, as it does today.
- At the tablet and desktop tiers (viewport width 768px and above), the section SHALL render each custom monster as a row-list item, matching the row style DMView's DM Nav column uses for the same list.

Selecting a monster from search results SHALL add it to the staging list with the same `is_custom` discriminator the search hit already carries. Selecting a monster from the "My Creatures" quick-pick section (pill or row, depending on tier) SHALL add it to the staging list directly from the already-fetched document (`is_custom: true`, `monster_id` set), without an additional network request. Submitting the form SHALL send `POST /api/encounters` (create mode) or `PUT /api/encounters/:id` (edit mode) with the encounter's `name`, `edition`, and the staged `monsters` array, then redirect to the Dashboard.

#### Scenario: DM adds a monster group to a new encounter
- **WHEN** the DM searches for and selects "Goblin" (an official monster), sets quantity to 3, and enters "Ambush Party" as the alias
- **THEN** the staging list shows one group `{ name: "Goblin", is_custom: false, quantity: 3, display_name: "Ambush Party" }`

#### Scenario: DM adds a custom monster group via search
- **WHEN** the DM selects a custom monster from search results
- **THEN** the staged group's `is_custom` is `true` and `monster_id` is set to that custom monster's id

#### Scenario: DM adds a custom monster group via the quick-pick section
- **WHEN** the DM clicks one of their own custom monsters in the "My Creatures" section, at any viewport tier
- **THEN** a staged group is appended with `is_custom: true`, `monster_id` set to that monster's id, and `name` set to its current name, with no additional fetch

#### Scenario: Quick-pick section only shows the selected edition
- **WHEN** the DM has custom monsters in both `"5e"` and `"5.5e"` and the builder's edition selector is set to `"5.5e"`
- **THEN** the "My Creatures" section lists only the `"5.5e"` monsters

#### Scenario: Phone tier shows pill buttons
- **WHEN** the Encounter Builder renders at a viewport width below 768px
- **THEN** the "My Creatures" section SHALL render each custom monster as a pill-style button

#### Scenario: Tablet and desktop tiers show row-list items
- **WHEN** the Encounter Builder renders at a viewport width of 768px or above
- **THEN** the "My Creatures" section SHALL render each custom monster as a row-list item instead of a pill-style button

#### Scenario: Submitting a new encounter
- **WHEN** the DM fills in a name, edition, and at least one monster group, then submits
- **THEN** the frontend SHALL send `POST /api/encounters` with the full payload and redirect to the Dashboard on success

#### Scenario: Editing an existing encounter loads its current state
- **WHEN** the DM opens `/encounters/:id/edit` for a previously saved encounter
- **THEN** the form SHALL populate the name, edition selector, and staging list from `GET /api/encounters/:id`

#### Scenario: Submitting in edit mode updates the existing document
- **WHEN** the DM changes the staging list on the edit screen and submits
- **THEN** the frontend SHALL send `PUT /api/encounters/:id` with the updated values, not create a new document
