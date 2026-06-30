## 1. Room State & Creation (Backend)

- [x] 1.1 Add `Edition string` field to `RoomState` struct in `room/room.go`
- [x] 1.2 Update `CreateRoom` in `room/room.go` to accept an `edition` parameter and store it on `RoomState`
- [x] 1.3 Update `CreateRoom` handler in `api/handler.go` to decode an optional `edition` from the JSON request body, default to `"5e"` if absent or invalid, call `room.Global.CreateRoom(edition)`, and include `edition` in the response JSON

## 2. Monster Search (Backend)

- [x] 2.1 Add `SearchMonsters(name, edition string) (*Monster, error)` method to `MonsterStore` in `store/monster.go`, querying `{ name: name, edition: edition }` with an exact match
- [x] 2.2 Add `SearchMonsters` handler in `api/handler.go`: read `q` and `edition` query params, return HTTP 400 if either is missing or edition is invalid, call `store.GlobalMonsters.SearchMonsters`, return result as a JSON array (empty array if nil)
- [x] 2.3 Register `GET /api/search/monsters` route in `main.go`

## 3. Frontend — Room Creation

- [x] 3.1 Add `edition` to the room state type in `frontend/src/App.tsx` (or wherever the WS state type is defined)
- [x] 3.2 Add an edition toggle/selector (5e / 5.5e, default "5e") to `JoinScreen.tsx` in the "Create New Room" flow
- [x] 3.3 Update `handleCreateRoom` in `JoinScreen.tsx` to include `{ edition }` in the `POST /api/rooms` request body

## 4. Frontend — DM Panel Search

- [x] 4.1 Update the monster lookup in `DMView.tsx` (currently `GET /api/monsters/${name}`) to use `GET /api/search/monsters?q=${name}&edition=${state.edition}` and parse the response as an array (take index 0 if non-empty)

## 5. Verification

- [x] 5.1 Create a 5e room and a 5.5e room; confirm `edition` appears correctly in the WebSocket state broadcast for each
- [x] 5.2 In a 5e room, search for a monster that exists in 5e only — confirm it returns; search for a 5.5e-only monster — confirm empty result
- [x] 5.3 Confirm the DM panel autofills name and max_hp correctly from the search result in both edition rooms
