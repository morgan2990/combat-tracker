# Epic 15: Pre-Combat Monster Fog of War

## US15.1: Room Context Edition-Aware and State Expansion (Go Backend)
**As a** Backend Developer,  
**I want to** ensure that the room's combat state is clearly communicated via WebSockets,  
**So that** the frontend can determine whether the room is in preparation mode or active combat.

### Technical Note & Context:
We will leverage the existing `is_combat_active` boolean field from the in-memory `RoomState` and its MongoDB mirrored document[cite: 11]. The Go backend must guarantee that this flag is dispatched transparently down the WebSocket broadcast pipe to all connected clients every time a room state change occurs[cite: 11].

### Acceptance Criteria:
- **AC 1:** Confirm that the Go backend includes the `is_combat_active` (Boolean) field in the room's root JSON payload transmitted via WebSockets[cite: 11].
- **AC 2:** When a room is freshly initialized or inflated from MongoDB, `is_combat_active` must explicitly default to `false`[cite: 1, 11].
- **AC 3:** When the DM triggers the "Start Combat" action (`US3.1`), the backend must set `is_combat_active = true` and broadcast the updated state immediately to all connections[cite: 3, 11].
- **AC 4:** When the DM triggers the "End Combat" action (`US4.1`), the backend must revert `is_combat_active = false` and broadcast the cleaned state[cite: 4, 11].

---

## US15.2: Client-Side Monster Masking in Staging View (Player Frontend)
**As a** Player,  
**I want** the system to hide any DM-staged monsters while the encounter is being prepared,  
**So that** I don't get spoiled on what enemies we are about to face.

### Acceptance Criteria:
- **AC 1:** The frontend Player View must subscribe to and track the room's `is_combat_active` state parameter in real-time.
- **AC 2:** **Staging Area Filter:** While `is_combat_active` evaluates to `false`, the player's interface must apply an automatic array filter over the incoming entity list before rendering the initiative ladder.
- **AC 3:** The filter must follow this visibility matrix when `is_combat_active === false`:
    - **Render:** All Player Characters (PCs) and any active Player Companions/Summons[cite: 6].
    - **Hide/Omit:** All DM-controlled entities (Monsters, NPCs, Hostile creatures).
- **AC 4:** The moment `is_combat_active` transitions to `true` (triggered by the DM clicking "Start Combat"), the frontend must automatically disable this filter, smoothly revealing the full list of monsters alongside their calculated initiative rolls[cite: 3, 10, 11].
- **AC 5:** **DM Exclusion:** This filtering logic must **never** apply to the DM Panel view. The DM must always see all staged and pending monsters in their dashboard to manage modifiers and verify the staging layout[cite: 10].