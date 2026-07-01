## MODIFIED Requirements

### Requirement: Entity Struct Initiative Fields

The `room.Entity` struct in `room/room.go` SHALL carry two new optional fields:

```go
InitiativeModifier *int `json:"initiative_modifier,omitempty"`
InitiativeRoll     *int `json:"initiative_roll,omitempty"`
```

- `InitiativeModifier`: the d20 modifier captured from the `add_creature` WS message when the entity is created. Nil for PCs, companions, and creatures where no modifier was provided.
- `InitiativeRoll`: the raw d20 face (1–20), set when the auto-roll fires. Nil until a roll occurs. Stays nil if initiative was set manually.

Both fields MUST be omitted from JSON serialisation when nil (`omitempty`).

#### Scenario: Creature added with modifier
- **GIVEN** a creature is added with a non-null `initiative_modifier`
- **WHEN** the entity is stored in room state
- **THEN** `InitiativeModifier` is set to the provided value and `InitiativeRoll` is nil (until combat starts or a roll fires)

#### Scenario: Creature added without modifier
- **GIVEN** a creature is added with no `initiative_modifier` in the WS message
- **WHEN** the entity is stored in room state
- **THEN** both `InitiativeModifier` and `InitiativeRoll` are nil

#### Scenario: Auto-roll fires
- **WHEN** the auto-roll engine sets a d20 result
- **THEN** `InitiativeRoll` is set to the raw die face and `Initiative` is set to the total
