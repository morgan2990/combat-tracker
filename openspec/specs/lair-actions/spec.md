# Spec: Lair Actions

## Capability

lair-actions

## Purpose

Defines the Lair Action entity type: a DM-injectable, fixed-initiative-20, HP-less hazard marker used to represent environmental or monster lair effects in the initiative order. Covers the WS message and room method used to add one, its initiative tie-break behavior relative to other entities, the DM Panel quick-inject control, and how lair action rows render (omitting HP/status UI) in both the DM Panel and Player View.

## Requirements

### Requirement: add_lair_action WS Message and AddLairAction Method

The server SHALL accept a DM-only WS message of type `add_lair_action` with no payload fields. `room.Room` SHALL expose `AddLairAction(sessionID string) error`, which appends a single entity to the room with:

- `name`: `"Lair Action"`
- `type`: `"lair_action"`
- `initiative`: `20`
- `max_hp`: `0`
- `current_hp`: `0`
- `conditions`: an empty slice (not nil)
- `is_hidden`: `true`

No other fields (`display_name`, `source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`, `initiative_roll`) are set; they retain their zero values.

#### Scenario: DM adds a lair action
- **WHEN** a DM-role client sends `{ "type": "add_lair_action" }`
- **THEN** the server appends an entity as specified above, sorts entities, and broadcasts the updated `RoomState`

#### Scenario: Non-DM cannot add a lair action
- **WHEN** a player-role client sends `add_lair_action`
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Multiple lair actions may coexist
- **WHEN** a DM sends `add_lair_action` more than once in the same room
- **THEN** each call appends an independent entity; the server SHALL NOT deduplicate or cap the count

### Requirement: Lair Action Initiative Tie-Break

When `sortEntities` compares two entities with equal, non-nil initiative values and exactly one of them has `type: "lair_action"`, that entity SHALL sort after the other, regardless of insertion order into `State.Entities`.

#### Scenario: Lair action loses a tie against a creature added earlier
- **GIVEN** a creature with `initiative: 20` was added before a lair action
- **WHEN** the lair action is added (also `initiative: 20`) and entities are sorted
- **THEN** the creature SHALL appear before the lair action in turn order

#### Scenario: Lair action loses a tie against a creature added later
- **GIVEN** a lair action with `initiative: 20` exists in the room
- **WHEN** a creature with `initiative: 20` is added afterward and entities are sorted
- **THEN** the creature SHALL appear before the lair action in turn order, even though it was added after

#### Scenario: Two lair actions at the same initiative preserve relative order
- **WHEN** two `lair_action` entities both have `initiative: 20` and are sorted
- **THEN** their relative order SHALL be unaffected by the tie-break rule (stable-sort insertion order applies between them, since neither "wins" against the other)

### Requirement: DM Panel Quick-Inject Button

The DM Panel's combat-controls bar SHALL render a `+ Add Lair Action` button, visible regardless of `is_started`, that sends `{ "type": "add_lair_action" }` when clicked.

#### Scenario: Button available before combat starts
- **WHEN** the DM Panel renders with `is_started: false`
- **THEN** the `+ Add Lair Action` button is visible and enabled

#### Scenario: Button available during combat
- **WHEN** the DM Panel renders with `is_started: true`
- **THEN** the `+ Add Lair Action` button is visible and enabled

### Requirement: Lair Action Row Rendering

In both the DM Panel and Player View initiative rows, an entity with `type: "lair_action"` SHALL render without: exact HP display, the HP delta/smart-input editor, dead/unconscious vital-state badges, condition toggles, and the Kill/Revive button. It SHALL still render with: the Remove button, the initiative value and editor, and (DM Panel only) the Name and Alias editors and the 👁/🙈 visibility toggle.

#### Scenario: No HP or status UI rendered
- **WHEN** either view renders an entity with `type: "lair_action"`
- **THEN** no HP number, HP bar, HP editor input, condition tag, or Kill/Revive control appears for that row

#### Scenario: Remove button still available
- **WHEN** the DM Panel renders a `lair_action` row
- **THEN** the Remove button is present and, when clicked, sends `remove_entity` for that entity's ID exactly as it does for any other entity type

#### Scenario: Initiative remains editable
- **WHEN** the DM Panel renders a `lair_action` row's expanded panel
- **THEN** the initiative input and its Set button are present and function identically to other entity types

#### Scenario: Name and alias remain editable in the DM Panel
- **WHEN** the DM Panel renders a `lair_action` row's expanded panel
- **THEN** the Name and Alias inputs are present, letting the DM relabel the entity (e.g. from "Lair Action" to "Collapsing Ceiling")

#### Scenario: Visibility toggle available in the DM Panel
- **WHEN** the DM Panel renders a `lair_action` row
- **THEN** the 👁/🙈 toggle button is present and dispatches `toggle_entity_visibility` exactly as it does for creature rows
