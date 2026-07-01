# Spec: Monster Form — Edition and Initiative Modifier Fields

## Capability
`monster-form`

## Purpose
Extends `MonsterForm.tsx` with a required edition selector and an optional initiative modifier input, fixing a broken submission path and enabling modifier-aware initiative rolling when monsters are added to combat.

## Requirements

### Requirement: Edition Selector

`MonsterForm.tsx` SHALL include a required `edition` select field.

- Options: `"5e"` and `"5.5e"`
- Default value: `"5e"`
- The field MUST be included in every JSON submission to the `UpsertMonster` handler.

#### Scenario: Form submitted without changing edition
- **WHEN** the user submits the form without touching the edition selector
- **THEN** the payload includes `"edition": "5e"`

#### Scenario: User selects 5.5e
- **WHEN** the user selects `"5.5e"` and submits
- **THEN** the payload includes `"edition": "5.5e"`

### Requirement: Initiative Modifier Input

`MonsterForm.tsx` SHALL include an optional numeric `initiative_modifier` input (integer; may be negative).

- Label: "Initiative Modifier (optional)"
- If left blank, the field MUST be omitted from the JSON payload entirely (not sent as 0 or null).
- If filled with a valid integer (including 0 or negative), it MUST be sent as a number.

#### Scenario: Modifier provided
- **WHEN** the user enters `2` in the initiative modifier field and submits
- **THEN** the payload contains `"initiative_modifier": 2`

#### Scenario: Modifier left blank
- **WHEN** the initiative modifier field is empty and the user submits
- **THEN** the payload does NOT contain an `initiative_modifier` key

#### Scenario: Zero modifier
- **WHEN** the user enters `0` in the initiative modifier field and submits
- **THEN** the payload contains `"initiative_modifier": 0`

#### Scenario: Negative modifier
- **WHEN** the user enters `-2` and submits
- **THEN** the payload contains `"initiative_modifier": -2`

### Requirement: JSON Payload Shape

When `initiative_modifier` is provided:
```json
{ "name": "...", "edition": "5e", "max_hp": 30, "initiative_modifier": 2 }
```

When `initiative_modifier` is blank:
```json
{ "name": "...", "edition": "5e", "max_hp": 30 }
```

### Requirement: Multipart PDF Upload Path

The multipart branch in the `UpsertMonster` handler MUST also parse `initiative_modifier` as optional:

```go
if v := r.FormValue("initiative_modifier"); v != "" {
    val, _ := strconv.Atoi(v)
    m.InitiativeModifier = &val
}
```

With `Monster.InitiativeModifier` typed as `*int`, a missing field decodes as nil on the backend.

#### Scenario: PDF upload with modifier
- **WHEN** a multipart form submission includes an `initiative_modifier` value
- **THEN** the stored monster has a non-nil `InitiativeModifier`

#### Scenario: PDF upload without modifier
- **WHEN** a multipart form submission omits `initiative_modifier`
- **THEN** the stored monster has a nil `InitiativeModifier`

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
