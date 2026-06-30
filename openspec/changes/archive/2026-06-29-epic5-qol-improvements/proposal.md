## Why

Three rough edges remain after Epics 1–4: DMs must create rooms via cURL rather than the browser UI; the Kill button doesn't zero HP, leaving the tracker visually inconsistent; and the PlayerView has no visual distinction between a dead entity and an unconscious one (HP = 0 but not confirmed dead by the DM).

## What Changes

- **Room creation UI (US5.1):** The DM tab on the join screen gains a "Create New Room" flow — enter a name, click Create, and the browser calls `POST /api/rooms` and auto-connects. A secondary "Rejoin Existing Room" section lets DMs re-enter a room_id + dm_token to reconnect after a page reload.
- **Kill zeroes HP (US5.2):** When the DM clicks Kill on any entity, `current_hp` is set to 0 alongside `dead: true`. Reviving sets `dead: false` only; HP stays at 0 until explicitly healed.
- **Unconscious vs Dead display (US5.3):** Two distinct visual states replace the single absence of styling:
  - **Unconscious** (`current_hp === 0 && dead === false`): amber tint, 😵 badge — entity is down but not confirmed dead
  - **Dead** (`dead === true`): grey-out, 💀 badge — DM has confirmed death
  Both states apply in PlayerView (tracker rows and editor panels) and DMView (EntityRow). No new backend fields are needed; both states are derived from existing `current_hp` and `dead`.

## Capabilities

### New Capabilities

_(none — all changes refine existing capabilities)_

### Modified Capabilities

- `room-creation`: add requirement that a DM can create a room from the browser UI without a separate API call
- `creature-management`: Kill action MUST simultaneously set `dead: true` and `current_hp: 0`
- `room-state`: define Unconscious (`current_hp === 0 && dead === false`) as a distinct entity state with its own required display treatment, separate from Dead (`dead === true`)

## Impact

- `frontend/src/components/JoinScreen.tsx`: DM tab redesign — two-section layout, internal `fetch('POST /api/rooms')` call
- `frontend/src/components/DMView.tsx`: EntityRow Kill onClick sends `current_hp: 0`; EntityRow adds Unconscious visual state
- `frontend/src/components/PlayerView.tsx`: tracker rows and editor panel headers add Dead and Unconscious visual states
- No backend changes
