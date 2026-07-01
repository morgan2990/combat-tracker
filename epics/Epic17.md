# Epic 17: Client-Side Hidden Entities (Invisibility & Ambushes)

## US17.1: Runtime Entity Expansion for Hidden Flags (Go Backend)
**As a** Backend Developer,  
**I want to** add an `is_hidden` boolean property to the combat entity structure,  
**So that** the state can toggle and broadcast whether a creature should be visible to players.

### Technical Note & Context:
To keep the Go backend highly performant and avoid the overhead of filtering slices or managing separate WebSocket channels per role, the full room payload will continue to be broadcasted to all connected clients. The masking security will be enforced at the user-interface layer on the client side.

### Acceptance Criteria:
- **AC 1:** Update the `Entity` struct in the Go backend to include an `is_hidden` boolean field (`json:"is_hidden"`), which must default to `false` when an entity is initialized or added to the room.
- **AC 2:** Ensure that the MongoDB room state saving and restoration pipeline (`US11.1` / `US11.2`) includes and persists this new `is_hidden` field inside the `entities` array[cite: 11].
- **AC 3:** Expose or expand a WebSocket message handler (e.g., `toggle_entity_visibility`) that allows the DM to flip the `is_hidden` boolean value of a specific entity ID and instantly triggers a room broadcast to all connections.

---

## US17.2: Visibility Toggle Controls (DM Panel View)
**As a** Dungeon Master,  
**I want** a quick toggle button next to each monster in my tracker,  
**So that** I can easily hide or reveal creatures as they slip into invisibility or jump out from an ambush.

### Acceptance Criteria:
- **AC 1:** On the DM Combat Panel initiative tracker rows, render a dedicated visibility toggle icon button (e.g., an eye icon) next to every creature[cite: 3].
- **AC 2:** The button must visually reflect the current state: an open eye icon if `is_hidden` is `false`, and a slashed eye icon (or greyed out indicator) if `is_hidden` is `true`.
- **AC 3:** Clicking the icon must dispatch the visibility toggle action over the WebSocket channel to the Go backend instantly.
- **AC 4:** The DM panel must **always** render hidden entities in the list regardless of the flag's value, using a distinct visual styling (e.g., 50% opacity background) so the DM knows at a glance which monsters are currently unseen by the players.

---

## US17.3: Client-Side Visibility Filter (Player Frontend View)
**As a** Player,  
**I want** the initiative tracker to automatically omit any creatures marked as hidden,  
**So that** I don't accidentally get spoiled on invisible enemies or unrevealed reinforcements.

### Acceptance Criteria:
- **AC 1:** Update the Player View initiative list component to read and evaluate the `is_hidden` property of each entity inside the real-time WebSocket room state payload.
- **AC 2:** Before rendering the initiative ladder array on the screen, the player interface must apply an explicit filter: `entities.filter(entity => !entity.is_hidden)`.
- **AC 3:** Any entity where `is_hidden === true` must be completely omitted from the rendered DOM, preventing players from seeing its name, its current status, or its placement order in the turn cycle.
- **AC 4:** If the DM toggles `is_hidden` back to `false`[cite: 3], the incoming WebSocket update must cause the frontend to instantly re-render, smoothly making the creature appear in its exact position on the player's initiative list.