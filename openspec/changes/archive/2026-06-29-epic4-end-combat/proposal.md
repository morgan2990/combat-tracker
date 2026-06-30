## Why

Once a combat encounter ends the DM has no way to clean up the tracker — creatures linger, the round counter stays frozen, and players remain locked in combat view. Epic 4 adds an "End Combat" action that resets the room to a between-encounter state in one click.

## What Changes

- Add an "End Combat" button to the DM panel, visible only when `is_started` is true
- Clicking shows an inline confirmation row before sending the action (prevents accidental clicks during live play)
- Upon confirmation, the backend removes all creature-type entities, retains all players and companions whose owner entity still exists, resets `is_started`, `round`, and `active_index`, and broadcasts the cleaned state to all clients
- Player and companion HP, conditions, and dead flags are deliberately preserved — healing happens explicitly, not automatically

## Capabilities

### New Capabilities

- `end-combat`: DM-only action to terminate an active combat encounter, filter entities to players and companions, and reset combat state

### Modified Capabilities

_(none — existing specs for room-state and combat-turn-flow already describe the pre/post-combat state; no requirement text changes)_

## Impact

- `room/room.go`: new `EndCombat(sessionID string) error` method on `Room`
- `ws/handler.go`: new `end_combat` case in `dispatch()`
- `frontend/src/components/DMView.tsx`: "End Combat" button with inline confirmation state
