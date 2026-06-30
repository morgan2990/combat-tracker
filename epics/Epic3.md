# Epic 3: The DM Panel (Absolute Control)

## US3.1: Turn Flow and Combat Initialization
**As a** Dungeon Master,  
**I want to** lock the initiative order and advance the turns,  
**So that** I can control the pacing of the combat encounter.

### Acceptance Criteria:
- **AC 1:** The DM interface must feature a "Start Combat" button that locks the entry of new natural initiatives (unless added manually mid-combat) and establishes the initial list order.
- **AC 2:** The DM must have a "Next Turn" button. Clicking it shifts the active turn indicator to the next entity down the initiative list.
- **AC 3:** When the "Next Turn" button reaches the last entity on the list, it must automatically loop back to the top (the highest initiative) and increment a "Round Counter" by 1.

---

## US3.2: Ephemeral Creature Management (Monsters/NPCs)
**As a** Dungeon Master,  
**I want to** quickly add and remove monsters using only a name, max HP, and manual initiative,  
**So that** I can improvise encounters on the fly without setting up a persistent database.

### Acceptance Criteria:
- **AC 1:** The DM panel must feature a rapid-input form containing fields for: `Name`, `Max HP`, and `Initiative`.
- **AC 2:** Upon submission, the entity is added directly to the volatile memory of the Go backend room instance. It will not be saved anywhere permanently.
- **AC 3:** The DM must be able to change the current HP of these monsters quickly through delta math fields (e.g., typing `-12` to apply damage or `+5` to heal).
- **AC 4:** Each DM-created creature must feature a "Remove/Kill" button that instantly deletes the entity from the room's memory array and updates all client screens.

---

## US3.3: Total Master Overrides
**As a** Dungeon Master,  
**I want to** have overriding control to modify any value of any entity (Players and Summons included) in the room,  
**So that** I can correct manual mistakes or apply environmental hazards instantly.

### Acceptance Criteria:
- **AC 1:** The DM panel must render edit buttons next to *every single entity* listed on the tracker (including Player Characters).
- **AC 2:** The DM can forcefully change a Player's HP, current status conditions, or initiative order placement.
- **AC 3:** When the DM overrides a player's stat, the backend must force-push the update to that specific player's screen, updating their local UI instantly.