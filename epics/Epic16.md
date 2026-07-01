# Epic 16: Display Name / Alias de Instancia para Monstruos

## US16.1: Runtime Entity Schema Expansion for Aliases (Go Backend)
**As a** Backend Developer,  
**I want to** extend the in-memory and persistent room entity structure to include an optional display name,  
**So that** each spawned monster instance can carry its own narrative identity distinct from its base template.

### Technical Note & Context:
This changes the volatile state structure of the active room and mirrors it into the MongoDB `rooms` collection snapshot[cite: 11]. The base `name` property continues to hold the original statblock reference[cite: 3].

### Acceptance Criteria:
- **AC 1:** Update the `Entity` struct in the Go backend to include a `display_name` field (`json:"display_name,omitempty"`).
- **AC 2:** Update the MongoDB room state serialization and restoration mechanics (`US11.1` / `US11.2`) to include and persist the `display_name` attribute within the `entities` array[cite: 11].
- **AC 3:** Modify the `add_creature` WebSocket message handling. If the incoming payload contains a string in the custom name field, populate `Entity.display_name` with that string. If the field is blank or omitted, `Entity.display_name` must default to an empty string.
- **AC 4:** **Batch Injection Baseline:** If the DM injects multiple instances of a creature at once (quantity > 1), and a custom name is specified, all entities generated in that specific batch must inherit the exact same string value as their `display_name`.

---

## US16.2: Information Masking and Render Split (Frontend Interface)
**As a** Dungeon Master,  
**I want** my panel to show both the original monster template name and its customized display name, while my players only see the custom alias,  
**So that** I don't lose track of the statblocks I am using while maintaining narrative secrets.

### Acceptance Criteria:
- **AC 1:** Update the Add Creature form overlay inside the DM panel. Add an optional text input field labeled "Custom Display Name / Alias (Optional)".
- **AC 2:** **DM Panel Context:** If an entity has a non-empty `display_name`, the DM initiative tracker must render both names using a dual-label layout (e.g., `"Guard Alpha (Orc Statblock)"` or `"Chief Bugbear (Custom Name)"`). If `display_name` is empty, it simply renders the original `name`[cite: 3].
- **AC 3:** **Player View Masking:** Update the Player View initiative list render. If an entity contains a non-empty `display_name`, the frontend *must only* render the contents of the `display_name` field. It must completely hide the base `name` field from the UI to protect the identity of the statblock[cite: 2].
- **AC 4:** If `display_name` is empty or null, the Player View falls back to rendering the standard `name` property[cite: 2].