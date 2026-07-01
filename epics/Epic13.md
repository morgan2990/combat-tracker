# Epic 13: Dashboard Custom Monster Creation

## US13.1: Relocate Custom Monster Form to Main Dashboard
**As a** Dungeon Master,  
**I want to** access the custom monster creation form directly from my main dashboard,  
**So that** I can prepare my homebrew creatures in advance without needing to open an active combat room.

### Technical Note & Context:
Currently, the monster creation form is only accessible from within an active combat room. This user story moves that entry point to the user's landing dashboard, mirroring the player's character creation placement and ensuring a cleaner separation of concerns between campaign preparation and live session management.

### Acceptance Criteria:
- **AC 1:** Completely remove the custom monster creation button and its associated form trigger from the inner combat room panel interface.
- **AC 2:** On the main landing dashboard screen (visible immediately after logging in), add a new, clearly visible button labeled `+ New Monster` (or `+ New Creature`) inside the **"As DM"** card container.
- **AC 3:** The positioning and visual styling of the `+ New Monster` button must be consistent with the existing `+ New Character` button located inside the **"As Player"** card.
- **AC 4:** Clicking the `+ New Monster` button must navigate the user to or open the monster registration form view (`MonsterForm.tsx`).
- **AC 5:** Submitting the completed form must dispatch the data to the Go backend (`POST /api/monsters`), saving the entity directly into MongoDB and updating the Typesense search index as established in previous epics.
- **AC 6:** Upon successful creation, the form must redirect the DM back to the main dashboard screen, providing a temporary success notification toast or message.