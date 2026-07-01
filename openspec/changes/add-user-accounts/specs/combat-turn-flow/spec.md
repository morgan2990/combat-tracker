## MODIFIED Requirements

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
