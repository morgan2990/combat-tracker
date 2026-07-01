## ADDED Requirements

### Requirement: Privacy Toggle

`MonsterForm.tsx` SHALL include a checkbox or toggle input labeled "Mark as Private Campaign Content", defaulting to unchecked (public), with an adjacent informational tooltip explaining that private monsters are only visible to the creating DM, in their dashboard and their active rooms.

- The field MUST be included in every JSON or multipart submission as `private` (boolean).

#### Scenario: Form submitted without touching the privacy toggle
- **WHEN** the user submits the form without checking the privacy toggle
- **THEN** the payload includes `"private": false`

#### Scenario: DM marks a monster private
- **WHEN** the user checks "Mark as Private Campaign Content" and submits
- **THEN** the payload includes `"private": true`

#### Scenario: Tooltip explains visibility scope
- **WHEN** the user hovers or focuses the informational tooltip next to the privacy toggle
- **THEN** the tooltip text SHALL explain that private monsters are only visible to the creating DM, in their dashboard and active rooms

### Requirement: Edit Mode for Custom Monsters

`MonsterForm.tsx` SHALL support an edit mode, reached via `/monsters/custom/:id/edit`, mirroring `CharacterForm.tsx`'s existing edit-mode pattern. On mount with an `id` route param, the form SHALL fetch the existing custom monster document (`GET /api/custom-monsters/:id`) and populate all fields, including the current `private` state, from the response. Submitting in edit mode SHALL send `PUT /api/custom-monsters/:id` instead of `POST /api/monsters`.

#### Scenario: Editing loads the current privacy state
- **WHEN** a DM opens `/monsters/custom/:id/edit` for a custom monster they previously marked private
- **THEN** the privacy toggle SHALL render as checked, reflecting the saved `private: true` value

#### Scenario: Editing loads a public monster's state correctly
- **WHEN** a DM opens `/monsters/custom/:id/edit` for a custom monster with `private: false`
- **THEN** the privacy toggle SHALL render as unchecked

#### Scenario: Submitting in edit mode updates the existing document
- **WHEN** a DM changes fields on the edit form and submits
- **THEN** the frontend SHALL send `PUT /api/custom-monsters/:id` with the updated values, not create a new document
