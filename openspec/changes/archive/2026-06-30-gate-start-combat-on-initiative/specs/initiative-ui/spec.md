## ADDED Requirements

### Requirement: Start Combat Button Disabled Pending Initiative

The DM view SHALL disable the "Start Combat" button whenever at least one `player` or `companion` entity in the room has `initiative === null`, matching the server's `start_combat` validation rule. A disabled button SHALL NOT send the `start_combat` message when clicked.

The disabled state SHALL be visually indicated (reduced opacity and a `not-allowed` cursor), consistent with other disabled buttons in the app.

#### Scenario: Button disabled while a player has no initiative
- **GIVEN** at least one `player` or `companion` entity has `initiative === null`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is shown disabled and clicking it does not send `start_combat`

#### Scenario: Button enabled once all initiatives are set
- **GIVEN** every `player` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is enabled and clicking it sends `start_combat`

#### Scenario: Creature entities do not affect the gate
- **GIVEN** one or more `creature` entities have `initiative === null` but every `player` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** the Start Combat button is enabled

### Requirement: Pending Initiative Summary

When the Start Combat button is disabled due to missing initiative, the DM view SHALL display a summary line listing the names of the blocking `player` and `companion` entities, comma-separated. The line SHALL NOT render when no entity is blocking.

#### Scenario: Summary lists blocking entities by name
- **GIVEN** entities named "Bob" (player, `initiative === null`) and "Fido" (companion, `initiative === null`) are blocking
- **WHEN** the DM view renders the Combat controls row
- **THEN** a summary line is shown naming both "Bob" and "Fido"

#### Scenario: Summary hidden once unblocked
- **GIVEN** every `player` and `companion` entity has a non-null `initiative`
- **WHEN** the DM view renders the Combat controls row
- **THEN** no pending-initiative summary line is rendered

#### Scenario: Sharing companion resolved alongside its owner
- **GIVEN** a companion has `shares_initiative === true` and its owning player has just had `initiative` set (which the server auto-copies to the companion)
- **WHEN** the DM view re-renders with the updated `RoomState`
- **THEN** the companion is no longer included in the pending-initiative summary
