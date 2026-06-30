## Context

Greenfield project. No existing code. The goal is to build the foundational session layer for a real-time D&D combat tracker: room creation, role-based joining, and live state synchronization over WebSockets. The system runs as a single Go binary serving both the HTTP API and embedded React frontend, deployed in a Docker container behind Cloudflare.

## Goals / Non-Goals

**Goals:**
- Room creation HTTP endpoint returning a unique room ID and DM token
- WebSocket endpoint with role-based authentication (DM vs. Player)
- In-memory room registry with concurrent-safe state management
- Full-state broadcast to all connected clients on any mutation
- React join screen and role-based view shell (Player view / DM view)
- Cloudflare-compatible WebSocket keepalive

**Non-Goals:**
- Persistent storage or database of any kind
- Room expiry or cleanup (rooms live until server restart)
- Horizontal scaling or multiple server instances
- Fog of War enforcement at the server layer (handled by the frontend)
- Player reconnection via session token (rejoin by name)

## Decisions

### 1. Full state broadcast over delta events

**Decision:** On any mutation, serialize the complete `RoomState` and broadcast it to every connected client.

**Rationale:** A D&D combat room has at most ~15 entities. The full state payload is 2-4 KB. Delta reconciliation adds complexity (event ordering, client-side divergence) with no meaningful benefit at this scale.

**Alternative considered:** Event-sourced deltas (`{ type: "hp_change", entity_id, delta }`). Rejected because it requires clients to apply deltas correctly and maintain consistency — not worth it for this scale.

---

### 2. Fog of War on the frontend only

**Decision:** The server broadcasts full entity data (including exact creature HP) to all clients. The React player view simply does not render exact HP for creature-type entities.

**Rationale:** The player base is a trusted friend group. Inspecting WebSocket frames to cheat on monster HP is not a realistic threat. Server-side per-role payload filtering would require iterating connections and serializing differently per role — significant complexity for zero practical gain.

**Alternative considered:** Server sends `{ hp_label: "Injured" }` to player connections. Rejected due to complexity without benefit.

---

### 3. DM token as a simple shared password

**Decision:** When creating a room, the DM receives a short random token. They must provide this token when connecting as DM via WebSocket. The server validates it against the stored value for that room.

**Rationale:** The DM token only needs to prevent accidental role confusion, not protect against adversarial users. A simple random string (e.g., 8 hex characters) is sufficient.

**Alternative considered:** Cryptographic JWT session tokens. Rejected as unnecessary complexity for a private friends game.

---

### 4. WebSocket connection parameters via query string

**Decision:** Clients pass `room_id`, `name`, `role`, and optionally `dm_token` as query parameters on the WebSocket upgrade request (e.g., `ws://host/ws?room_id=X7K2P&name=Aragorn&role=player`).

**Rationale:** WebSocket upgrade requests cannot carry a body; query params are the idiomatic way to pass connection metadata before the WS handshake is complete. This avoids a pre-auth HTTP round-trip and keeps the join flow simple.

---

### 5. Go serves the React frontend via embed.FS

**Decision:** The React build output (`/dist`) is embedded into the Go binary at compile time using `embed.FS`. Go serves static files directly with no separate web server.

**Rationale:** Single binary, zero runtime file system dependencies, clean container image. The multi-stage Dockerfile (Node → Go → Alpine) handles the build pipeline.

---

### 6. WebSocket keepalive ping every 30 seconds

**Decision:** The Go server sends a WebSocket ping frame to each connected client every 30 seconds. Clients must respond with a pong (browser WebSocket API handles this automatically).

**Rationale:** Cloudflare drops idle WebSocket connections after ~100 seconds. A 30-second ping interval provides a safe margin. The ping loop runs as a goroutine per connection.

---

### 7. WebSocket library: gorilla/websocket

**Decision:** Use `github.com/gorilla/websocket` for WebSocket handling in Go.

**Rationale:** Battle-tested, widely used, full-featured (ping/pong support, read/write deadlines). The newer `nhooyr.io/websocket` is also good, but gorilla has more community examples and is a safe default for a solo project.

## Risks / Trade-offs

- **In-memory state lost on restart** → Accepted for MVP. Sessions are short (2-4 hours); a restart during a session requires the DM to re-create the room.
- **Cloudflare WS idle timeout** → Mitigated by 30s server-side ping loop.
- **Name collision race condition** → Mitigated by a `sync.RWMutex` on the room struct; name validation and registration happen inside a write lock.
- **No reconnection identity** → A player who refreshes simply rejoins with the same name. If their name is still registered (old connection not yet cleaned up), the server must detect the stale connection and allow the rejoin. Cleanup on WS close handles this.
- **Single container, no horizontal scaling** → Acceptable; this is a private game session, not a public service.
