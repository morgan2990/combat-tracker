# Epic 1: Room Management and Connectivity (Core / Infrastructure)

## US1.1: Combat Room Creation (Backend)
**As a** Dungeon Master,  
**I want to** create a new combat room through the API,  
**So that** I can generate a unique space to manage the encounter.

### Acceptance Criteria:
- **AC 1:** Given a `POST /api/rooms` endpoint, the API must respond with a unique and short `room_id` (e.g., a 5-6 alphanumeric character code).
- **AC 2:** The API must initialize an empty combat state for that specific room (no players, no creatures in memory).
- **AC 3:** The creator of the room must receive a unique session token or key identifying them exclusively as the **Dungeon Master (DM)** for that room.
- **AC 4:** The room state must be managed in-memory within the Go backend server.

---

## US1.2: Connection and Role Selection (Frontend / Backend)
**As a** User (DM or Player),  
**I want to** enter a room code and select my role or character,  
**So that** I can join the live combat session.

### Acceptance Criteria:
- **AC 1:** The web interface must present a form requiring a `room_id` and the user's name/character name.
- **AC 2:** If the user selects the **DM** role, they must provide the master token/password created in US1.1.
- **AC 3:** If the user selects the **Player** role, the backend must validate that the character name is not already taken or duplicated within that active room.
- **AC 4:** Upon validation, a persistent connection (via WebSockets) must be established to stream real-time state updates to the client.