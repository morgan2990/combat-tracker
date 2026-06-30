## 1. MongoDB Integration (Backend)

- [x] 1.1 Add `go.mongodb.org/mongo-driver` dependency (`go get`)
- [x] 1.2 Create `store/mongo.go` — connect to MongoDB using `MONGODB_URI` env var (default `mongodb://localhost:27017`), database `combatapp`, collection `entities`
- [x] 1.3 Define `store.Profile` struct with fields: `Name`, `Type`, `MaxHP`, `ParentPCName`, `SharesInitiative`
- [x] 1.4 Implement `store.UpsertEntity(p Profile) error` — upsert by `name` field
- [x] 1.5 Implement `store.GetEntityByName(name string) (*Profile, error)` — returns nil if not found
- [x] 1.6 Implement `store.GetCompanionsByParent(parentName string) ([]Profile, error)`
- [x] 1.7 Initialize MongoDB connection in `main.go` on startup; fatal if connection fails

## 2. Profile API Endpoints (Backend)

- [x] 2.1 Add `POST /api/entities` handler in `api/handler.go` — validate payload (name non-empty, type player|companion, max_hp > 0, parent_pc_name required for companion), call `store.UpsertEntity`, return 200 or 400
- [x] 2.2 Add `GET /api/entities/{name}` handler — call `store.GetEntityByName` + `store.GetCompanionsByParent`, return `{ profile, companions }` or 404
- [x] 2.3 Register both routes in `main.go`

## 3. Runtime Entity Model Changes (Backend)

- [x] 3.1 Change `room.Entity.Initiative` from `int` to `*int`; update JSON tag to `json:"initiative"` (null when nil)
- [x] 3.2 Add `SharesInitiative bool` field to `room.Entity`
- [x] 3.3 Update `room.sortEntities` to sort nil initiative values last
- [x] 3.4 Update `room.Client` struct to add `MaxHP int` field (populated from WS query param)
- [x] 3.5 Update `ws/handler.go` WS upgrade to read `max_hp` query param and store on `Client`

## 4. WebSocket Message Changes (Backend)

- [x] 4.1 Remove `MaxHP` field from `setupCharacterMsg`; update `room.SetupCharacter` to take max_hp from `Client.MaxHP` instead of the WS message
- [x] 4.2 Update `room.SetupCharacter` to create entity with `initiative: nil`
- [x] 4.3 Reject `setup_character` in `ws/handler.go` if `client.MaxHP == 0` (no profile loaded)
- [x] 4.4 Add `setInitiativeMsg` struct `{ initiative int }` and `set_initiative` case in `ws/handler.go` dispatch
- [x] 4.5 Implement `room.SetInitiative(sessionID string, initiative int) error` — sets the player entity's initiative and propagates to all companions in the room with `SharesInitiative: true` and matching `OwnerID`
- [x] 4.6 Update `add_companion` message to include `SharesInitiative bool` and make `Initiative *int` (nullable); update `room.AddCompanion` signature accordingly
- [x] 4.7 Add `refresh_from_profile` case in `ws/handler.go` dispatch
- [x] 4.8 Implement `room.RefreshFromProfile(sessionID string, store store.Store) error` — fetches profile + companions from MongoDB, updates `MaxHP` on matching entities in the room, caps `CurrentHP` if needed, broadcasts

## 5. start_combat Pre-flight Check (Backend)

- [x] 5.1 Update `room.StartCombat` to check that every entity with `type == "player"` or `type == "companion"` has a non-nil initiative; return an error if any are nil

## 6. React Router Setup (Frontend)

- [x] 6.1 Install `react-router-dom` (`npm install react-router-dom`)
- [x] 6.2 Wrap `App.tsx` with `<BrowserRouter>`; define three routes: `/` (JoinScreen), `/characters/new` (CharacterForm), `/room` (game view)
- [x] 6.3 Update `App.tsx` to navigate to `/room` after a successful WS connect instead of swapping component state

## 7. Character Creation Screen (Frontend)

- [x] 7.1 Create `frontend/src/components/CharacterForm.tsx` — form with Name, Max HP fields and a dynamic companion section (add/remove rows with Name, Max HP, Shares Initiative toggle per row)
- [x] 7.2 On submit, send `POST /api/entities` for the player profile, then one request per companion in the list
- [x] 7.3 Show success message and link back to `/` on completion
- [x] 7.4 Pre-fill form if `GET /api/entities/:name` returns a profile (load-and-edit flow via name query param or input)

## 8. Join Screen Changes (Frontend)

- [x] 8.1 Add "Find my character" button to the player tab in `JoinScreen.tsx`; wire it to `GET /api/entities/:name`
- [x] 8.2 On profile found: display max_hp as read-only, reveal initiative input field and "Join Room" button
- [x] 8.3 On profile not found (404): show error message with link to `/characters/new`
- [x] 8.4 On fetch error (non-404): show generic "Service unavailable" error and block join
- [x] 8.5 Pass `max_hp` as a query param on the WS connection URL when joining

## 9. Auto-load Companions and set_initiative (Frontend)

- [x] 9.1 After `setup_character` is sent, automatically send one `add_companion` WS message per companion in the fetched profile (name, max_hp, shares_initiative, no initiative)
- [x] 9.2 Add initiative input to the post-join setup form in `PlayerView.tsx` (visible after `setup_character` succeeds, before `set_initiative` is sent)
- [x] 9.3 Wire the initiative input submit to send `{ "type": "set_initiative", "initiative": N }` over the WS

## 10. Refresh from Profile Button (Frontend)

- [x] 10.1 Add a "Refresh from profile" button to the player's own entity card in `PlayerView.tsx`
- [x] 10.2 On click, send `{ "type": "refresh_from_profile" }` over the WS connection

## 11. Frontend Type Updates

- [x] 11.1 Update `types.ts` — `Entity.initiative` changes from `number` to `number | null`
- [x] 11.2 Update `types.ts` — add `shares_initiative: boolean` to `Entity`
- [x] 11.3 Audit all places in the frontend that read `entity.initiative` and guard for null (display as "—" or "pending")
