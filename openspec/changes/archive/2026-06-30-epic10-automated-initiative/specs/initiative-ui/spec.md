# Spec: Initiative UI — Staging Placeholder and Roll Tooltip (delta)

## Capability
`initiative-ui`

## Type
Delta — extends DMView.tsx and types.ts

## Add Creature form changes

Remove the initiative number input entirely. The form sends:

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

- `initiative_modifier` is only included in the message when the search found a monster with a non-null modifier.
- If no monster was found (manual name entry), `initiative_modifier` is omitted — backend treats it as nil.

## Staging placeholder

In the entity list (pre-combat), when `entity.initiative === null`, display `--` in place of the initiative value. No other change to the row layout.

## Breakdown tooltip

When `entity.initiative_roll != null`, render a tooltip on hover over the initiative value:

```
16 (d20: 13 + mod: +3)
```

Format: `{initiative} (d20: {initiative_roll} + mod: {modifier_string})`
Where `modifier_string` formats the modifier with sign: `+3`, `+0`, `-2`.

The tooltip is DM-only — players see only the final initiative number (existing behaviour).

## Frontend Entity type

In `types.ts`, add to the `Entity` interface:

```typescript
initiative_modifier?: number | null
initiative_roll?: number | null
```

Optional (`?`) so existing WS payloads without these fields don't require a type cast.
