# Epic 14: Custom Monster Privacy and Ownership Scope

## US14.1: Database Schema Expansion for Private Flags
**As a** Backend Developer,  
**I want to** add a privacy flag and an owner field to the monster database schema,  
**So that** the system can restrict access to custom creatures based on who created them.

### Technical Note & Context:
We need to track which unique account or token created a custom monster. Official monsters imported via the scrubber remain public and accessible to everyone.

### Acceptance Criteria:
- **AC 1:** Extend the MongoDB `monsters` collection schema and the corresponding Go struct with two new fields:
    - `private` (Boolean, defaults to `false` if omitted).
    - `owner_id` (String, nullable, stores the unique identifier or token of the DM who created it).
- **AC 2:** Extend the Typesense `monsters` index schema to include the `private` (bool) and `owner_id` (string) fields to ensure the search cache can replicate this visibility logic.
- **AC 3:** Update the manual monster creation service (`POST /api/monsters`) so that the incoming payload accepts the `private` boolean from the frontend, automatically extracts the logged-in DM's identity, and persists both values into MongoDB and Typesense.

---

## US14.2: Owner-Scoped Search Filtering (Go Backend & Typesense)
**As a** Dungeon Master,  
**I want** the monster search endpoint to filter out other users' private creatures,  
**So that** my custom campaign secrets remain hidden from players who might be DMs in other rooms.

### Acceptance Criteria:
- **AC 1:** Update the search endpoint (`GET /api/search/monsters`) to require authentication or identification of the requesting DM.
- **AC 2:** Modify the Typesense search query logic to enforce strict multitenancy using a composite `filter_by` string. The query must only return documents that match any of the following conditions:
    - Official monsters: `is_custom:=false`
    - Public custom monsters: `is_custom:=true && private:=false`
    - Private personal monsters: `is_custom:=true && private:=true && owner_id:=<requesting_dm_id>`
- **AC 3:** If a DM attempts to directly request a monster profile via an explicit API call by ID, the Go backend must return a `403 Forbidden` error if the monster is marked as private and the `owner_id` does not match the requester.

---

## US14.3: Privacy Toggle on Monster Form (Frontend View)
**As a** Dungeon Master,  
**I want to** easily toggle whether a new creature is private or public when filling out the form,  
**So that** I can share generic statblocks with the community or keep boss fights entirely to myself.

### Acceptance Criteria:
- **AC 1:** Update the `MonsterForm.tsx` component to include a checkbox or toggle input labeled "Mark as Private Campaign Content".
- **AC 2:** Provide a small informational tooltip next to the toggle explaining that private monsters will only be visible to them in their dashboard and active rooms.
- **AC 3:** Ensure that when editing an existing custom monster, the current privacy state is correctly loaded from the backend and reflected on the toggle switch.