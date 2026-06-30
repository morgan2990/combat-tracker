## 1. Backend: Entity Model

- [x] 1.1 Add `Dead bool` field to `Entity` struct in `room/room.go` with JSON tag `"dead"`; initialize to `false` in `SetupCharacter`, `AddCompanion`, and any new `AddCreature` method

## 2. Backend: Sort Rewrite

- [x] 2.1 Rewrite `sortEntities()` in `room/room.go`: remove the `is_started` guard; when `is_started` is true, record the active entity ID before sorting and scan the sorted slice afterward to restore `active_index` to that entity's new position

## 3. Backend: Combat Turn Flow Methods

- [x] 3.1 Add `StartCombat(sessionID string) error` to `Room`: check `c.Role == "dm"`, return error if already started or no session; set `IsStarted = true`, `Round = 1`, `ActiveIndex = 0`; broadcast
- [x] 3.2 Add `NextTurn(sessionID string) error` to `Room`: check DM role and `IsStarted`; if `ActiveIndex` is at the last entity set `ActiveIndex = 0` and increment `Round`, otherwise increment `ActiveIndex`; broadcast

## 4. Backend: Creature Management Methods

- [x] 4.1 Add `AddCreature(sessionID string, name string, maxHP, initiative int) error` to `Room`: check DM role, validate inputs; create entity with `Type: "creature"`, `Dead: false`; call `sortEntities()`; broadcast
- [x] 4.2 Add `RemoveEntity(sessionID string, entityID string) error` to `Room`: check DM role; find and remove entity; adjust `ActiveIndex` per design rule (below / at / above); broadcast
- [x] 4.3 Add `RemoveDeadCreatures(sessionID string) error` to `Room`: check DM role; remove all entities where `Dead == true && Type == "creature"`; adjust `ActiveIndex` for each removal in order; broadcast only if any entities were removed
- [x] 4.4 Add `ToggleDead` support: the `DMUpdateEntity` method (task 5.1) handles dead toggle — no separate method needed

## 5. Backend: DM Override Method

- [x] 5.1 Add `DMUpdateEntity(sessionID string, entityID string, name string, currentHP, tempHP, initiative int, conditions []string, dead bool) error` to `Room`: check DM role; find entity; apply `name` only when `entity.Type == "creature"`; clamp `currentHP` to `[0, max_hp]`; if `initiative` changed call `sortEntities()`; apply all fields; broadcast

## 6. Backend: WS Dispatcher — New Message Types

- [x] 6.1 Add `start_combat` case to `dispatch()` in `ws/handler.go`: unmarshal (no payload), call `rm.StartCombat(c.SessionID)`, broadcast on success
- [x] 6.2 Add `next_turn` case: unmarshal (no payload), call `rm.NextTurn(c.SessionID)`, broadcast on success
- [x] 6.3 Add `add_creature` case: unmarshal `name`, `max_hp`, `initiative`; call `rm.AddCreature()`; broadcast on success
- [x] 6.4 Add `remove_entity` case: unmarshal `entity_id`; call `rm.RemoveEntity()`; broadcast on success
- [x] 6.5 Add `remove_dead_creatures` case: unmarshal (no payload); call `rm.RemoveDeadCreatures()`; broadcast on success (method only broadcasts if something changed)
- [x] 6.6 Add `dm_update_entity` case: unmarshal `entity_id`, `name`, `current_hp`, `temp_hp`, `initiative`, `conditions`, `dead`; call `rm.DMUpdateEntity()`; broadcast on success

## 7. Frontend: Types Update

- [x] 7.1 Add `dead: boolean` field to the `Entity` interface in `frontend/src/types.ts`

## 8. Frontend: Full DMView Implementation

- [x] 8.1 Add combat controls bar: "Start Combat" button (hidden after `is_started`); "Next Turn" button (visible only when `is_started`); round counter display
- [x] 8.2 Add "Add Creature" form: Name, Max HP, Initiative inputs; submit sends `{ type: "add_creature", name, max_hp, initiative }`; clear form on submit
- [x] 8.3 Render full initiative tracker: all entities in received order; active entity highlighted (▶ indicator); dead entities rendered greyed-out
- [x] 8.4 Add per-entity DM edit panel: smart HP input (parse `+N`/`-N` as delta, bare integer as absolute; clamp client-side; send `dm_update_entity`); condition toggles (same 8 chips as PlayerView); initiative number input (send `dm_update_entity`)
- [x] 8.5 Add creature-only name field in the DM edit panel (hidden for player and companion entities)
- [x] 8.6 Add dead toggle button per entity (label changes between "Kill" and "Revive" based on `entity.dead`); sends `dm_update_entity` with toggled `dead` value
- [x] 8.7 Add "Remove" button per entity; sends `{ type: "remove_entity", entity_id }` on confirmation or direct click
- [x] 8.8 Add "Remove All Dead Creatures" button (visible only when at least one dead creature exists); sends `{ type: "remove_dead_creatures" }`

## 9. Integration Verification

- [ ] 9.1 Build Go backend (`go build ./...`); confirm no compile errors after Entity and Room changes
- [ ] 9.2 Start server and dev frontend; create a room, join as DM and two players; verify setup forms appear for players and DM sees both in tracker
- [ ] 9.3 Click "Start Combat" as DM; verify `is_started` locks the button, round counter shows "Round 1", and players see the active indicator
- [ ] 9.4 Click "Next Turn" repeatedly; verify active indicator advances through the list and wraps back to the top incrementing to "Round 2"
- [ ] 9.5 Add a creature mid-combat; verify it appears sorted by initiative and the active turn does not shift to a different entity
- [ ] 9.6 Use smart HP input: type `+5`, `-12`, and `20` on a creature; verify delta and absolute modes apply correctly and HP is clamped
- [ ] 9.7 Click "Kill" on a creature; verify it goes greyed-out on both DM and player screens; click "Revive" and verify it returns to normal
- [ ] 9.8 Change a creature's initiative mid-combat; verify list re-sorts and active turn remains on the correct entity
- [ ] 9.9 Click "Remove All Dead Creatures"; verify only dead creature-type entities are removed; player and companion entities remain
- [ ] 9.10 Override a player's HP and conditions as DM; verify the change appears instantly on the player's screen
