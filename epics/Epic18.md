# Epic 18: Quick-Inject Lair Actions Tracker

## US18.1: Lair Action Entity Structure and Support (Go Backend)
**As a** Backend Developer,  
**I want to** support a specialized entity type for environmental hazards,  
**So that** the system can process non-creature initiative elements without requiring health metrics or stats.

### Technical Note & Context:
By rules-as-written in D&D, Lair Actions always execute at initiative count 20, losing ties automatically against any creature that also rolled a 20. We will handle this by injecting a lightweight, specialized runtime entity directly into the room's memory layer and persisting it to MongoDB via the existing snapshot pipeline[cite: 1, 11].

### Acceptance Criteria:
- **AC 1:** Update the `Entity` structure in the Go backend to ensure fields like `max_hp`, `current_hp`, and `initiative_modifier` can accept `nil` or empty states (e.g., using pointers like `*int`)[cite: 10, 11].
- **AC 2:** Implement a new WebSocket incoming message handler `add_lair_action`. When triggered, the backend must instantly append a new object to the room's entities array with:
    - `id`: A unique generated UUID or short string[cite: 11].
    - `name`: `"Lair Action"`
    - `initiative`: `20`
    - `max_hp`: `nil` / `null`[cite: 11]
    - `current_hp`: `nil` / `null`[cite: 11]
    - `is_hidden`: `false` (inherits default entity behavior)
- **AC 3:** Ensure that when `sortEntities()` is called, the sorting algorithm safely handles the static initiative score of 20 and integrates the Lair Action into the normal turn rotation sequence.
- **AC 4:** This special entity must be fully compatible with the MongoDB save/load cycle (`US11.1` / `US11.2`), maintaining its presence across server restarts if the combat is active[cite: 11].

---

## US18.2: Quick-Inject Interface and Specialized Rendering (Frontend View)
**As a** Dungeon Master,  
**I want** a single-click button to inject a Lair Action entry into the combat tracker,  
**So that** I can manage lair-specific hazards smoothly without manual setup.

### Acceptance Criteria:
- **AC 1:** On the DM Combat Panel control bar, place a dedicated button labeled `+ Add Lair Action`.
- **AC 2:** Clicking the button must dispatch the `add_lair_action` command over the active WebSocket channel immediately.
- **AC 3:** **Specialized Tracker Render (DM & Player Views):** Modify the initiative rows component to check if an entity's `max_hp` is `null` / `nil`. If true, the frontend must completely hide the standard health indicators, health bars, delta math input fields, and status badges for that specific row[cite: 2, 3, 5, 11].
- **AC 4:** The Lair Action row must render with distinct styling (e.g., a specific background color theme, or a hazard/shield icon next to the name) in both the DM and Player views to visually isolate it from normal combatants.
- **AC 5:** The DM must still have access to the "Remove" action button for this row, allowing them to delete the Lair Action from the active encounter at any time.