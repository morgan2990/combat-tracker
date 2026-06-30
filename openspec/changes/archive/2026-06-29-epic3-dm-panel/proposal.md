## Why

The DM Panel placeholder from Epic 1 has no controls — the DM can observe the tracker but cannot run a combat encounter. Epic 3 delivers the full DM experience: starting combat, advancing turns, adding and killing monsters, and overriding any entity's stats on the fly.

## What Changes

- Add a "Start Combat" button that locks the initiative order and marks `is_started = true`
- Add a "Next Turn" button that advances `active_index`, wraps at the end, and increments the round counter
- Add a rapid creature-add form (name, max HP, initiative); creatures added mid-combat are sorted in by initiative
- Add per-entity controls for the DM: smart HP input (`+N`/`-N` for delta, bare number for absolute), condition toggles, initiative override, dead toggle, and a hard-remove button
- Add a "Remove All Dead Creatures" batch button that purges dead creature-type entities
- **BREAKING**: Initiative sort is no longer frozen when `is_started = true`; sorts now always run but preserve the active entity by ID rather than by array index
- Add `dead` boolean field to `Entity`; dead entities stay in the tracker, rendered greyed-out

## Capabilities

### New Capabilities

- `combat-turn-flow`: Start combat, advance turns, wrap-and-increment round, `is_started` / `active_index` / `round` lifecycle
- `creature-management`: DM adds ephemeral creature entities mid-combat; kill toggle (dead flag); per-entity remove; batch remove-all-dead-creatures
- `dm-entity-override`: DM can edit any entity's HP, conditions, initiative, and dead flag without ownership restriction; creature names are also editable by the DM; player names are not

### Modified Capabilities

- `room-state`: Entity gains a `dead bool` field; initiative sort semantics change — sorts always execute, and when `is_started` is true the sort preserves the active entity position by ID rather than freezing order entirely

## Impact

- `room/room.go`: new methods (`StartCombat`, `NextTurn`, `AddCreature`, `RemoveEntity`, `RemoveDeadCreatures`, `DMUpdateEntity`); `sortEntities()` rewritten to preserve active entity by ID; `Entity` struct gets `Dead bool` field
- `ws/handler.go`: five new DM-only message types dispatched (`start_combat`, `next_turn`, `add_creature`, `remove_entity`, `remove_dead_creatures`, `dm_update_entity`)
- `frontend/src/components/DMView.tsx`: full implementation replacing placeholder
- `frontend/src/types.ts`: `Entity` interface gains `dead` field
