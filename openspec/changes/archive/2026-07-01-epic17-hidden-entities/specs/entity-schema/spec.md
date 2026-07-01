## ADDED Requirements

### Requirement: Entity Struct Hidden Flag

The `room.Entity` struct in `room/room.go` SHALL carry a hidden-visibility flag:

```go
IsHidden bool `json:"is_hidden"`
```

- `IsHidden` marks a creature as currently unseen by players (an ambusher, an invisible monster), independent of any other state (dead, unconscious, initiative).
- The zero value is `false`; no explicit initialization is required when an entity is created.
- Unlike `DisplayName`, the JSON tag has no `omitempty` — a boolean masking flag is always serialized explicitly, matching how `Dead` is already handled.
- `IsHidden` is only ever set to `true` via `toggle_entity_visibility` on creature-type entities; PCs and companions never have `IsHidden: true` in this scope, since no DM Panel control targets those types.

#### Scenario: Entity defaults to visible
- **WHEN** any entity is added to a room (via `add_creature`, `setup_character`, or `add_companion`)
- **THEN** `Entity.IsHidden` is `false`

#### Scenario: DM hides a creature
- **GIVEN** a creature entity with `IsHidden: false`
- **WHEN** the DM sends `toggle_entity_visibility` for that entity's ID
- **THEN** `Entity.IsHidden` becomes `true`

#### Scenario: DM reveals a hidden creature
- **GIVEN** a creature entity with `IsHidden: true`
- **WHEN** the DM sends `toggle_entity_visibility` for that entity's ID again
- **THEN** `Entity.IsHidden` becomes `false`

### Requirement: toggleEntityVisibilityMsg WS Message

`ws/handler.go` SHALL define a `toggleEntityVisibilityMsg` struct, dispatched on message type `toggle_entity_visibility`:

```go
type toggleEntityVisibilityMsg struct {
    EntityID string `json:"entity_id"`
}
```

This mirrors the existing `removeEntityMsg` shape — the action is addressed by entity ID alone, with no other fields.

#### Scenario: Valid toggle message
- **WHEN** a WS message of type `toggle_entity_visibility` includes `"entity_id": "abc123"`
- **THEN** the server flips `IsHidden` on the entity with ID `"abc123"` and broadcasts the updated `RoomState`

#### Scenario: Toggle targets a nonexistent entity
- **WHEN** a WS `toggle_entity_visibility` message references an entity ID not present in the room
- **THEN** the server SHALL NOT broadcast a state change and SHALL NOT error the connection

### Requirement: ToggleEntityVisibility Room Method

`room.Room` SHALL expose a `ToggleEntityVisibility(sessionID, entityID string) error` method that flips the target entity's `IsHidden` value, following the same ownership-check and error-handling pattern as `RemoveEntity`.

#### Scenario: Non-owner cannot toggle visibility
- **WHEN** `ToggleEntityVisibility` is called with a `sessionID` that is not the room's DM
- **THEN** the method returns an error and no entity is modified

#### Scenario: Toggling visibility does not trigger an immediate persist
- **WHEN** `ToggleEntityVisibility` succeeds
- **THEN** the room is marked dirty for the next periodic sweep; it is not one of the events that trigger an immediate MongoDB write

### Requirement: Frontend Entity Type Hidden Field

The `Entity` interface in `types.ts` SHALL include a new field:

```typescript
is_hidden: boolean
```

Unlike `display_name`, this field is not optional — the server always serializes it explicitly (no `omitempty`), so frontend consumers can rely on it always being present and boolean.

#### Scenario: Frontend reads the hidden flag
- **WHEN** a `RoomState` payload includes an entity with `"is_hidden": true`
- **THEN** frontend consumers SHALL treat that entity as currently hidden from players
