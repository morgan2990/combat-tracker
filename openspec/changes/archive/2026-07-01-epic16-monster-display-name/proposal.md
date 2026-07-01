## Why

DMs currently can't give a spawned monster instance a narrative identity distinct from its statblock name — a DM wanting to introduce "Chief Bugbear" or "Guard Alpha" has to either rename the base entity (losing track of which statblock/HP/modifiers it came from) or expose the real monster name to players, spoiling the encounter. Epic 16 adds an optional per-instance alias (`display_name`) that DMs see alongside the base name, while players see only the alias.

## What Changes

- Add an optional `display_name` field to the entity model (Go struct, Mongo persistence, WS payload, frontend type) — empty/nil by default, matching the pattern already used for `initiative_modifier`/`initiative_roll`.
- Add an optional "Custom Display Name / Alias" input to the DM Panel's Add Creature form.
- When quantity > 1 and an alias is provided, auto-number the alias per instance exactly like the base name already is (`"Guard 1"`, `"Guard 2"`, `"Guard 3"`) — **deviates from Epic 16's literal AC4**, which asked for one identical alias across the whole batch; decided during exploration that indistinguishable player-facing rows for a multi-monster batch is worse than the minor loss of "same name" flavor, and it mirrors existing base-name numbering exactly.
- DM Panel initiative tracker renders `"{display_name} ({name})"` when an alias is set (e.g. `"Guard 1 (Goblin 1)"`), falling back to `name` alone when it isn't.
- Player View initiative ladder renders `display_name` only when set, completely omitting `name`; falls back to `name` when no alias is set. This extends the existing role-based-presentation pattern already used for HP masking and pre-combat creature masking.
- **Scope addition beyond Epic 16's literal ACs**: allow the DM to edit an entity's `display_name` after creation, via the same expanded entity-row control that already lets a DM live-edit a creature's base `name`. The epic only specified setting the alias at creation time (`add_creature`); decided during exploration that a write-once alias with no correction path would be a real usability gap given the app already has a live-rename pattern for the adjacent `name` field.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `entity-schema`: adds `DisplayName`/`display_name` to the `Entity` struct and the `addCreatureMsg` WS message, following the same optional-field precedent as `InitiativeModifier`.
- `room-persistence`: the enumerated persisted-entity field list (which the spec explicitly states is exhaustive) gains `display_name`; snapshot/restore round-trip it.
- `room-state`: extends "Frontend is responsible for role-based data presentation" with scenarios for DM dual-label rendering and player-side name masking.

## Impact

- `room/room.go`: `Entity` struct, `AddCreature` (alias param + batch numbering), `DMUpdateEntity` (alias edit support), `snapshot`/`inflateRoom`.
- `store/room.go`: `RoomEntitySnapshot` struct.
- `ws/handler.go`: `addCreatureMsg` and `dmUpdateEntityMsg` structs, their dispatch calls.
- `frontend/src/types.ts`: `Entity.display_name`.
- `frontend/src/components/DMView.tsx`: Add Creature form new field, dual-label row render, alias live-edit control in the expanded row.
- `frontend/src/components/PlayerView.tsx`: row render falls back from `display_name` to `name`.
- No changes to companion or PC entities — this is creature-only, since only `add_creature` gets the new field.
