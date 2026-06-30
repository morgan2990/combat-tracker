# Proposal: Epic 10 — Automated Monster Initiative Rolling

## Why

Epics 8 and 9 together gave us everything needed to auto-roll initiative:

- Epic 8 scraped `initiative_modifier` from 5e.tools JSON and stored it on every monster in MongoDB.
- Epic 9 added edition context to rooms and wired monster search to that edition, so the frontend knows which monster it's adding.

The Add Creature form currently has a manual initiative input. The DM types a number, which is error-prone and slow. With modifiers already in the database, the backend can roll d20 + modifier automatically — eliminating the manual step for any saved monster.

There is also a pre-existing bug: `MonsterForm.tsx` (the "Register Monster" admin form) does not send an `edition` field, so the `UpsertMonster` handler returns 400 on every submit. This is a US10.1 fix, but it's small enough to bundle here.

## What Changes

| Area | Change |
|---|---|
| `store.Monster.InitiativeModifier` | `int` → `*int`; nil means "not set" |
| `room.Entity` | Gains `InitiativeModifier *int` and `InitiativeRoll *int` |
| `add_creature` WS message | `initiative int` replaced by `initiative_modifier *int` |
| `room.StartCombat` | Auto-rolls d20+modifier for staged creatures before setting IsStarted |
| `room.AddCreature` | Rolls immediately when `IsStarted && modifier != nil` |
| `MonsterForm.tsx` | Gains `edition` select and optional `initiative_modifier` input |
| `DMView.tsx` — Add Creature form | Initiative input removed; modifier from search result passed in WS msg |
| `types.ts` | `Entity` gains `initiative_modifier` and `initiative_roll` (both `number | null`) |
| `DMView.tsx` — initiative display | `--` placeholder for nil; tooltip for rolled breakdown |

## What Stays the Same

- Players set their own initiative via `set_initiative` (unchanged).
- DM can still manually override any entity's initiative via `dm_update_entity` at any time.
- Sorting logic (`sortEntities`) is unchanged — nil sorts last already.
- Custom monsters without a modifier (nil) get nil initiative; DM sets manually.
- The `StartCombat` guard that requires players/companions to have initiative set is unchanged.

## Out of Scope

- Re-rolling initiative (reset and re-roll all creatures mid-combat).
- Player-facing initiative roll UI.
- Displaying initiative rolls to players.
