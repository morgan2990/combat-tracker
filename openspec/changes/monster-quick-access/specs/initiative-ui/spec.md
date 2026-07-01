## ADDED Requirements

### Requirement: Add Creature Form — My Creatures Quick-Pick

The Add Creature form in `DMView.tsx` SHALL render a "My Creatures" section listing the DM's own custom monsters for the room's edition (fetched via `GET /api/custom-monsters?edition=<room's edition>`), alongside — not instead of — the existing monster search input. Selecting an entry from this section SHALL populate the form's `name`, `max_hp`, and statblock-reference state directly from the already-fetched document, without an additional network request.

#### Scenario: DM sees their own custom monsters for the room's edition
- **WHEN** the DM Panel renders in a `"5e"` room and the DM owns 2 custom monsters in `"5e"` and 1 in `"5.5e"`
- **THEN** the "My Creatures" section lists only the 2 `"5e"` monsters

#### Scenario: Selecting a quick-pick monster populates the form without a follow-up fetch
- **WHEN** the DM clicks one of their custom monsters in the "My Creatures" section
- **THEN** the form's name, max HP, and statblock-reference fields (`source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`) are populated directly from the list response, with no additional `GET /api/custom-monsters/:id` request

#### Scenario: Search remains available alongside quick-pick
- **WHEN** the DM Panel renders with a non-empty "My Creatures" section
- **THEN** the existing monster search input and its debounced dropdown continue to function exactly as before, unaffected by the new section
