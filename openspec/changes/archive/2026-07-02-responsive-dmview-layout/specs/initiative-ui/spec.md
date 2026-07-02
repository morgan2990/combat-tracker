## MODIFIED Requirements

### Requirement: Add Creature Form — My Creatures Quick-Pick

At the phone tier (viewport width below 768px), the Add Creature form in `DMView.tsx` SHALL render a "My Creatures" section inline, listing the DM's own custom monsters for the room's edition (fetched via `GET /api/custom-monsters?edition=<room's edition>`), alongside — not instead of — the existing monster search input. At the tablet and desktop tiers (768px and above), the same "My Creatures" list SHALL instead render persistently in the DM Nav column rather than inline in the form. In both presentations, selecting an entry SHALL populate the Add Creature form's `name`, `max_hp`, and statblock-reference state directly from the already-fetched document, without an additional network request, and the existing monster search input remains available in the form at every tier.

#### Scenario: DM sees their own custom monsters for the room's edition (phone tier)
- **WHEN** the DM Panel renders at a phone-tier viewport width in a `"5e"` room, and the DM owns 2 custom monsters in `"5e"` and 1 in `"5.5e"`
- **THEN** the inline "My Creatures" section in the Add Creature form lists only the 2 `"5e"` monsters

#### Scenario: DM Nav column lists custom monsters (tablet/desktop tier)
- **WHEN** DMView renders at a tablet or desktop tier viewport width in a `"5e"` room, and the DM owns 2 custom monsters in `"5e"` and 1 in `"5.5e"`
- **THEN** the DM Nav column's "My Creatures" list shows only the 2 `"5e"` monsters

#### Scenario: Selecting a quick-pick monster populates the form without a follow-up fetch
- **WHEN** the DM clicks one of their custom monsters, whether in the phone-tier inline section or the tablet/desktop DM Nav column
- **THEN** the Add Creature form's name, max HP, and statblock-reference fields (`source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`) are populated directly from the list response, with no additional `GET /api/custom-monsters/:id` request

#### Scenario: Search remains available alongside quick-pick
- **WHEN** the DM Panel renders with a non-empty "My Creatures" list, at any tier
- **THEN** the existing monster search input and its debounced dropdown in the Add Creature form continue to function exactly as before, unaffected by the quick-pick list's placement
