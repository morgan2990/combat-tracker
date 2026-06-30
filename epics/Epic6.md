# Epic 6: Persistent Player Profiles and Companion Management

## US6.1: MongoDB Schema and Backend Entity Storage
**As a** Backend Developer,  
**I want to** implement a MongoDB collection to store user-defined entities (PCs and Companions),  
**So that** players can reuse their characters across multiple combat sessions without manual data reentry.

### Acceptance Criteria:
- **AC 1:** The Go backend must integrate a MongoDB database and define a flexible schema for an `Entity`.
- **AC 2:** The `entities` document schema must include fields for:
    - `name` (String, unique identifier for retrieval)
    - `type` (String, restricted to either `"PC"` or `"Companion"`)
    - `max_hp` (Integer)
    - `parent_pc_name` (String, optional, used only if `type` is `"Companion"` to link it to a main character)
- **AC 3:** Provide a backend endpoint `POST /api/entities` that validates the payload structure and saves or updates the document in the MongoDB collection.

---

## US6.2: Character and Companion Creation Screen
**As a** Player,  
**I want to** have a dedicated screen to create and configure my characters and companions,  
**So that** they are securely saved in the database before any combat begins.

### Acceptance Criteria:
- **AC 1:** The web frontend must include a new route/view (e.g., `/characters/new`) containing a character creation form.
- **AC 2:** The form must collect the `Name` and `Max HP` of the character.
- **AC 3:** The screen must feature a dynamic nested section: "Add Pre-configured Companion/Pet". 
- **AC 4:** Submitting the form must send a batch request to the Go API, storing both the main PC and any associated Companions (properly linked via `parent_pc_name`) into MongoDB.

---

## US6.3: Streamlined Room Join via Character Name Autofetch
**As a** Player,  
**I want to** simply type my character's name when joining a room,  
**So that** the system automatically loads my profile stats and all associated companions into the live tracker.

### Acceptance Criteria:
- **AC 1:** On the room joining form (from US1.2), the character selection field must trigger a profile fetch when the player inputs their saved character's name.
- **AC 2:** The Go backend must query MongoDB for an entity matching that name with `type: "PC"`. 
- **AC 3:** **Cascade Fetching:** The backend query must also find any documents where `parent_pc_name` matches the requested character name to locate its companions.
- **AC 4:** When joining the room successfully, both the PC and all of their linked companions must automatically load into the active room's memory array with their saved `max_hp` and current HP full.
- **AC 5:** If a character name is typed but does not exist in the database, the frontend should show a friendly error message prompting the user to either check the spelling or create the character first.