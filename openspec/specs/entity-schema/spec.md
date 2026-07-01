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

### Requirement: Entity Struct Display Name Field

The `room.Entity` struct in `room/room.go` SHALL carry an optional display name field:

```go
DisplayName string `json:"display_name,omitempty"`
```

- `DisplayName` is a narrative alias for a creature instance, distinct from the base `Name` (which continues to reference the source statblock/template).
- Empty string means no alias was set; `PCID`, companions, and PC entities never have a non-empty `DisplayName` since only `add_creature` populates it.
- Unlike `InitiativeModifier`/`InitiativeRoll`, `DisplayName` is a plain `string`, not a pointer — there is no meaningful distinction between "unset" and "empty" for this field.

#### Scenario: Creature added with a custom alias
- **GIVEN** a creature is added via `add_creature` with a non-empty custom name field
- **WHEN** the entity is stored in room state
- **THEN** `Entity.DisplayName` is set to the provided string

#### Scenario: Creature added without a custom alias
- **GIVEN** a creature is added via `add_creature` with the custom name field blank or omitted
- **WHEN** the entity is stored in room state
- **THEN** `Entity.DisplayName` is the empty string

### Requirement: addCreatureMsg Display Name Field

The `addCreatureMsg` struct in `ws/handler.go` SHALL include a `display_name` field:

```go
type addCreatureMsg struct {
    Name               string `json:"name"`
    MaxHP              int    `json:"max_hp"`
    InitiativeModifier *int   `json:"initiative_modifier"`
    Quantity           int    `json:"quantity"`
    SourceType         string `json:"source_type"`
    ReferenceURL       string `json:"reference_url"`
    PDFObjectKey       string `json:"pdf_object_key"`
    DisplayName        string `json:"display_name"`
}
```

A missing or blank `display_name` field decodes as the empty string (no alias).

#### Scenario: Message with a display name
- **WHEN** a WS `add_creature` message includes `"display_name": "Guard Alpha"`
- **THEN** `addCreatureMsg.DisplayName` is `"Guard Alpha"`

#### Scenario: Message without a display name
- **WHEN** a WS `add_creature` message omits `display_name` or sends an empty string
- **THEN** `addCreatureMsg.DisplayName` is the empty string

### Requirement: Batch-Added Creatures Get Auto-Numbered Aliases

When `add_creature` specifies `quantity > 1` and a non-empty `display_name`, the server SHALL auto-number the alias per instance using the same `"{base} {n}"` scheme already used for the base `name` field, rather than assigning every instance the identical alias string.

#### Scenario: Batch with an alias
- **GIVEN** a DM adds 3 creatures in one `add_creature` call with `name: "Goblin"` and `display_name: "Guard"`
- **WHEN** the entities are created
- **THEN** the resulting entities have `Name` values `"Goblin 1"`, `"Goblin 2"`, `"Goblin 3"` and `DisplayName` values `"Guard 1"`, `"Guard 2"`, `"Guard 3"` respectively

#### Scenario: Batch without an alias
- **GIVEN** a DM adds multiple creatures in one `add_creature` call with no `display_name`
- **WHEN** the entities are created
- **THEN** every resulting entity has `DisplayName` equal to the empty string (not numbered)

### Requirement: dmUpdateEntityMsg Display Name Field

The `dmUpdateEntityMsg` struct in `ws/handler.go` SHALL include a `display_name` field, and `DMUpdateEntity` SHALL apply it unconditionally to creature-type entities (unlike `Name`, an empty `display_name` is a meaningful, intentional value — it clears the alias).

#### Scenario: DM sets an alias after creation
- **GIVEN** a creature entity currently has an empty `DisplayName`
- **WHEN** the DM sends `dm_update_entity` with a non-empty `display_name`
- **THEN** the entity's `DisplayName` is updated to that value

#### Scenario: DM clears an existing alias
- **GIVEN** a creature entity currently has a non-empty `DisplayName`
- **WHEN** the DM sends `dm_update_entity` with `display_name` set to the empty string
- **THEN** the entity's `DisplayName` is cleared to the empty string

#### Scenario: display_name update does not affect non-creature entities
- **WHEN** `dm_update_entity` targets an entity with `type` other than `"creature"`
- **THEN** the entity's `DisplayName` field is not modified (PCs and companions never carry a `DisplayName`)

### Requirement: Frontend Entity Type Display Name Field

The `Entity` interface in `types.ts` SHALL include a new optional field:

```typescript
display_name?: string
```

The field MAY be absent or empty in payloads; consumers MUST treat both as "no alias, fall back to `name`."

#### Scenario: Field absent from an older payload
- **WHEN** a `RoomState` payload omits `display_name` for an entity (e.g. a PC or companion)
- **THEN** frontend consumers SHALL treat the entity as having no alias and render `name`
