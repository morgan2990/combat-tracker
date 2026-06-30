## Why

The combat tracker needs a foundation for real-time multiplayer sessions: a DM must be able to create a room and share a code with players, and all participants must connect to a shared, live combat state. Without this infrastructure, none of the player or DM features in later epics can function.

## What Changes

- New `POST /api/rooms` endpoint that generates a room ID and DM token
- New WebSocket endpoint (`GET /ws`) that handles role-based connection and authentication
- In-memory room state management in Go (room registry with mutex-guarded access)
- React join screen (enter room code + name + role selection)
- React app shell that renders either the Player view or DM view based on authenticated role
- Multi-stage Dockerfile (Node → Go → Alpine) for a single self-contained container
- WebSocket keepalive ping loop (every 30s) to prevent Cloudflare idle connection drops

## Capabilities

### New Capabilities
- `room-creation`: DM creates a room via HTTP POST and receives a unique room ID and DM token
- `room-connection`: Users join a room via WebSocket using a room code, character name, and role; connection is validated and a live session is established
- `room-state`: In-memory data model for a room's combat state (entities, turn order, round counter), broadcast as full JSON snapshots to all connected clients

### Modified Capabilities
<!-- None — this is a greenfield project -->

## Impact

- **New Go packages**: HTTP router, WebSocket handler, room registry, state broadcaster
- **New React entrypoint**: join flow + role-based view routing
- **Dockerfile**: multi-stage build required; Go binary embeds React `/dist` via `embed.FS`
- **Cloudflare**: WS traffic must pass through Cloudflare proxy; keepalive pings are required to prevent idle timeout disconnects
- **No database dependency**: all state is in-memory; rooms are lost on container restart (intentional for MVP)
