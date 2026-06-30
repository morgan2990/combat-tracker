## 1. Project Scaffold & Build Pipeline

- [x] 1.1 Initialize Go module (`go mod init`) and create `backend/` directory structure (`main.go`, `room/`, `ws/`, `api/`)
- [x] 1.2 Initialize React app with Vite in `frontend/` (`npm create vite@latest`)
- [x] 1.3 Add `gorilla/websocket` dependency to Go (`go get github.com/gorilla/websocket`)
- [x] 1.4 Write multi-stage Dockerfile: Node build stage → Go build stage (embed frontend dist) → Alpine final image
- [x] 1.5 Verify `go:embed` directive compiles and serves the React `dist/` folder from the Go binary

## 2. In-Memory Room Registry (Go)

- [x] 2.1 Define `Entity` struct with all fields: `id`, `name`, `type`, `owner_id`, `session_id`, `max_hp`, `current_hp`, `temp_hp`, `initiative`, `conditions`
- [x] 2.2 Define `RoomState` struct: `room_id`, `is_started`, `round`, `active_index`, `entities`
- [x] 2.3 Define `Room` struct wrapping `RoomState` with a `sync.RWMutex` and a map of active WS connections keyed by session ID
- [x] 2.4 Implement `RoomRegistry`: a global map of `room_id → *Room` protected by its own mutex, with `CreateRoom()` and `GetRoom()` methods
- [x] 2.5 Implement random room ID generator (5-char alphanumeric, collision-retry loop)
- [x] 2.6 Implement random DM token generator (8 hex characters)

## 3. HTTP API (Go)

- [x] 3.1 Wire up an HTTP router (standard library `net/http` or `chi`) in `main.go`
- [x] 3.2 Implement `POST /api/rooms` handler: generate room ID + DM token, register room, return `{ room_id, dm_token }` with HTTP 201
- [x] 3.3 Register the static file server route for the embedded React app (catch-all serving `index.html` for SPA routing)

## 4. WebSocket Handler (Go)

- [x] 4.1 Implement `GET /ws` upgrade handler: parse `room_id`, `name`, `role`, `dm_token` from query params
- [x] 4.2 Add pre-upgrade validation: 404 if room not found, 403 if DM token mismatch, 409 if player name already taken (all within a write lock)
- [x] 4.3 Register the new connection in the room's connection map with a generated session ID
- [x] 4.4 Implement `broadcastState(room *Room)`: serialize `RoomState` to JSON and write to every connection in the room's connection map
- [x] 4.5 Broadcast full room state immediately after a new client connects
- [x] 4.6 Implement the read loop: listen for incoming client messages (actions will be handled in later epics; for now, just keep the connection alive)
- [x] 4.7 Implement connection cleanup on close: remove from connection map, free the player's name, broadcast updated state to remaining clients
- [x] 4.8 Implement server-side ping loop: goroutine per connection that sends a ping frame every 30 seconds and closes the connection if pong is not received within a deadline

## 5. React Join Screen

- [x] 5.1 Create `JoinScreen` component with fields: Room Code, Character Name, Role selector (Player / DM), and DM Token field (shown only when DM role is selected)
- [x] 5.2 On form submit, open a WebSocket connection to `/ws` with the correct query parameters
- [x] 5.3 Handle WebSocket upgrade rejection codes: show "Room not found" for 404, "Wrong DM token" for 403, "Name already taken" for 409
- [x] 5.4 On successful connection, transition app state from `joining` to `connected` and store the user's role

## 6. React App Shell & State

- [x] 6.1 Set up a React context (or Zustand store) to hold the live `RoomState` received from the server and the current user's role and entity ID
- [x] 6.2 Implement the WebSocket message listener: parse incoming JSON and update the `RoomState` in the store on every `state_update` message
- [x] 6.3 Implement conditional rendering: show `PlayerView` component if role is `player`, show `DMView` component if role is `dm`
- [x] 6.4 Create placeholder `PlayerView` component (displays raw entity list from state — full UI in Epic 2)
- [x] 6.5 Create placeholder `DMView` component (displays raw entity list from state — full UI in Epic 3)
- [x] 6.6 Handle WebSocket `onclose` event in the React app: show a "Disconnected — reconnecting…" banner and attempt to reconnect

## 7. Integration Verification

- [x] 7.1 Run the full stack locally (Go server + React dev server proxied to Go) and verify room creation via `curl POST /api/rooms`
- [x] 7.2 Open two browser tabs, join the same room as DM and as Player, and verify both receive the state broadcast on connect
- [x] 7.3 Close one browser tab and verify the remaining client receives an updated broadcast (connection cleanup works)
- [x] 7.4 Verify the Dockerfile builds successfully and the container serves both the API and the React frontend on a single port
