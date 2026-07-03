# Spec: Party Room Claiming

## Purpose

Defines how a Room automatically links to a Party the first time one of that party's members completes character setup in the room, so the room can later expose party-scoped features (such as the DM's aggregated party inventory view) without requiring explicit DM configuration. The link is established silently — it is not broadcast to players and does not prevent unaffiliated PCs, or a second party's members, from also joining the same room.

## Requirements

### Requirement: A room claims a party on first affiliated join
A `Room` SHALL carry an optional `party_id`, unset by default. When a player's `setup_character` action succeeds and the room's `party_id` is unset, the server SHALL look up whether the joining PC belongs to a party; if it does, the server SHALL set the room's `party_id` to that party's ID. This claim SHALL NOT be included in the broadcast `RoomState` and SHALL NOT trigger a `BroadcastState()` call.

#### Scenario: First party-affiliated player claims the room
- **WHEN** a PC belonging to party P completes `setup_character` in a room with no `party_id` set
- **THEN** the server SHALL set the room's `party_id` to P's ID without broadcasting any WebSocket message

#### Scenario: Second member of the same party joining does not change the claim
- **WHEN** a PC belonging to party P completes `setup_character` in a room whose `party_id` is already P
- **THEN** the server SHALL leave the room's `party_id` unchanged

#### Scenario: A party is not reassigned once claimed
- **WHEN** a PC belonging to party Q completes `setup_character` in a room whose `party_id` is already set to a different party P
- **THEN** the server SHALL leave the room's `party_id` as P; the joining PC's items and currency SHALL NOT appear in the room's aggregated party inventory view

#### Scenario: An unaffiliated PC does not claim the room
- **WHEN** a PC that belongs to no party completes `setup_character` in a room with no `party_id` set
- **THEN** the server SHALL leave the room's `party_id` unset

#### Scenario: An unaffiliated PC joining first does not block a later claim
- **WHEN** an unaffiliated PC completes `setup_character` in a room with no `party_id` set, and afterward a PC belonging to party P completes `setup_character` in the same room
- **THEN** the server SHALL set the room's `party_id` to P's ID

#### Scenario: A PC belonging to multiple parties resolves deterministically
- **WHEN** the joining PC belongs to more than one party
- **THEN** the server SHALL claim the room using one party chosen by a stable, deterministic rule (not a random or unspecified selection), and SHALL NOT treat the ambiguity as an error
