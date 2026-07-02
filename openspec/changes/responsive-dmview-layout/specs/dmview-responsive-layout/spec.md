## ADDED Requirements

### Requirement: Viewport-Width-Based Responsive Tiers
DMView SHALL determine its layout tier (phone, tablet, or desktop) purely from viewport width. It SHALL NOT use `navigator.userAgent`, touch-capability detection, or any other device/input-type signal to select the tier.

#### Scenario: Wide tablet reaches the desktop tier
- **WHEN** DMView renders in a browser reporting a 1920px-wide viewport, regardless of the underlying device being a tablet
- **THEN** DMView SHALL render the desktop tier layout

#### Scenario: Narrowed desktop browser window reaches the phone tier
- **WHEN** a desktop browser window is resized to a viewport narrower than 768px
- **THEN** DMView SHALL render the phone tier layout

### Requirement: Phone Tier Layout Unchanged
At viewport widths below 768px, DMView SHALL render the existing single-column layout: header, combat controls, Add Creature form with inline "My Creatures" quick-pick, Encounter Templates dropdown, and initiative tracker stacked vertically, with the statblock panel as a fixed-position overlay drawer. The whole page SHALL scroll as a single unit, as it does today.

#### Scenario: Phone tier renders single column
- **WHEN** DMView renders at a viewport width below 768px
- **THEN** it SHALL render one column containing all DM Panel content, with no persistent DM Nav or Statblock column

### Requirement: Tablet Tier Two-Column Layout
At viewport widths from 768px up to the desktop threshold (~1320px), DMView SHALL render two columns: a DM Nav column on the left and the Tracker column (header, combat controls, Add Creature form, initiative tracker) on the right. The statblock panel SHALL remain a fixed-position overlay drawer, not a grid column, at this tier.

#### Scenario: Tablet tier renders two columns
- **WHEN** DMView renders at a viewport width between 768px and the desktop threshold
- **THEN** it SHALL render the DM Nav column and the Tracker column side by side, with no Statblock grid column

### Requirement: Desktop Tier Three-Column Layout
At viewport widths at or above ~1320px, DMView SHALL render three columns: DM Nav, Tracker, and Statblock, arranged left to right.

#### Scenario: Desktop tier renders three columns
- **WHEN** DMView renders at a viewport width at or above the desktop threshold
- **THEN** it SHALL render the DM Nav, Tracker, and Statblock columns side by side

### Requirement: DM Nav Column Composition
At the tablet and desktop tiers, the DM Nav column SHALL contain the DM's saved encounters list and custom monsters list, persistently visible without requiring a click to open (see the `encounter-injection` and `initiative-ui` capabilities for the lists' fetch and click-selection behavior).

#### Scenario: DM Nav column shows both lists without interaction
- **WHEN** DMView renders at a tablet or desktop tier width
- **THEN** the DM Nav column SHALL display the encounters list and the custom monsters list without the DM needing to click to reveal either

### Requirement: Tracker Column Max Width
The Tracker column's maximum width SHALL be approximately 580px (reduced from the phone tier's previous 720px single-column cap), so that the combined DM Nav, Tracker, and Statblock column widths reach the desktop threshold on viewports around 1366px wide.

#### Scenario: Tracker column capped below 720px at tablet/desktop tiers
- **WHEN** DMView renders at a tablet or desktop tier width
- **THEN** the Tracker column's rendered width SHALL NOT exceed approximately 580px

### Requirement: Statblock Column Empty-State Placeholder
At the desktop tier, when no creature's statblock is currently open, the Statblock column SHALL display a placeholder image rather than being empty or collapsing to zero width.

#### Scenario: No statblock open shows placeholder
- **WHEN** DMView renders at a desktop tier width and no creature's statblock is open
- **THEN** the Statblock column SHALL display a placeholder image

#### Scenario: Opening a statblock replaces the placeholder
- **WHEN** the DM opens a creature's statblock at a desktop tier width
- **THEN** the Statblock column SHALL display that creature's statblock content in place of the placeholder image

### Requirement: Independent Per-Column Scroll
At the tablet and desktop tiers, each rendered column (DM Nav, Tracker, and, at desktop, Statblock) SHALL scroll independently within its own vertical axis, rather than the whole page scrolling as a single unit.

#### Scenario: Long tracker list does not affect DM Nav column position
- **WHEN** the initiative tracker in the Tracker column has enough entities to overflow its height at a tablet or desktop tier width, and the DM scrolls the Tracker column
- **THEN** the DM Nav column's scroll position SHALL be unaffected

### Requirement: Capped Total Width With Centering
At the tablet and desktop tiers, the overall multi-column layout SHALL have a capped total width and SHALL center within the viewport with gutters on wider screens, rather than growing the columns to fill all available horizontal space.

#### Scenario: Ultra-wide viewport shows gutters instead of stretched columns
- **WHEN** DMView renders at a viewport width well beyond the desktop threshold (e.g. 1920px)
- **THEN** the DM Nav, Tracker, and Statblock columns SHALL retain their capped widths and the layout SHALL be centered with empty gutter space on either side, rather than the columns growing wider
