## 1. US5.1: JoinScreen — DM Room Creation UI

- [x] 1.1 Add `creating` boolean state and `createError` string state to `JoinScreen`; extract the name input so it is shared between both DM sub-flows
- [x] 1.2 Restructure the DM role section into two visually separated sub-sections: "Create New Room" (name + button) and "Rejoin Existing Room" (name + room code + DM token + button)
- [x] 1.3 Implement the "Create New Room" button onClick: disable the button, call `fetch('/api/rooms', { method: 'POST' })`, parse `{ room_id, dm_token }` from the JSON response, then call `onJoin(room_id, name, 'dm', dm_token)`
- [x] 1.4 On fetch failure (non-2xx or network error), set `createError` and re-enable the button; render the error message inline below the Create button

## 2. US5.2: DMView — Kill Zeroes HP

- [x] 2.1 In `EntityRow` (DMView), update the Kill button onClick to send `sendUpdate({ dead: true, current_hp: 0 })` when toggling to dead; Revive remains `sendUpdate({ dead: false })` with no HP change

## 3. US5.3: Unconscious / Dead Visual States

- [x] 3.1 Add `entityVitalState` helper inline in `DMView`: returns `'dead'` if `entity.dead`, `'unconscious'` if `entity.current_hp === 0`, otherwise `'alive'`
- [x] 3.2 Update `EntityRow` (DMView): apply amber background (`#fff8e1`) and `😵 Unconscious` badge for the unconscious state alongside the existing grey/💀 Dead treatment
- [x] 3.3 Add the same `entityVitalState` helper inline in `PlayerView`
- [x] 3.4 Update the initiative tracker rows in `PlayerView`: apply Dead (grey, `#aaa` text, `💀 Dead` badge) and Unconscious (amber tint, dim text, `😵 Unconscious` badge) treatments to player and companion entities; creature rows are unaffected (fog-of-war)
- [x] 3.5 Update the "My entity" editor panel header in `PlayerView`: show the Dead or Unconscious badge next to the entity name when applicable
- [x] 3.6 Update the companion editor panel headers in `PlayerView`: apply the same badge logic as the player entity panel

## 4. Integration Verification

- [x] 4.1 Open the join screen as DM, enter a name, click "Create New Room"; verify the DM panel opens without requiring any manual copy-paste
- [x] 4.2 Reload the page; use the "Rejoin Existing Room" form with the room code and DM token; verify successful reconnection
- [x] 4.3 Simulate a network error on room creation (disconnect or use browser DevTools); verify the error message appears and the button re-enables
- [x] 4.4 Kill a creature in the DM panel; verify its HP drops to 0 and the row is greyed out with 💀 Dead on all clients
- [x] 4.5 Revive the creature; verify HP stays at 0 and the row shows 😵 Unconscious (amber) on all clients
- [x] 4.6 Heal the revived creature to a non-zero HP; verify it returns to normal alive styling on all clients
- [x] 4.7 As a player, use delta buttons to bring your own HP to 0; verify 😵 Unconscious appears in the tracker row and in your editor panel header
- [x] 4.8 Verify that a player with `dead: true` shows 💀 Dead (not unconscious) even when HP is also 0
