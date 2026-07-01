## Why

DMs have no way to represent a creature that's temporarily unseen by players — an ambusher waiting in the shadows, a monster that turned invisible mid-fight — without either removing it from the tracker entirely (losing HP/initiative/condition state) or leaving it visible and spoiling the surprise. Epic 17 adds a per-creature `is_hidden` toggle the DM controls live, filtered out of the Player View but always visible to the DM.

## What Changes

- Add an optional `is_hidden` boolean to the entity model (Go struct, Mongo persistence, WS payload, frontend type), defaulting to `false`.
- Add a `toggle_entity_visibility` WS message (DM-only, mirrors the existing `remove_entity` message shape: just an `entity_id`) that flips a single entity's `is_hidden` value and broadcasts the updated state. This is a dirty-marking mutation, not one of the five events that trigger an immediate Mongo write (per `room-persistence`).
- DM Panel: a 👁/🙈 toggle button on every creature row (matching the app's existing plain-emoji icon convention — 💀, 😵, 📋, 🗑), always rendering hidden creatures with a distinct 50%-opacity style. The DM always sees every entity regardless of `is_hidden`.
- Player View: fold `!is_hidden` into the initiative-ladder filter already introduced by Epic 15 (`epic15-monster-fog-of-war`), so a hidden creature is completely absent from the rendered list — no name, HP, or turn-order slot. Reveal is instant on the next broadcast, no transition, consistent with the Epic 15 decision.
- **Scope decision**: `is_hidden` only applies to creature-type entities in this change — the DM Panel toggle only appears on creature rows, matching how the epic's UI is actually scoped (US17.2 AC1 says "next to every creature"). PCs/companions can never be hidden; a "player turns invisible" use case is out of scope here and would be a future change.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `entity-schema`: adds `IsHidden`/`is_hidden` to the `Entity` struct and a new `toggleEntityVisibilityMsg` WS message, following the same optional-field precedent as `InitiativeModifier` and `DisplayName`.
- `room-persistence`: the enumerated persisted-entity field list gains `is_hidden`; snapshot/restore round-trip it.
- `room-state`: extends "Frontend is responsible for role-based data presentation" with scenarios for DM always-visible/distinct-styling and player-side hidden-entity omission, and composes with the existing pre-combat creature-masking scenarios from Epic 15.

## Impact

- `room/room.go`: `Entity` struct, new `ToggleEntityVisibility` method, `snapshot`/`inflateRoom`.
- `store/room.go`: `RoomEntitySnapshot` struct.
- `ws/handler.go`: new `toggleEntityVisibilityMsg` struct and `toggle_entity_visibility` dispatch case.
- `frontend/src/types.ts`: `Entity.is_hidden`.
- `frontend/src/components/DMView.tsx`: eye-icon toggle button, hidden-row styling.
- `frontend/src/components/PlayerView.tsx`: extend the existing `visibleEntities` filter with `!e.is_hidden`.
- No changes to companion or PC entities, and no changes to the `dm_update_entity` message (visibility is its own dedicated toggle, not folded into the general entity-update message).
