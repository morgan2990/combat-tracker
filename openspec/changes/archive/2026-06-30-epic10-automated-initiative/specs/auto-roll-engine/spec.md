# Spec: Auto-Roll Engine (new capability)

## Capability
`auto-roll-engine`

## Type
New — backend initiative rolling logic in room/room.go

## rollD20 helper

```go
func rollD20() int {
    n, _ := rand.Int(rand.Reader, big.NewInt(20))
    return int(n.Int64()) + 1
}
```

Uses the existing `crypto/rand` import. Returns a uniform integer in [1, 20].

## AddCreature changes

Signature change:
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

- Mid-combat (IsStarted) + non-nil modifier → rolls immediately, initiative set.
- Pre-combat or nil modifier → initiative stays nil, modifier stored for later.
- Each creature in a batch (quantity > 1) gets its own independent `rollD20()` call.

## StartCombat changes

After the existing player/companion initiative guard, before setting `IsStarted = true`:

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

Creatures where the DM already set initiative manually (`Initiative != nil`) are skipped.

## Invariants

- A creature with nil `Initiative` after StartCombat means nil `InitiativeModifier` — DM must set it.
- `InitiativeRoll` is always non-nil if and only if the initiative was auto-rolled (not manually set).
- `dm_update_entity` does not touch `InitiativeRoll` — if the DM overrides the value, the raw roll stays for reference (tooltip still shows the original roll).
