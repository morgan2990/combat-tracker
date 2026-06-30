## 1. Backend: EndCombat Method

- [x] 1.1 Add `EndCombat(sessionID string) error` to `Room` in `room/room.go`: check DM role; return error if `!is_started`; two-pass entity filter (pass 1: collect player IDs; pass 2: keep players and companions whose `owner_id` is in the player ID set, discard everything else); set `IsStarted = false`, `Round = 0`, `ActiveIndex = 0`; broadcast

## 2. Backend: WS Dispatcher

- [x] 2.1 Add `end_combat` case to `dispatch()` in `ws/handler.go`: no payload to unmarshal; call `rm.EndCombat(c.SessionID)`; broadcast on success

## 3. Frontend: DMView End Combat Button

- [x] 3.1 Add `confirmingEnd` boolean state to `DMView`; render "End Combat" button (visible only when `is_started`) that sets `confirmingEnd = true` on click
- [x] 3.2 When `confirmingEnd` is true, replace the "End Combat" button with an inline confirmation row: warning text + "Cancel" button (resets `confirmingEnd = false`) + "Yes, End Combat" button (sends `{ type: "end_combat" }` and resets `confirmingEnd = false`)
- [x] 3.3 Reset `confirmingEnd` to `false` whenever `is_started` transitions from `true` to `false` in incoming `RoomState` (prevents stale confirmation UI if another client triggers end combat)

## 4. Integration Verification

- [x] 4.1 Build Go backend (`go build ./...`); confirm no compile errors
- [x] 4.2 Start server and dev frontend; run a full combat session (join DM + players, start combat, add a creature, advance turns); click "End Combat", verify confirmation row appears
- [x] 4.3 Click "Cancel"; verify button is restored and no state change occurs on any client
- [x] 4.4 Click "End Combat" again and confirm; verify creatures are removed, players and companions remain, round resets to 0, all clients show "Waiting" state
- [x] 4.5 Verify player HP, conditions, and dead flags are unchanged after End Combat
