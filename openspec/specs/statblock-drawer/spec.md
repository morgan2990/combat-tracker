# Spec: Statblock Drawer

## Purpose

Defines the DM-facing UI for viewing monster stat blocks inline within the combat tracker. When a creature entity carries a statblock reference (`source_type`, `reference_url`, or `pdf_object_key`), the DM can open a slide-out drawer anchored to that entity's row to inspect the stat block without leaving the tracker. Player and companion entities are never eligible for this feature.

## Requirements

### Requirement: DM tracker shows a statblock icon for creatures with a reference
The DM panel SHALL render a statblock icon button next to the name of any creature entity whose `source_type` is non-empty. Entities with no `source_type` SHALL NOT show the icon. The icon is only rendered in `DMView`; `PlayerView` is unaffected.

#### Scenario: Statblock icon shown for creature with URL reference
- **WHEN** a creature entity in the DM tracker has `source_type: "url"`
- **THEN** a statblock icon button SHALL appear next to that entity's name in the tracker row

#### Scenario: Statblock icon shown for creature with PDF reference
- **WHEN** a creature entity in the DM tracker has `source_type: "pdf"`
- **THEN** a statblock icon button SHALL appear next to that entity's name in the tracker row

#### Scenario: No icon for creatures without a reference
- **WHEN** a creature entity has `source_type` absent or empty
- **THEN** no statblock icon SHALL be rendered for that entity

#### Scenario: Player and companion entities never show the icon
- **WHEN** any entity with `type: "player"` or `type: "companion"` is in the tracker
- **THEN** no statblock icon SHALL be rendered regardless of whether `source_type` is set

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

### Requirement: Drawer renders a URL-type statblock as an image
When the open creature has `source_type: "url"`, the drawer SHALL render the statblock as an `<img>` element pointing to `reference_url`.

#### Scenario: URL statblock image renders
- **WHEN** the drawer opens for a creature with `source_type: "url"` and a valid `reference_url`
- **THEN** the drawer SHALL display `<img src={reference_url}>` scaled to fit the drawer width

### Requirement: Drawer renders a PDF-type statblock as an embedded PDF
When the open creature has `source_type: "pdf"`, the drawer SHALL render an `<embed>` element whose `src` points to `GET /api/monsters/:name/pdf`.

#### Scenario: PDF statblock renders
- **WHEN** the drawer opens for a creature with `source_type: "pdf"`
- **THEN** the drawer SHALL display `<embed src="/api/monsters/{name}/pdf" type="application/pdf">` sized to fill the drawer

### Requirement: Drawer content is lazy-loaded
The drawer content (image or PDF embed) SHALL NOT be fetched or rendered until the DM first opens the drawer for that creature.

#### Scenario: Statblock not fetched before drawer opens
- **WHEN** a creature with a statblock reference is present in the tracker and the DM has not clicked the icon
- **THEN** no network request for the statblock resource SHALL be made by the client
