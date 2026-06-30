# Epic 7: DM Monster Repository and Statblock Drawer

## US7.1: MongoDB Schema Expansion for DM Monsters
**As a** Dungeon Master,  
**I want to** save custom or official monsters in the database,  
**So that** I don't have to retype their max HP or configure their reference links every session.

### Acceptance Criteria:
- **AC 1:** The Go backend must extend the MongoDB `entities` collection or create a new `monsters` collection to support DM-saved creatures.
- **AC 2:** The monster document schema must support:
    - `name` (String, unique identifier for the DM)
    - `max_hp` (Integer)
    - `source_type` (String: restricted to `"URL"` or `"PDF"`)
    - `reference_url` (String, optional, used if `source_type` is `"URL"`)
    - `pdf_file_path` (String, optional, storage path for uploaded file if `source_type` is `"PDF"`)
- **AC 3:** Provide a backend endpoint `POST /api/monsters` that handles both JSON metadata and multipart/form-data for PDF file uploads.

---

## US7.2: Quick-Add Monster and Management Screen
**As a** Dungeon Master,  
**I want** a screen to pre-register monsters and a quick-search dropdown in my combat panel,  
**So that** I can instantly inject saved monsters into an active combat room.

### Acceptance Criteria:
- **AC 1:** Provide a management route (e.g., `/monsters/new`) where the DM can input the monster's base stats and select between linking an external URL (like D&D Beyond or Open5e) or uploading a custom homebrew PDF.
- **AC 2:** On the DM Combat Panel (from US3.2), replace or complement the manual text input with an autocomplete search dropdown that fetches saved monsters from MongoDB.
- **AC 3:** Selecting a saved monster must automatically autofill its `Name` and `Max HP` into the initiative staging area.

---

## US7.3: Slide-Out Statblock Preview Drawer (iFrame / PDF Viewer)
**As a** Dungeon Master,  
**I want** a collapsible drawer next to each monster in the initiative tracker that loads its stats in the background,  
**So that** I can quickly check its attacks and abilities without leaving the app or opening new browser tabs.

### Acceptance Criteria:
- **AC 1:** Every DM-created creature on the tracker that has a saved reference must display a small "Statblock" icon next to its name in the DM panel.
- **AC 2:** Clicking the icon must toggle a slide-out drawer or modal component on the frontend.
- **AC 3:** **Conditional Rendering:** - If the monster's `source_type` is `"URL"`, the drawer must render an `<iframe>` pointing directly to the saved `reference_url`.
    - If the monster's `source_type` is `"PDF"`, the drawer must render an embedded PDF viewer container loading the file stream from the Go backend.
- **AC 4:** To optimize performance, the frontend should lazy-load or background-fetch the resource only when the entity is added to the combat or when the drawer is first initialized.