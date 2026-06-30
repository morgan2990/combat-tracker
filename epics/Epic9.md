# Epic 9: Edition Selection and 2024 (5.5e) Compatibility

## US9.1: Room Edition Setting (Backend & State Management)
**As a** Dungeon Master,  
**I want to** select whether my combat room uses D&D 5e (2014) or D&D 5.5e (2024 Rules),  
**So that** the application queries, filters, and loads the correct monster versions and statblocks for that specific game.

### Acceptance Criteria:
- **AC 1:** Expand the room creation payload (`POST /api/rooms`) and the in-memory Go room structure to include an `edition` field (String, strictly restricted to `"5e"` or `"5.5e"`).
- **AC 2:** If no edition is provided during creation, the system must default to `"5e"`.
- **AC 3:** The room state broadcasted via WebSockets must expose this `edition` value to the frontend, allowing the client UI to adjust any edition-specific labeling.

---

## US9.2: Multi-Edition Scrubber Expansion (Go Backend Ingestion)
**As a** System Administrator / DM,  
**I want** the data ingestion engine to parse both 2014 (5e) and 2024 (5.5e) source files from the data repository,  
**So that** the MongoDB database contains distinct entities for both editions.

### Technical Note & Context:
In 5e.tools and its mirrors, 2024 Core Rulebook revisions (often referred to as 5.5e or "XPH" / "MM24") are separated either by specific source book tags (e.g., `source: "MM"` vs `source: "XPH"` / `source: "MM24"`) or distinct subfolders.

### Acceptance Criteria:
- **AC 1:** Update the MongoDB schema for `monsters` to include an indexed `edition` field (`"5e"` or `"5.5e"`).
- **AC 2:** The Go scrubber (`US8.1`) must be updated to process both traditional legacy books and the new 2024 updated rulebook files.
- **AC 3:** During mapping, the engine must look at the creature's origin book source tag:
    - Legacy sources (e.g., MM, VGM, MTF) map to `edition: "5e"`.
    - 2024/2025 Revised sources (e.g., PHB24, MM24) map to `edition: "5.5e"`.
- **AC 4:** If a monster exists in both editions (e.g., *Goblin* in 2014 and *Goblin* in 2024), they must be saved as two distinct documents in MongoDB differentiated by their `edition` field.

---

## US9.3: Edition-Aware Monster Search and Statblock Delivery
**As a** Dungeon Master,  
**I want** the autocomplete monster search bar to only return creatures that match my room's edition,  
**So that** I don't accidentally pull a legacy 2014 monster layout into a 2024 ruleset campaign.

### Acceptance Criteria:
- **AC 1:** Update the backend search endpoint (`GET /api/monsters?search=...&edition=...`) to require the room's current edition as a query parameter.
- **AC 2:** The database query must filter results matching `edition: current_room_edition`.
- **AC 3:** When a monster is selected and injected into combat, the slide-out preview drawer (`US7.3`) must target the specific URL variant or resource path that serves that exact edition's statblock (e.g., appending the correct edition anchors like `.../bestiary.html#goblin_mm` vs `.../bestiary.html#goblin_mm24`).