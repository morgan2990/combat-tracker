## MODIFIED Requirements

### Requirement: DM can toggle a statblock panel for a creature

Clicking the statblock icon SHALL toggle the statblock panel for that entity. At the phone and tablet tiers (viewport width below ~1320px), the panel SHALL render as a slide-out drawer overlay anchored to the viewport's right edge, as today. At the desktop tier (~1320px and above), the panel SHALL render as the content of the persistent Statblock grid column instead of an overlay. At every tier, only one creature's statblock SHALL be open at a time; opening a new one replaces any previously open one, and closing it returns the panel/column to its empty state.

#### Scenario: DM opens the panel at phone/tablet tier
- **WHEN** the DM clicks the statblock icon for a creature with a reference at a phone or tablet tier viewport width
- **THEN** the drawer SHALL slide open over the viewport's right edge and display the statblock content

#### Scenario: DM opens the panel at desktop tier
- **WHEN** the DM clicks the statblock icon for a creature with a reference at a desktop tier viewport width
- **THEN** the Statblock column SHALL display that creature's statblock content, replacing its placeholder image

#### Scenario: DM closes the panel by clicking the icon again
- **WHEN** the panel is open for a creature and the DM clicks the same statblock icon, at any tier
- **THEN** the drawer SHALL collapse (phone/tablet tier) or the Statblock column SHALL return to its placeholder image (desktop tier)

#### Scenario: Opening a second creature's statblock replaces the first
- **WHEN** a statblock is open for creature A and the DM clicks the statblock icon for creature B, at any tier
- **THEN** creature A's statblock SHALL close/be replaced and creature B's statblock SHALL be shown
