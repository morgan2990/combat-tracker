# Spec: Monster Form — Edition and Initiative Modifier Fields (delta)

## Capability
`monster-form`

## Type
Delta — extends the existing MonsterForm.tsx

## Problem
`MonsterForm.tsx` currently submits without an `edition` field. The `UpsertMonster` handler validates edition and returns 400 if missing or invalid, so manual monster registration has been broken since Epic 8.

## Changes

### Edition selector
- A required `edition` select field with options `"5e"` and `"5.5e"`, defaulting to `"5e"`.
- Included in every JSON submission.

### Initiative Modifier input
- An optional `initiative_modifier` numeric input (integer, may be negative).
- Label: "Initiative Modifier (optional)"
- If left blank, the field is omitted from the JSON payload entirely (not sent as 0 or null).
- If filled with a valid integer (including 0 or negative), it is sent as a number.

## JSON payload

When `initiative_modifier` is provided:
```json
{ "name": "...", "edition": "5e", "max_hp": 30, "initiative_modifier": 2 }
```

When `initiative_modifier` is blank:
```json
{ "name": "...", "edition": "5e", "max_hp": 30 }
```

With `Monster.InitiativeModifier` now being `*int`, a missing field decodes as nil on the backend.

## Multipart (PDF upload path)
The existing multipart branch in `UpsertMonster` handler must also parse `initiative_modifier` as optional:
```go
if v := r.FormValue("initiative_modifier"); v != "" {
    val, _ := strconv.Atoi(v)
    m.InitiativeModifier = &val
}
```
