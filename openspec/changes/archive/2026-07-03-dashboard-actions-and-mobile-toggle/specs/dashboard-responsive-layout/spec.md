## ADDED Requirements

### Requirement: Viewport-Width-Based Responsive Tiers
The Dashboard SHALL determine its layout tier (phone, tablet, or desktop) purely from viewport width, using the same tiering as `dmview-responsive-layout`. It SHALL NOT use `navigator.userAgent`, touch-capability detection, or any other device/input-type signal to select the tier.

#### Scenario: Narrow viewport reaches the phone tier
- **WHEN** the Dashboard renders at a viewport width below 768px
- **THEN** the Dashboard SHALL render the phone tier layout

#### Scenario: Wide viewport reaches tablet or desktop tier
- **WHEN** the Dashboard renders at a viewport width of 768px or above
- **THEN** the Dashboard SHALL render the existing side-by-side column layout, unchanged from before this change

### Requirement: Phone Tier Renders a DM/Player Toggle
At the phone tier, the Dashboard SHALL render a two-option toggle control ("As DM" / "As Player") above the panel content, and SHALL render only the panel corresponding to the currently selected option. It SHALL NOT render both the "As DM" and "As Player" panels stacked simultaneously at this tier.

#### Scenario: Only one panel renders at a time on phone
- **WHEN** the Dashboard renders at the phone tier
- **THEN** exactly one of the "As DM" or "As Player" panels SHALL be present in the rendered output, selected by the toggle

#### Scenario: Toggling switches the visible panel
- **WHEN** a user on the phone tier activates the "As Player" option while "As DM" is showing
- **THEN** the Dashboard SHALL replace the visible panel with the "As Player" panel

#### Scenario: Toggle control remains visible regardless of selection
- **WHEN** the Dashboard renders at the phone tier with either option selected
- **THEN** the toggle control SHALL remain visible so the user can switch again

### Requirement: Phone Tier Defaults to the DM Panel
On first render at the phone tier, with no prior in-session selection, the Dashboard SHALL show the "As DM" panel by default.

#### Scenario: Fresh phone-tier load shows DM panel
- **WHEN** the Dashboard first renders at the phone tier in a new session
- **THEN** the "As DM" panel SHALL be the one visible

### Requirement: Tablet and Desktop Tiers Unaffected
At the tablet and desktop tiers, the Dashboard SHALL render the "As DM" and "As Player" panels side by side, as it did before this change, with no toggle control present.

#### Scenario: No toggle control above phone tier
- **WHEN** the Dashboard renders at the tablet or desktop tier
- **THEN** the DM/Player toggle control SHALL NOT be present, and both panels SHALL render side by side
