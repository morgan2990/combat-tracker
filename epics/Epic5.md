# Epic 5: Quality of Life Improvements and State Refinements

## US5.1: UI Room Creation Button (Frontend & Backend Integration)
**As a** Dungeon Master,  
**I want to** click a "Create Room" button on the landing page,  
**So that** I don't have to trigger a manual cURL command to initialize a session.

### Acceptance Criteria:
- **AC 1:** The frontend landing page must feature a visible, dedicated "Create Room" button for DMs.
- **AC 2:** Clicking the button must automatically trigger the `POST /api/rooms` request to the Go backend.
- **AC 3:** Upon a successful backend response, the frontend must automatically capture the generated `room_id` and DM token, redirecting the user straight into their newly created DM panel.
- **AC 4:** The user should not have to manually copy-paste codes or tokens to access the room they just created.

---

## US5.2: Instant HP Zeroing on "Kill" Action (DM Panel)
**As a** Dungeon Master,  
**I want** the "Kill" button to automatically set a creature's current HP to 0 in addition to changing its status,  
**So that** the tracker accurately reflects that the monster has fallen without requiring manual math.

### Acceptance Criteria:
- **AC 1:** When the DM clicks the "Kill / Remove" button on any DM-created creature, the Go backend must update two fields simultaneously:
    - Set the status condition to `Dead`.
    - Force the current `HP` value to `0`.
- **AC 2:** This combined state change must be broadcasted immediately to all connected clients.
- **AC 3:** *(Regression Check)* This action must not delete the entity from memory until Epic 4's "End Combat" is triggered, allowing players to still see the dead creature in the initiative order if necessary.

---

## US5.3: "Dead" Status Condition for Players and Companions (Player View)
**As a** Player,  
**I want** the system to clearly display the "Dead" status on my character and my companions when our HP reaches 0,  
**So that** the rest of the party and I are visually aware of our critical condition.

### Acceptance Criteria:
- **AC 1:** The player interface must automatically append/display a distinct "Dead" status badge next to their Character or Companion's name whenever their current HP reaches `0`.
- **AC 2:** If the DM uses the updated "Kill" button (from US5.2) or an override on a Player's Companion, the "Dead" status must immediately render on that player's screen via the WebSocket update.
- **AC 3:** The visual styling for the "Dead" status should be highly visible (e.g., greyed out row or a red indicator) to clearly differentiate it from standard conditions like *Prone* or *Stunned*.