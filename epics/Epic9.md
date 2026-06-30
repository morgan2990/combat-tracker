# Epic 9: Edition Selection and Room Configuration

## US9.1: Room Edition Setting (Backend & State Management)
**As a** Dungeon Master,  
**I want to** select whether my combat room uses D&D 5e (2014) or D&D 5.5e (2024 Rules) when I create it,  
**So that** the application queries, filters, and loads the correct monster versions and statblocks for that specific campaign.

### Acceptance Criteria:
- **AC 1:** Expand the in-memory Go `RoomState` struct to include an `edition` field (String, restricted to `"5e"` or `"5.5e"`).
- **AC 2:** Expand the room creation payload (`POST /api/rooms`) to accept an optional `edition` field. If omitted or invalid, the backend must default to `"5e"`.
- **AC 3:** The room state broadcast via WebSockets must expose the `edition` value to all connected clients so the frontend can use it for edition-specific logic.
- **AC 4:** The frontend room creation screen must present an edition selector (5e / 5.5e) before the DM submits the form.

---

## US9.2: Multi-Edition Scrubber *(Superseded by Epic 8)*
This user story has been fully addressed by the implementation of Epic 8 (`US8.1` / `US8.2`). The scrubber uses an explicit `--edition` flag and separate 5etools repositories per edition, which supersedes the source-tag-based approach originally described here. No additional work required.

---

## US9.3: Edition-Aware Monster Search Endpoint
**As a** Dungeon Master,  
**I want** the monster search bar to return results filtered to my room's edition,  
**So that** I don't accidentally pull a 5e (2014) monster into a 5.5e (2024) campaign or vice versa.

### Technical Note & Context:
This endpoint establishes the contract that Epic 12 (Typesense autocomplete) will fulfill. For now it uses a MongoDB exact-name match filtered by edition. When Epic 12 lands, the frontend contract (`GET /api/search/monsters?q=...&edition=...`) stays unchanged — only the backend query is swapped for Typesense.

### Acceptance Criteria:
- **AC 1:** Expose a backend endpoint `GET /api/search/monsters?q=<name>&edition=<edition>`. Both parameters are required; return HTTP 400 if either is missing or `edition` is not `"5e"` or `"5.5e"`.
- **AC 2:** The endpoint must query the `monsters` collection for an exact match on `{ name: q, edition: edition }` and return the matching document as a JSON array (empty array if not found).
- **AC 3:** On the DM Combat Panel, the monster search input must dispatch requests to `GET /api/search/monsters` using the room's current `edition` from the WebSocket state, replacing the existing exact-name lookup.
