## Why

The server already refuses `start_combat` when a player or companion entity has no initiative set (`room.go` `StartCombat`), but the DM client gives no indication of this — the DM can click "▶ Start Combat" and nothing happens, with no error and no broadcast. The DM has to manually scan the entity list for `--` placeholders to figure out who is blocking the start.

## What Changes

- The "▶ Start Combat" button in `DMView.tsx` is visually disabled (greyed out, `not-allowed` cursor, no click handler firing) whenever at least one `player` or `companion` entity has `initiative === null`, mirroring the server-side rule exactly.
- While disabled, a summary line is rendered next to/below the button listing the names of the blocking entities, e.g. `Waiting on initiative: Bob, Fido`. The line only renders when the blocking list is non-empty; it disappears once everyone is set and the button re-enables.
- Companions with `shares_initiative: true` are not double-counted as "stuck" once their owning player sets initiative, since the server auto-copies the player's value to them — the frontend check uses the same `entity.initiative === null` field the server already populates, so this falls out naturally with no extra logic.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `initiative-ui`: adds a new requirement for disabling the Start Combat button and displaying a pending-initiative summary line in the DM view, based on entity initiative state already present in `RoomState`.

## Impact

- Frontend only: `frontend/src/components/DMView.tsx`.
- No backend/API changes — the server's existing `StartCombat` validation in `room.go` is the source of truth this UI mirrors.
- No changes to `combat-turn-flow` spec (server behavior is unchanged).
