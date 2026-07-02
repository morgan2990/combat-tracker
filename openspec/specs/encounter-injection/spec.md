# Spec: Encounter Injection

## Capability

encounter-injection

## Purpose

Provides the in-room WebSocket pipeline that lets a DM select one of their saved encounter blueprints and spawn all of its monster groups into a live room in a single operation — resolving each group's official or custom monster reference, batching entity creation under one room-lock acquisition, and broadcasting exactly one updated `RoomState` for the whole injection.

## Requirements

### Requirement: DM Panel Encounter Templates Dropdown

At the phone tier (viewport width below 768px), the DM Panel SHALL render an "Encounter Templates" dropdown control that fetches `GET /api/encounters?edition=<room's edition>` on open and lists the DM's saved encounters matching the room's edition. At the tablet and desktop tiers (768px and above), the same encounters list SHALL instead render persistently in the DM Nav column, fetched when DMView mounts rather than on a toggle click. In both presentations, selecting an encounter SHALL send an `inject_encounter` WS message with the encounter's `id`.

#### Scenario: Dropdown lists only matching-edition encounters (phone tier)
- **WHEN** the DM Panel opens the Encounter Templates dropdown at a phone-tier viewport width in a `"5e"` room, and the DM owns encounters in both `"5e"` and `"5.5e"`
- **THEN** only the `"5e"` encounters appear in the dropdown list

#### Scenario: DM Nav column lists only matching-edition encounters (tablet/desktop tier)
- **WHEN** DMView renders at a tablet or desktop tier viewport width in a `"5e"` room, and the DM owns encounters in both `"5e"` and `"5.5e"`
- **THEN** the DM Nav column's encounters list shows only the `"5e"` encounters, visible without any click to open it

#### Scenario: Selecting an encounter dispatches injection
- **WHEN** the DM selects an encounter, whether from the phone-tier dropdown or the tablet/desktop DM Nav column list
- **THEN** the frontend SHALL send `{ "type": "inject_encounter", "encounter_id": "<id>" }` over the room's WebSocket connection

### Requirement: inject_encounter WS Message and Resolution

The server SHALL accept a DM-only WS message of type `inject_encounter` with an `encounter_id` field. On receipt, the server SHALL:
1. Fetch the encounter document, verifying it is owned by the requesting DM.
2. For each `EncounterMonster` group, resolve full monster stats: if `is_custom` is `false`, via `GetMonsterByName` using the group's `name`; if `is_custom` is `true`, via `GetCustomMonsterByID` using the group's `monster_id`.
3. Skip any group whose reference fails to resolve (e.g. a deleted custom monster), continuing with the remaining groups.
4. Inject the remaining resolved groups into the room in a single operation (see the `Encounter Injection Room Method` requirement) and broadcast the updated `RoomState` exactly once for the whole batch.

#### Scenario: DM injects an encounter with all monsters resolvable
- **WHEN** a DM sends `inject_encounter` for an encounter with 2 monster groups, both still resolvable
- **THEN** the server spawns both groups' worth of creatures and broadcasts one updated `RoomState`

#### Scenario: One group's monster reference no longer resolves
- **GIVEN** an encounter has a group referencing a custom monster that was since deleted
- **WHEN** the DM injects that encounter
- **THEN** the server SHALL skip that group, inject every other group normally, and broadcast the updated `RoomState` reflecting only the successfully resolved groups

#### Scenario: Non-owner cannot inject another DM's encounter
- **WHEN** a WS client sends `inject_encounter` referencing an encounter not owned by that connection's authenticated user
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Non-DM cannot inject an encounter
- **WHEN** a player-role client sends `inject_encounter`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: Encounter Injection Room Method

`room.Room` SHALL expose a method that accepts pre-resolved monster group data (name, max HP, initiative modifier, source metadata, quantity, display name per group) and, under a single mutex acquisition, performs the same per-instance entity-creation logic `AddCreature` already performs for each group — batch quantity expansion with auto-numbered names/aliases, and per-instance auto-rolled initiative when `IsStarted` is true — followed by a single `sortEntities` call. The room's lock SHALL be held for the whole batch, not re-acquired per group.

#### Scenario: Batch injection produces the same entities as repeated manual adds
- **GIVEN** an encounter with groups `{Goblin, qty 3}` and `{Orc, qty 1}`
- **WHEN** the encounter is injected
- **THEN** the resulting entities are indistinguishable from what four sequential `add_creature` calls (3 Goblins, 1 Orc) would have produced, including auto-numbered names and per-instance rolls

#### Scenario: Injected creatures respect pre-combat masking like any other creature
- **WHEN** an encounter is injected while `is_started` is `false`
- **THEN** the newly injected creatures are hidden from player-role clients by the existing pre-combat creature-masking rule, exactly as a manually-added creature would be

#### Scenario: Injected creatures roll initiative like any other creature
- **WHEN** an encounter is injected while `is_started` is `true` and a group's resolved monster has a non-nil `initiative_modifier`
- **THEN** each instance in that group receives its own independent auto-rolled `initiative`, exactly as `AddCreature` already does mid-combat
