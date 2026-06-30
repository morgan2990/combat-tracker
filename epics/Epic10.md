# Epic 10: Automated Monster Initiative Rolling

## US10.1: MongoDB Schema Update for Initiative Modifiers
**As a** Backend Developer,
**I want to** update the monster database schema to properly represent initiative modifiers,
**So that** the Go backend can auto-roll initiative accurately for all monsters, distinguishing "no modifier set" from "modifier is zero."

### Technical Note:
In D&D 5e/5.5e, the initiative modifier equals the creature's Dexterity modifier. The 5e.tools scrubber already computes `floor((dex - 10) / 2)` and stores it. Custom monsters registered via the UI may omit the modifier entirely, which must be preserved as a distinct nil state.

### Acceptance Criteria:
- **AC 1:** `Monster.InitiativeModifier` is `*int` in the MongoDB schema. Nil means "not set"; a pointer to any integer (including 0 or negative) means "modifier is known."
- **AC 2:** The Data Scrubber (Epic 8) always sets a non-nil pointer using `floor((dex - 10) / 2)`.
- **AC 3:** The manual monster registration form (`MonsterForm.tsx`) gains an `edition` selector (fixing the existing 400 bug) and an optional `Initiative Modifier` integer field. If left blank, the field is omitted from the JSON payload so the backend stores nil.

---

## US10.2: Auto-Roll Engine and WebSocket Protocol (Go Backend)
**As a** Dungeon Master,
**I want** the system to automatically roll d20 + modifier for monsters either when I start combat or immediately upon adding them mid-combat,
**So that** initiative is calculated dynamically and each creature carries its own independent roll.

### Technical Note:
The `Entity` struct gains `InitiativeModifier *int` (captured at add-time from the WS message) and `InitiativeRoll *int` (the raw d20 face, set when the roll fires). The `add_creature` WS message replaces its `initiative int` field with `initiative_modifier *int`. The auto-roll uses `crypto/rand` (already in scope) to produce a uniformly random integer 1–20.

### Acceptance Criteria:
- **AC 1:** `Entity` struct has `InitiativeModifier *int` (`json:"initiative_modifier,omitempty"`) and `InitiativeRoll *int` (`json:"initiative_roll,omitempty"`).
- **AC 2:** The `add_creature` WS message carries `initiative_modifier *int` instead of `initiative int`.
- **AC 3:** Backend rolling scenarios:
  - **Scenario A (pre-combat staging):** Creature added before "Start Combat" is staged with `InitiativeModifier` set and `Initiative` nil. No roll yet.
  - **Scenario B (StartCombat trigger):** `StartCombat` loops over all creature entities with non-nil `InitiativeModifier` and nil `Initiative`, rolls d20 for each, sets `Initiative = roll + modifier` and `InitiativeRoll = roll`, then calls `sortEntities()`.
  - **Scenario C (mid-combat reinforcement):** Creature added after `IsStarted == true` with non-nil `InitiativeModifier` is rolled immediately; `Initiative` and `InitiativeRoll` are set before appending to entities.
- **AC 4:** When quantity > 1, each creature in the batch receives its own independent d20 roll (not a shared roll).
- **AC 5:** Nil `InitiativeModifier` means no auto-roll; `Initiative` stays nil and the DM sets it manually via the existing override controls.

---

## US10.4: UI Feedback and Staging Area
**As a** Dungeon Master,
**I want to** see which monsters are pending a roll before combat starts, and see their roll breakdown afterward,
**So that** I can manage upcoming encounters cleanly and verify the numbers.

### Technical Note:
The frontend `Entity` type gains `initiative_modifier: number | null` and `initiative_roll: number | null` mirroring the backend. The Add Creature form removes its manual initiative input entirely; the modifier is read from the monster search result and passed transparently in the WS message.

### Acceptance Criteria:
- **AC 1:** In the pre-combat staging view, entities with `initiative === null` display `--` in place of an initiative value.
- **AC 2:** When `initiative_roll` is non-nil, the DM panel shows a breakdown tooltip on the initiative value: `16 (d20: 13 + mod: +3)`.
- **AC 3:** The DM can still manually override the final initiative at any time using the existing `dm_update_entity` controls (no change to that path).
- **AC 4:** Every auto-roll (at StartCombat or mid-combat add) triggers a WebSocket broadcast so all clients see the re-sorted initiative ladder immediately.
- **AC 5:** The Add Creature form has no initiative input. When a monster is found via search, its `initiative_modifier` is passed in the `add_creature` WS message. When no monster is found (manual name entry), `initiative_modifier` is omitted (nil).
- **AC 6:** Frontend `Entity` type: `initiative_modifier: number | null`, `initiative_roll: number | null`.
