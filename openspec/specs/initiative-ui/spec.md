# Spec: Initiative UI — Staging Placeholder and Roll Tooltip

## Capability
`initiative-ui`

## Purpose
Updates the DM view to remove the manual initiative input from the Add Creature form, display a staging placeholder for unrolled entities, and show a breakdown tooltip for auto-rolled initiative values.

## Requirements

### Requirement: Add Creature Form — Remove Initiative Input

The Add Creature form in `DMView.tsx` SHALL remove the initiative number input entirely.

The form's `add_creature` WS message SHALL use this shape:

```typescript
{
  type: 'add_creature',
  name,
  max_hp: maxHP,
  quantity,
  source_type: monster?.source_type ?? '',
  reference_url: monster?.reference_url ?? '',
  pdf_object_key: monster?.pdf_object_key ?? '',
  ...(monster?.initiative_modifier != null && { initiative_modifier: monster.initiative_modifier }),
}
```

- `initiative_modifier` is only included when the monster search returned a monster with a non-null modifier.
- If no monster was found (manual name entry), `initiative_modifier` is omitted — the backend treats it as nil.

#### Scenario: Monster with modifier selected
- **GIVEN** the DM searched for and selected a monster that has a non-null `initiative_modifier`
- **WHEN** the DM submits the Add Creature form
- **THEN** the WS message includes `initiative_modifier` set to the monster's modifier value

#### Scenario: Manual creature name entered
- **GIVEN** the DM typed a name without selecting a monster from search results
- **WHEN** the DM submits the Add Creature form
- **THEN** the WS message does NOT include an `initiative_modifier` key

### Requirement: Staging Placeholder

In the entity list (pre-combat and during combat), when `entity.initiative === null`, the initiative column SHALL display `--` in place of a numeric value.

No other change to the row layout is required.

#### Scenario: Initiative not yet set
- **GIVEN** an entity has `initiative === null`
- **WHEN** the entity list is rendered
- **THEN** the initiative cell shows `--`

#### Scenario: Initiative set
- **GIVEN** an entity has a numeric `initiative` value
- **WHEN** the entity list is rendered
- **THEN** the initiative cell shows the numeric value (existing behaviour)

### Requirement: Auto-Roll Breakdown Tooltip

When `entity.initiative_roll != null`, a tooltip SHALL appear on hover over the initiative value in the DM view showing the roll breakdown.

Tooltip format:
```
16 (d20: 13 + mod: +3)
```

Format string: `{initiative} (d20: {initiative_roll} + mod: {modifier_string})`

Where `modifier_string` formats the modifier with sign: `+3`, `+0`, `-2`.

The tooltip is DM-only — players see only the final initiative number (existing behaviour unchanged).

#### Scenario: Auto-rolled initiative hovered by DM
- **GIVEN** an entity has a non-null `initiative_roll`
- **WHEN** the DM hovers over the initiative value
- **THEN** a tooltip shows the breakdown in the format above

#### Scenario: Manually set initiative
- **GIVEN** an entity has a numeric `initiative` but null `initiative_roll`
- **WHEN** the DM hovers over the initiative value
- **THEN** no breakdown tooltip is shown

#### Scenario: Player view
- **GIVEN** the current user is a player (not DM)
- **WHEN** viewing an entity with a non-null `initiative_roll`
- **THEN** no tooltip is rendered — only the final initiative number is visible

### Requirement: Frontend Entity Type Update

The `Entity` interface in `types.ts` SHALL include two new optional nullable fields:

```typescript
initiative_modifier?: number | null
initiative_roll?: number | null
```

Both fields are optional (`?`) so existing WS payloads without these fields do not require a type cast.

### Requirement: Start Combat Button Disabled Pending Initiative

The DM view SHALL disable the "Start Combat" button whenever at least one `pc` or `companion` entity in the room has `initiative === null`, matching the server's `start_combat` validation rule. A disabled button SHALL NOT send the `start_combat` message when clicked.

The disabled state SHALL be visually indicated (reduced opacity and a `not-allowed` cursor), consistent with other disabled buttons in the app.

#### Scenario: Button disabled while a PC has no initiative
- **GIVEN** at least one `pc` or `companion` entity has `initiative === null`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is shown disabled and clicking it does not send `start_combat`

#### Scenario: Button enabled once all initiatives are set
- **GIVEN** every `pc` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is enabled and clicking it sends `start_combat`

#### Scenario: Creature entities do not affect the gate
- **GIVEN** one or more `creature` entities have `initiative === null` but every `pc` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is enabled

### Requirement: Pending Initiative Summary

When the Start Combat button is disabled due to missing initiative, the DM view SHALL display a summary line listing the names of the blocking `pc` and `companion` entities, comma-separated. The line SHALL NOT render when no entity is blocking.

#### Scenario: Summary lists blocking entities by name
- **GIVEN** entities named "Bob" (pc, `initiative === null`) and "Fido" (companion, `initiative === null`) are blocking
- **WHEN** the DM view renders the Combat controls row
- **THEN** a summary line is shown naming both "Bob" and "Fido"

#### Scenario: Summary hidden once unblocked
- **GIVEN** every `pc` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** no pending-initiative summary line is rendered

#### Scenario: Sharing companion resolved alongside its owner
- **GIVEN** a companion has `shares_initiative === true` and its owning PC has just had `initiative` set (which the server auto-copies to the companion)
- **WHEN** the DM view re-renders with the updated `RoomState`
- **THEN** the companion is no longer included in the pending-initiative summary
