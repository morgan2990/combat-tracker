# Epic 19: Diseñador y Persistencia de Encuentros Prearmados

## US19.1: MongoDB Schema for Saved Encounters
**As a** Backend Developer,  
**I want to** implement a database collection to store pre-configured combat encounters,  
**So that** DMs can save their tactical setups (monster types, quantities, and custom display names) permanently.

### Technical Note & Context:
An encounter acts as a reusable blueprint. It doesn't store active initiatives or live HP states, but rather the template data needed to inject a group of monsters into an active room instantly.

### Acceptance Criteria:
- **AC 1:** Create a new `encounters` collection in MongoDB.
- **AC 2:** The encounter document schema must include the following fields:
    - `id` (Unique string index)
    - `name` (String, e.g., "Goblin Ambush")
    - `owner_id` (String, linking the blueprint strictly to the unique ID/token of the creator DM)
    - `edition` (String, restricted to `"5e"` or `"5.5e"`)
    - `monsters` (Array of objects, each specifying: `monster_id` matching MongoDB/Typesense, `quantity` integer, and an optional `display_name` string override)
- **AC 3:** Provide backend endpoints for CRUD operations:
    - `POST /api/encounters` (Create/Update an encounter)
    - `GET /api/encounters` (List encounters, strictly filtered by the authenticated `owner_id`)
    - `DELETE /api/encounters/:id` (Remove an encounter, enforcing ownership checks)

---

## US19.2: Encounter Builder Screen (Dashboard Frontend)
**As a** Dungeon Master,  
**I want** a dedicated interface on my dashboard to build and save encounters,  
**So that** I can plan my campaign sessions calmly days before the game starts.

### Acceptance Criteria:
- **AC 1:** On the main landing dashboard screen, inside the **"As DM"** card, add a secondary button labeled `+ New Encounter`.
- **AC 2:** Clicking the button must route the user to an Encounter Builder view (e.g., `/encounters/new`).
- **AC 3:** The screen must feature:
    - A text field for the Encounter Name.
    - An edition selector dropdown (5e / 5.5e).
    - An autocomplete monster search input field leveraging the Typesense endpoint (`GET /api/search/monsters`) filtered by the selected edition.
- **AC 4:** When a monster is selected from the search dropdown, it must be added to a staging list on the screen where the DM can adjust its `quantity` and optionally type a "Custom Display Name / Alias" for that group.
- **AC 5:** Submitting the form must dispatch a payload to `POST /api/encounters`, sending the metadata along with the array of selected monster IDs, quantities, and aliases, then redirecting the DM back to the main dashboard.

---

## US19.3: Inject Pre-made Encounter into Active Room (WebSocket Pipeline)
**As a** Dungeon Master,  
**I want to** load a pre-saved encounter directly from my inner combat panel,  
**So that** all the configured monsters instantly spawn on the tracker without manual search entry mid-session.

### Acceptance Criteria:
- **AC 1:** On the inner DM Combat Panel (inside an active room), add an "Encounter Templates" dropdown menu or button that fetches all blueprints owned by the DM via `GET /api/encounters` matching the room's current edition.
- **AC 2:** Selecting a saved encounter from the list must trigger a specific WebSocket command `inject_encounter` to the Go backend, passing the `encounter_id`.
- **AC 3:** **Backend Expansion Engine:** Upon receiving `inject_encounter`, the Go backend must fetch the encounter blueprint from MongoDB, loop through its `monsters` array, and automatically trigger the standard monster injection logic for each item:
    - Spawn the exact `quantity` specified.
    - Apply the `display_name` alias to all generated instances if present.
    - Read each monster's `initiative_modifier` and trigger independent d20 rolls for each if combat is already active, or stage them with `--` if pre-combat fog of war is active.
- **AC 4:** Once the loop completes, the updated room state must be broadcasted immediately to all connected clients, rendering the newly injected enemies instantly.