# Epic 10: Automated Monster Initiative Rolling

## US10.1: MongoDB Schema Update for Initiative Modifiers
**As a** Backend Developer,  
**I want to** update the monster database schema to include initiative/dexterity modifiers,  
**So that** the Go backend has the necessary data to automate rolls at the appropriate time.

### Technical Note:
In D&D 5e/5.5e, the initiative modifier is derived directly from the creature's Dexterity modifier. In the 5e.tools JSON, this can be parsed from the `dex` attribute (e.g., a score of `14` yields a `+2` modifier).

### Acceptance Criteria:
- **AC 1:** Expand the MongoDB `monsters` collection schema to include an `initiative_modifier` field (Integer, can be positive, negative, or zero).
- **AC 2:** Update the Data Scrubber (`US8.1` / `US9.2`) to calculate and extract this modifier during JSON ingestion. 
    - *Formula:* `modifier = floor((dexterity - 10) / 2)`
- **AC 3:** Ensure that manual monster creation (`US7.2`) also includes an optional `Initiative Modifier` numerical field in the UI form.

---

## US10.2: Conditional Initiative Engine (Go Backend State Logic)
**As a** Dungeon Master,  
**I want** the system to automatically roll a d20 + modifier for monsters either when I start combat or immediately upon adding them if combat is already running,  
**So that** initiative is always calculated dynamically based on the current state of the room.

### Acceptance Criteria:
- **AC 1:** The Go backend must evaluate the room's status (`is_combat_active` boolean) to determine when to trigger the Random Number Generation (RNG) engine:
    - **Scenario A (Pre-combat setup):** If the DM adds saved monsters *before* clicking "Start Combat", the monsters are staged in the room with an uncalculated initiative value.
    - **Scenario B (Combat Trigger):** When the DM clicks "Start Combat", the Go backend must loop through all staged monsters, execute a $d20 + modifier$ roll for each, and assign the resulting score directly to each specific entity's data payload.
    - **Scenario C (Mid-combat reinforcements):** If a saved monster is added *after* combat has already started, the Go backend must execute the $d20 + modifier$ roll **instantly** and assign the score to the new entity upon creation.
- **AC 2:** **ID-Driven Association:** Every rolled or modified initiative value must be tied directly to its unique entity ID, ensuring that each creature carries its own individual calculated score at all times.
- **AC 3:** The visual turn indicator and the UI state must rely strictly on the entity IDs to track state transitions, ensuring that adding new creatures mid-combat seamlessly maps them into the room based on their assigned score.

---

## US10.4: UI Feedback and Staging Area
**As a** Dungeon Master,  
**I want to** see which monsters are pending an initiative roll before combat starts, and see their mathematical breakdown after the roll,  
**So that** I can manage upcoming encounters cleanly.

### Acceptance Criteria:
- **AC 1:** In the pre-combat staging view, monsters without a calculated initiative should display a placeholder icon (e.g., a grayed-out `d20` icon or `--`).
- **AC 2:** Once the rolls are triggered (either by starting the combat or as mid-combat reinforcements), the DM Panel must display the final score along with a small tooltip or indicator showing the breakdown (e.g., `16 (Rolled: 13 + Mod: +3)`).
- **AC 3:** *(Override Safety)* The DM must still be able to manually edit the final initiative value at any time using the override controls (`US3.3`) in case they want to adjust the turn order manually.
- **AC 4:** Every automated calculation must trigger a WebSocket state update to re-sort the initiative ladder from highest to lowest for all connected clients.