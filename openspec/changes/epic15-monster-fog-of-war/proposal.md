## Why

Right now the Player View renders every entity in `RoomState.entities`, including creatures the DM has staged but combat hasn't started yet. A DM building an encounter mid-session (adding goblins, setting modifiers, checking the initiative queue) spoils the encounter for players who can see the full monster list update live over the WebSocket before "Start Combat" is even clicked. Epic 15 asks for a "fog of war" during staging: players should only see PCs and companions until combat is actually live, then see everything the moment it starts.

## What Changes

- **No backend change required.** The epic's AC1–4 (a combat-active flag present in every broadcast, defaulting to false, flipping true/false on start/end combat) are already fully satisfied by the existing `RoomState.IsStarted` field (`json:"is_started"`), which is already broadcast on every state mutation per the `room-state` spec. This proposal treats that as a decision record, not new work — no rename to `is_combat_active` (the Mongo-side field of that name is an unrelated dashboard-summary concept).
- Add a client-side filter in `PlayerView.tsx`: while `is_started` is `false`, the initiative ladder hides all entities of `type: "creature"`; PCs and companions always render.
- The filter is disabled entirely once `is_started` becomes `true` — no transition/animation, consistent with how every other state change in the app renders (instant re-render).
- Add a staging-specific empty-state message (distinct from the existing generic "No combatants yet.") shown when the *filtered* list is empty but the DM has actually staged hidden creatures, so players know prep is underway rather than assuming nothing has happened.
- The DM Panel (`DMView.tsx`) is unaffected — it already renders the full unfiltered entity list, and is a fully separate component with no shared list-rendering code, so no explicit exclusion logic is needed.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `room-state`: extends the existing "Frontend is responsible for role-based data presentation" requirement with a new scenario covering pre-combat creature visibility for player-role clients.

## Impact

- `frontend/src/components/PlayerView.tsx`: add the staging filter and empty-state copy over the existing initiative ladder render.
- `openspec/specs/room-state/spec.md`: new scenario under the existing role-based presentation requirement (no change to the data model, since `is_started` already exists).
- No changes to `room/room.go`, `ws/handler.go`, `store/room.go`, or `DMView.tsx`.
