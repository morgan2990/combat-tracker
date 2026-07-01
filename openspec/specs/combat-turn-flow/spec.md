# Spec: Combat Turn Flow

## Purpose

Defines how the DM starts a combat encounter and advances turns through the initiative order, including the round counter lifecycle.

## Requirements

### Requirement: DM can start combat to lock the encounter and set the active turn
The DM SHALL be able to send a `start_combat` action that transitions the room into active combat, sets the round counter, and establishes the first active turn. Before allowing combat to start, the server SHALL verify that every `pc` and companion entity in the room has a non-null initiative value.

#### Scenario: DM starts combat — all initiatives set
- **WHEN** a DM-role client sends `{ "type": "start_combat" }` and every `pc` and companion entity in the room has a non-null initiative
- **THEN** the server SHALL set `is_started = true`, `round = 1`, `active_index = 0`, and broadcast the updated `RoomState` to all connected clients

#### Scenario: DM blocked from starting combat — pending initiatives
- **WHEN** a DM-role client sends `start_combat` and one or more `pc` or companion entities have `initiative: null`
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Start combat is idempotent
- **WHEN** a DM sends `start_combat` and `is_started` is already true
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Non-DM cannot start combat
- **WHEN** a player-role client sends `start_combat`
- **THEN** the server SHALL ignore the message and send no broadcast

### Requirement: DM can advance the turn to the next entity in initiative order
The DM SHALL be able to send a `next_turn` action that moves the active turn indicator forward through the sorted entity list.

#### Scenario: DM advances to the next entity
- **WHEN** a DM-role client sends `{ "type": "next_turn" }` and `active_index` is not at the last entity
- **THEN** the server SHALL increment `active_index` by one and broadcast the updated `RoomState`

#### Scenario: Turn wraps to the top and increments the round
- **WHEN** a DM-role client sends `next_turn` and `active_index` is at the last entity in the list
- **THEN** the server SHALL set `active_index = 0`, increment `round` by one, and broadcast the updated `RoomState`

#### Scenario: Next turn requires combat to be started
- **WHEN** a DM sends `next_turn` and `is_started` is false
- **THEN** the server SHALL ignore the message and send no broadcast

#### Scenario: Non-DM cannot advance the turn
- **WHEN** a player-role client sends `next_turn`
- **THEN** the server SHALL ignore the message and send no broadcast
