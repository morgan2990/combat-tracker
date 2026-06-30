# Spec: Entity Schema — Initiative Fields (delta)

## Capability
`entity-schema`

## Type
Delta — extends existing Entity struct and addCreatureMsg

## Backend: room.Entity

Two new fields added to `room.Entity` in `room/room.go`:

```go
InitiativeModifier *int `json:"initiative_modifier,omitempty"`
InitiativeRoll     *int `json:"initiative_roll,omitempty"`
```

- `InitiativeModifier`: the d20 modifier, captured from the `add_creature` WS message when the entity is created. Nil for players, companions, and creatures where no modifier was provided.
- `InitiativeRoll`: the raw d20 face (1–20), set when the auto-roll fires. Nil until a roll occurs. Stays nil if initiative was set manually.

Both are omitted from JSON when nil (`omitempty`).

## Backend: addCreatureMsg

In `ws/handler.go`, the `addCreatureMsg` struct changes:

Remove: `Initiative int`
Add: `InitiativeModifier *int`

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

A missing or JSON-null `initiative_modifier` field decodes as nil.

## Backend: store.Monster

`Monster.InitiativeModifier` promoted from `int` to `*int`:

```go
InitiativeModifier *int `bson:"initiative_modifier,omitempty" json:"initiative_modifier,omitempty"`
```

Nil means the modifier was never set (custom monster with blank form). Non-nil (including 0) means the modifier is known.

## Frontend: types.ts

Two new fields on the `Entity` interface:

```typescript
initiative_modifier: number | null
initiative_roll: number | null
```

Both can be absent in older payloads; treat missing as null.
