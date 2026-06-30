# Epic 2: The Player View (Restricted & Focused)

## US2.1: Initiative Tracker Visualization
**As a** Player,  
**I want to** view the ordered initiative list and the general status of my party members,  
**So that** I know when my turn is coming and can plan my actions.

### Acceptance Criteria:
- **AC 1:** The frontend must display a list of all active combatants ordered from highest to lowest according to their current initiative score.
- **AC 2:** The current active turn must be clearly highlighted with a distinct visual indicator.
- **AC 3:** **Fog of War / Information Restriction:** Players must **not** see the exact current or maximum HP of DM-controlled creatures (monsters). They should only see a qualitative indicator (e.g., *Healthy, Injured, Dying*) or just the monster's name and initiative.
- **AC 4:** Players must see the exact HP and status effects of other party members (PCs).

---

## US2.2: Character State Modification
**As a** Player,  
**I want to** update my own character's hit points (HP) and status conditions,  
**So that** I can keep my sheet updated during combat without overwhelming the DM.

### Acceptance Criteria:
- **AC 1:** The system must restrict editing capabilities so that a player can *only* modify the HP, Temporary HP, and status conditions (e.g., *Prone, Stunned, Poisoned*) of the entity tied to their specific player session.
- **AC 2:** Any modification made by the player must be immediately sent to the Go backend and broadcasted via WebSockets to all other connected clients in real-time.
- **AC 3:** If a player tries to send a payload modifying another player's or creature's ID, the Go backend must reject the request with an unauthorized action error.

---

## US2.3: Companion / Summon Management
**As a** Player with specific classes (e.g., Ranger, Wizard),  
**I want to** add and control a companion or summoned creature tied to my character,  
**So that** I can track its initiative and HP independently.

### Acceptance Criteria:
- **AC 1:** The player interface must provide an "Add Summon/Pet" button.
- **AC 2:** The form will require a `Name`, `Max HP`, and a manually rolled `Initiative`. 
- **AC 3:** The Go backend must register this new entity in the room's memory and link it to the `player_id` of the owner.
- **AC 4:** The owner player receives full edit permissions over this companion's HP and conditions. Other players cannot edit it.
- **AC 5:** If the player disconnects, their companion entities must remain in the room until manually removed by the DM.