# Epic 12: Typesense Fast Autocomplete for Monsters

## US12.1: Typesense Schema Definition and Data Sync Pipeline
**As a** Backend Developer,  
**I want to** mirror the MongoDB monster collection into a Typesense search index,  
**So that** the application can perform ultra-fast, typo-tolerant autocomplete queries on monster names.

### Technical Note & Context:
MongoDB will remain the source of truth for persistent data storage. Typesense will act as a dedicated search-memory layer. The Go backend must handle data indexing and synchronization.

### Acceptance Criteria:
- **AC 1:** The Go backend must initialize a Typesense collection named `monsters` with the following schema fields:
    - `id` (String, matching the MongoDB document ID)
    - `name` (String, facet: true)
    - `max_hp` (Int32)
    - `initiative_modifier` (Int32)
    - `edition` (String, facet: true)
- **AC 2:** **Scrubber Integration:** Update the Data Scrubber pipeline (`US8.1` / `US9.2`) so that after bulk-inserting documents into MongoDB, it simultaneously pushes or batches those documents to the Typesense `monsters` index.
- **AC 3:** Update the manual monster creation service (`US7.2`) so that whenever a DM saves a custom monster, the backend performs a dual-write (updates MongoDB first, then upserts into Typesense).

---

## US12.2: Typo-Tolerant Autocomplete Search Endpoint (Go Backend)
**As a** Dungeon Master,  
**I want** the search API to query Typesense instead of MongoDB when I type in the monster search bar,  
**So that** I get instant, filtered, and typo-tolerant results even if I misspell a creature's name.

### Acceptance Criteria:
- **AC 1:** Expose a backend endpoint `GET /api/search/monsters?q=...&edition=...`.
- **AC 2:** This endpoint must forward the query string `q` and the room's current `edition` parameter directly to the Typesense client driver.
- **AC 3:** Configure the Typesense query configuration to enable:
    - Typo tolerance (e.g., matching "Goblen" to "Goblin" using a Levenshtein distance of 1 or 2).
    - Prefix matching (so typing "Beh" instantly matches "Beholder").
    - Strict filtering based on the `edition` field (`filter_by: "edition:=5e"` or `"edition:=5.5e"`).
- **AC 4:** The API must return a clean, lightweight JSON array containing the `id`, `name`, `max_hp`, and `initiative_modifier` of the top matching hits.

---

## US12.3: Real-Time Autocomplete Dropdown Menu (Frontend View)
**As a** Dungeon Master,  
**I want** the monster search input field to display an instant dropdown list of matches only after I have typed 3 or more characters,  
**So that** the system avoids firing unnecessary queries for extremely broad single or double-letter terms.

### Acceptance Criteria:
- **AC 1:** On the DM Combat Panel, the monster search input field must implement a character-length check. The system must **not** fire any API requests to `/api/search/monsters` if the input text contains fewer than 3 characters.
- **AC 2:** Once the user has typed 3 or more characters, the frontend must apply a debounce mechanism (e.g., waiting 150-200ms after the user stops typing) before dispatching the search request to the Go backend.
- **AC 3:** If the user deletes characters and the input length drops back below 3, the frontend must instantly clear and close the dropdown menu without hitting the API.
- **AC 4:** The matching results must render instantly in a clean dropdown overlay below the text box, displaying the monster's name, its edition badge (`5e` or `5.5e`), and its base max HP for rapid visual identification.
- **AC 5:** Pressing `Enter` or clicking on a dropdown item must select that creature, clear the search text box, automatically autofill its stats into the staging area, and ready it for initiative calculation.