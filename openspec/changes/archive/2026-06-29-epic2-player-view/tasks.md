## 1. Backend: Room Methods

- [x] 1.1 Add `sortEntities()` private method to `Room`: stable descending sort of `State.Entities` by `initiative`, skipped when `is_started` is true
- [x] 1.2 Add `SetupCharacter(sessionID string, maxHP, initiative int) error` to `Room`: create player entity, link `session_id`, call `sortEntities()`, return error if `maxHP <= 0` or session has no associated name
- [x] 1.3 Add `UpdateEntity(sessionID string, entityID string, currentHP, tempHP int, conditions []string) error` to `Room`: find entity, validate ownership (own session or owned companion), clamp `current_hp` to `max_hp`, apply update
- [x] 1.4 Add `AddCompanion(sessionID string, name string, maxHP, initiative int) error` to `Room`: find player's entity by `session_id`, create companion with `owner_id`, call `sortEntities()`
- [x] 1.5 Update `ValidateAndRegister` to handle reconnection: after name-free check passes, scan `State.Entities` for an entity matching name + type "player"; if found, update its `session_id` to the new session before registering the client

## 2. Backend: WS Message Dispatcher

- [x] 2.1 Define an `ActionMessage` struct in `ws/` with a `Type string` field and raw JSON `Payload json.RawMessage` for routing
- [x] 2.2 Replace the discard read loop in `serve()` with a dispatcher: unmarshal each message into `ActionMessage`, switch on `Type`, call the appropriate room method
- [x] 2.3 Implement `handleSetupCharacter`: unmarshal payload, call `rm.SetupCharacter()`, broadcast on success
- [x] 2.4 Implement `handleUpdateEntity`: unmarshal payload, call `rm.UpdateEntity()`, broadcast on success
- [x] 2.5 Implement `handleAddCompanion`: unmarshal payload, call `rm.AddCompanion()`, broadcast on success

## 3. Frontend: Post-Connect Setup Detection

- [x] 3.1 In `App.tsx`, after the first `onmessage` sets `roomState`, check if an entity with `name === myName && type === "player"` exists; if found set `needsSetup = false`, if not set `needsSetup = true`
- [x] 3.2 Add a `needsSetup` state flag to `App`; render `SetupForm` when `status === "connected" && needsSetup`

## 4. Frontend: SetupForm Component

- [x] 4.1 Create `frontend/src/components/SetupForm.tsx` with numeric inputs for Max HP and Initiative and a submit button
- [x] 4.2 On submit, send `{ type: "setup_character", max_hp: N, initiative: M }` over the WebSocket and disable the form until the next state broadcast confirms the entity exists

## 5. Frontend: Full PlayerView Implementation

- [x] 5.1 Replace the placeholder initiative list in `PlayerView.tsx` with a full list rendering all entities in received order (server-sorted); highlight the active entity when `is_started` is true
- [x] 5.2 Apply fog of war: render qualitative HP label (Healthy / Hurt / Injured / Dying / Dead) for `type: "creature"` entities; render exact HP for `type: "player"` and `type: "companion"`
- [x] 5.3 Add the hybrid HP editor for the player's own entity: delta buttons `[-10][-5][-1]` and `[+1][+5][+10]` flanking the HP display; tapping the HP display opens an inline `<input type="number">` for direct set
- [x] 5.4 On any HP change (delta or direct set), clamp to `[0, max_hp]` client-side and send `update_entity`
- [x] 5.5 Add condition toggles below the HP editor: render the 8 predefined conditions (Prone, Stunned, Poisoned, Blinded, Frightened, Incapacitated, Restrained, Paralyzed) as tap-to-toggle chips; send `update_entity` with the updated conditions array on each toggle
- [x] 5.6 Add "Add Summon/Pet" button below own entity section; clicking opens the `CompanionForm`

## 6. Frontend: CompanionForm Component

- [x] 6.1 Create `frontend/src/components/CompanionForm.tsx` with inputs for Name, Max HP, and Initiative and a submit button
- [x] 6.2 On submit, send `{ type: "add_companion", name: "...", max_hp: N, initiative: M }` over the WebSocket and close the form

## 7. Frontend: Companion Editing in PlayerView

- [x] 7.1 For each companion entity whose `owner_id` matches the player's own entity ID, render an HP editor and condition toggles identical to the player's own entity section
- [x] 7.2 Send `update_entity` with the companion's `entity_id` when the owner edits a companion's HP or conditions

## 8. Integration Verification

- [ ] 8.1 Start the Go server; verify `POST /api/rooms` still works and `go build` is clean after the new room methods
- [ ] 8.2 Open a player tab: join a room, verify the setup form appears, submit Max HP + Initiative, verify the entity appears in the initiative list
- [ ] 8.3 Open a second player tab with a different name: verify both players appear in each other's tracker in the correct initiative order
- [ ] 8.4 Update HP via delta buttons and direct set on the first player; verify the change is immediately visible on the second tab
- [ ] 8.5 Toggle conditions on the first player; verify they appear on both tabs
- [ ] 8.6 Add a companion on the first player; verify it appears in the tracker sorted by initiative; verify the second player cannot edit it
- [ ] 8.7 Close the first player tab and reopen; verify reconnection re-links to the existing entity (setup form is skipped) and companion is still present
- [ ] 8.8 Attempt to send an `update_entity` for another player's entity ID (via browser DevTools WS console); verify the server ignores it and no broadcast occurs
