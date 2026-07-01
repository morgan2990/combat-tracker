# Spec: Entity Schema — Initiative Fields

## Capability
`entity-schema`

## Purpose
Extends the shared `room.Entity` struct and the `addCreatureMsg` WebSocket message to carry per-entity initiative modifier and raw d20 roll values, enabling automatic initiative rolling and breakdown display.

## Requirements

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

### Requirement: addCreatureMsg Struct Update

The `addCreatureMsg` struct in `ws/handler.go` SHALL replace the existing `Initiative int` field with `InitiativeModifier *int`:

```go
type addCreatureMsg struct {
    Name               string `json:"name"`
    MaxHP              int    `json:"max_hp"`
    InitiativeModifier *int   `json:"initiative_modifier"`
    Quantity           int    `json:"quantity"`
    SourceType         string `json:"source_type"`
    ReferenceURL       string `json:"reference_url"`
    PDFObjectKey       string `json:"pdf_object_key"`
}
```

A missing or JSON-null `initiative_modifier` field MUST decode as nil.

#### Scenario: Message with modifier
- **WHEN** a WS `add_creature` message includes `"initiative_modifier": 3`
- **THEN** `addCreatureMsg.InitiativeModifier` is a non-nil pointer to 3

#### Scenario: Message without modifier
- **WHEN** a WS `add_creature` message omits `initiative_modifier` or sends null
- **THEN** `addCreatureMsg.InitiativeModifier` is nil

### Requirement: Monster Store Pointer Promotion

`store.Monster.InitiativeModifier` SHALL be promoted from `int` to `*int`:

```go
InitiativeModifier *int `bson:"initiative_modifier,omitempty" json:"initiative_modifier,omitempty"`
```

- Nil means the modifier was never set (custom monster with blank form).
- Non-nil (including 0) means the modifier is known.

### Requirement: Frontend Entity Type Fields

The `Entity` interface in `types.ts` SHALL include two new nullable fields:

```typescript
initiative_modifier: number | null
initiative_roll: number | null
```

Both fields MAY be absent in older payloads; consumers MUST treat a missing field as null.
