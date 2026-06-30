# Epic 11: Room State Persistence

## US11.1: Room Session Persistence and Auto-Save Pipeline
**As a** Dungeon Master,  
**I want** the state of my combat room to be persisted in MongoDB,  
**So that** the session isn't lost if the server restarts, and I can resume the encounter exactly where we left off.

### Technical Note & Context:
To maintain high performance, the live, rapid combat updates should still run via WebSockets in the Go memory layer. However, the state must be mirrored to MongoDB periodically (debounce/throttle) or upon critical state-changing events.

### Acceptance Criteria:
- **AC 1:** Create or extend a `rooms` collection in MongoDB to store the complete snapshot of a room's state.
- **AC 2:** The persisted room document must include:
    - `room_id` (Unique string index)
    - `dm_token` (String)
    - `is_combat_active` (Boolean)
    - `current_round` (Integer)
    - `active_turn_entity_id` (String, nullable)
    - `edition` (String, `"5e"` or `"5.5e"`)
    - `entities` (Array of objects containing the current runtime stats: ID, name, current_hp, max_hp, initiative, conditions, and connection status).
- **AC 3:** **Triggered Persistence:** The Go backend must write the updated room state to MongoDB whenever:
    - A player joins or leaves the room.
    - Combat is officially started (`US3.1`) or ended (`US4.1`).
    - The turn advances to the next entity ID.
    - A periodic ticker completes (e.g., auto-save snapshot every 30 seconds if changes occurred).

---

## US11.2: Room State Restoration on Launch
**As a** Dungeon Master or Player,  
**I want** the system to automatically load the active state from the database when accessing an existing room URL,  
**So that** we can rejoin a disconnected session instantly.

### Acceptance Criteria:
- **AC 1:** When a client hits the connection endpoint or connects via WebSocket using an existing `room_id`, the Go backend must first check if the room instance exists in its local memory pool.
- **AC 2:** If the room is *not* in memory (e.g., the server crashed or restarted), the backend must query MongoDB for the matching `room_id`.
- **AC 3:** If found, the Go backend must inflate the room structure back into its volatile memory pool, restore the active WebSocket channels, and sync all existing entity statuses.
- **AC 4:** Once restored, the backend must push the complete room context down the WebSocket pipeline, allowing the frontend to render the exact state (health, round, conditions, and entity positions) before the interruption.
- **AC 5:** If the room ID does not exist in memory or MongoDB, the API should return a proper `404 Not Found` error.