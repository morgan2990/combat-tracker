# Spec: End Combat

## Purpose

Defines the rules for terminating an active combat encounter: the server-side state transitions when `end_combat` is received, the DM client confirmation flow required before sending the action, and the post-combat view updates applied to all connected clients.

## Requirements

### Requirement: DM can end an active combat encounter
The DM SHALL be able to send an `end_combat` action that terminates the current encounter, removes all creature entities, retains player and companion entities, and resets the combat state fields.

#### Scenario: DM ends combat
- **WHEN** a DM-role client sends `{ "type": "end_combat" }` and `is_started` is true
- **THEN** the server SHALL remove all entities with `type == "creature"`, retain all entities with `type == "player"`, retain all entities with `type == "companion"` whose `owner_id` matches the `id` of a remaining player entity, set `is_started = false`, `round = 0`, `active_index = 0`, and broadcast the updated `RoomState` to all connected clients

#### Scenario: Orphaned companions are removed
- **WHEN** `end_combat` is processed and a companion entity's `owner_id` does not match any remaining player entity's `id`
- **THEN** the server SHALL remove that companion along with the creatures

#### Scenario: Player and companion state is preserved
- **WHEN** `end_combat` is processed
- **THEN** surviving player and companion entities SHALL retain their current `current_hp`, `temp_hp`, `conditions`, `dead`, and `initiative` values unchanged

#### Scenario: End combat requires active combat
- **WHEN** a DM sends `end_combat` and `is_started` is false
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Non-DM cannot end combat
- **WHEN** a player-role client sends `end_combat`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: DM client shows an inline confirmation before ending combat
The DM client SHALL require explicit confirmation before sending `end_combat`, using an inline UI pattern that does not require a browser dialog.

#### Scenario: DM clicks End Combat for the first time
- **WHEN** the DM clicks the "End Combat" button
- **THEN** the client SHALL replace the button with an inline confirmation row showing a warning message and "Cancel" and "Yes, End Combat" buttons; no message SHALL be sent to the server yet

#### Scenario: DM cancels the confirmation
- **WHEN** the DM clicks "Cancel" in the confirmation row
- **THEN** the client SHALL restore the original "End Combat" button and send no message to the server

#### Scenario: DM confirms end combat
- **WHEN** the DM clicks "Yes, End Combat" in the confirmation row
- **THEN** the client SHALL send `{ "type": "end_combat" }` over the WebSocket

### Requirement: All clients return to the waiting state after combat ends
After `end_combat` is broadcast, all connected clients SHALL reflect the between-encounter state without requiring a page reload.

#### Scenario: Player view after end combat
- **WHEN** a player-role client receives a `RoomState` with `is_started = false` after combat was active
- **THEN** the client SHALL display the initiative list without an active turn indicator and show the "Waiting for DM to start combat" banner

#### Scenario: DM view after end combat
- **WHEN** the DM-role client receives the post-end-combat `RoomState`
- **THEN** the DM panel SHALL hide the "Next Turn" and "End Combat" controls and show the "Start Combat" button; the tracker SHALL display only the surviving players and companions
