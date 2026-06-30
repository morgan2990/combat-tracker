# Spec: Auto-Roll Engine

## Capability
`auto-roll-engine`

## Purpose
Provides cryptographically random d20 rolling for creature initiative inside the room package, integrating with `AddCreature` and `StartCombat` so monsters with a known modifier receive an automatic initiative total without DM intervention.

## Requirements

### Requirement: rollD20 Helper

The `room` package SHALL expose a private `rollD20` function that returns a uniform integer in [1, 20] using `crypto/rand`:

```go
func rollD20() int {
    n, _ := rand.Int(rand.Reader, big.NewInt(20))
    return int(n.Int64()) + 1
}
```

#### Scenario: Distribution
- **WHEN** `rollD20` is called
- **THEN** the result is always between 1 and 20 inclusive

### Requirement: AddCreature Signature and Auto-Roll Logic

`AddCreature` SHALL accept `initiativeModifier *int` in place of the former `initiative int` parameter:

```
Before: AddCreature(sessionID, name string, maxHP, initiative, quantity int, ...) error
After:  AddCreature(sessionID, name string, maxHP int, initiativeModifier *int, quantity int, ...) error
```

Per-creature logic in the batch loop:

```
var init *int
var roll *int
if r.State.IsStarted && initiativeModifier != nil {
    d := rollD20()
    total := d + *initiativeModifier
    roll = &d
    init = &total
}
entity := Entity{
    ...
    Initiative:         init,
    InitiativeModifier: initiativeModifier,
    InitiativeRoll:     roll,
}
```

#### Scenario: Mid-combat creature with modifier
- **GIVEN** combat has started (`IsStarted == true`)
- **WHEN** a creature is added with a non-nil `initiativeModifier`
- **THEN** the entity is created with `Initiative` set to `rollD20() + modifier`, `InitiativeRoll` set to the raw die face, and `InitiativeModifier` set to the modifier

#### Scenario: Mid-combat creature without modifier
- **GIVEN** combat has started
- **WHEN** a creature is added with nil `initiativeModifier`
- **THEN** `Initiative`, `InitiativeRoll`, and `InitiativeModifier` are all nil on the new entity

#### Scenario: Pre-combat creature with modifier
- **GIVEN** combat has NOT started (`IsStarted == false`)
- **WHEN** a creature is added with a non-nil `initiativeModifier`
- **THEN** `InitiativeModifier` is stored but `Initiative` and `InitiativeRoll` remain nil (rolled at StartCombat)

#### Scenario: Batch quantity
- **WHEN** a creature with a modifier is added with `quantity > 1` mid-combat
- **THEN** each individual entity in the batch receives its own independent `rollD20()` call

### Requirement: StartCombat Auto-Roll Pass

`StartCombat` SHALL include a pre-sort pass that auto-rolls initiative for any creature with a non-nil `InitiativeModifier` and a nil `Initiative`:

```
for i := range r.State.Entities {
    e := &r.State.Entities[i]
    if e.Type == "creature" && e.InitiativeModifier != nil && e.Initiative == nil {
        d := rollD20()
        total := d + *e.InitiativeModifier
        e.InitiativeRoll = &d
        e.Initiative = &total
    }
}
r.sortEntities()
r.State.IsStarted = true
r.State.Round = 1
r.State.ActiveIndex = 0
```

#### Scenario: Pre-staged creature with modifier
- **GIVEN** one or more creatures with a non-nil `InitiativeModifier` and nil `Initiative` are in the roster
- **WHEN** `StartCombat` is called
- **THEN** each such creature receives an auto-rolled `Initiative` and `InitiativeRoll` before entities are sorted

#### Scenario: Manually set initiative preserved
- **GIVEN** a creature already has a non-nil `Initiative` (set manually by the DM)
- **WHEN** `StartCombat` is called
- **THEN** that creature's `Initiative` is NOT changed by the auto-roll pass

### Requirement: Auto-Roll Invariants

The system SHALL maintain the following invariants:

- A creature with nil `Initiative` after `StartCombat` means nil `InitiativeModifier` — the DM must set initiative manually.
- `InitiativeRoll` is non-nil if and only if the initiative was auto-rolled (not manually set).
- The `dm_update_entity` handler MUST NOT modify `InitiativeRoll` — if the DM overrides an initiative value, the raw roll is preserved for tooltip reference.
