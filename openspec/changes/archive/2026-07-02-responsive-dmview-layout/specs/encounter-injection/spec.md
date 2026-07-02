## MODIFIED Requirements

### Requirement: DM Panel Encounter Templates Dropdown

At the phone tier (viewport width below 768px), the DM Panel SHALL render an "Encounter Templates" dropdown control that fetches `GET /api/encounters?edition=<room's edition>` on open and lists the DM's saved encounters matching the room's edition. At the tablet and desktop tiers (768px and above), the same encounters list SHALL instead render persistently in the DM Nav column, fetched when DMView mounts rather than on a toggle click. In both presentations, selecting an encounter SHALL send an `inject_encounter` WS message with the encounter's `id`.

#### Scenario: Dropdown lists only matching-edition encounters (phone tier)
- **WHEN** the DM Panel opens the Encounter Templates dropdown at a phone-tier viewport width in a `"5e"` room, and the DM owns encounters in both `"5e"` and `"5.5e"`
- **THEN** only the `"5e"` encounters appear in the dropdown list

#### Scenario: DM Nav column lists only matching-edition encounters (tablet/desktop tier)
- **WHEN** DMView renders at a tablet or desktop tier viewport width in a `"5e"` room, and the DM owns encounters in both `"5e"` and `"5.5e"`
- **THEN** the DM Nav column's encounters list shows only the `"5e"` encounters, visible without any click to open it

#### Scenario: Selecting an encounter dispatches injection
- **WHEN** the DM selects an encounter, whether from the phone-tier dropdown or the tablet/desktop DM Nav column list
- **THEN** the frontend SHALL send `{ "type": "inject_encounter", "encounter_id": "<id>" }` over the room's WebSocket connection
