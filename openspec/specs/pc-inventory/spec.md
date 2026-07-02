# Spec: PC Inventory

## Purpose

Defines the personal item list and currency purse carried by each PC, and the InventoryPanel UI used to view and edit them from Dashboard and from an active room.

## Requirements

### Requirement: A PC has a personal item list
Each PC document SHALL include an `items` array of `{ name: string, quantity: number }` entries, editable only by the PC's owner via the existing `PUT /api/pcs/:id` endpoint. `name` is a free-text label with no uniqueness constraint; multiple entries with the same `name` MAY exist.

#### Scenario: Owner adds an item to their PC
- **WHEN** a PC's owner sends `PUT /api/pcs/:id` with `items` including `{ "name": "Healing Potion", "quantity": 2 }`
- **THEN** the server SHALL save the updated `items` array on the PC document and return HTTP 200

#### Scenario: Owner removes an item from their PC
- **WHEN** a PC's owner sends `PUT /api/pcs/:id` with an `items` array that omits a previously saved entry
- **THEN** the server SHALL overwrite the stored `items` array to match the submitted array (the omitted entry no longer appears)

#### Scenario: Item edit rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` with an `items` change for a PC whose `owner_user_id` does not match the requester
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: New PC starts with an empty item list
- **WHEN** a PC is created via `POST /api/pcs` without an `items` field
- **THEN** the server SHALL store `items` as an empty array

### Requirement: A PC has a personal currency purse
Each PC document SHALL include a `currency` object with integer fields `pp`, `gp`, `ep`, `sp`, `cp` (platinum, gold, electrum, silver, copper), editable only by the PC's owner via `PUT /api/pcs/:id`. Each denomination is stored and edited independently; the system SHALL NOT auto-convert between denominations.

#### Scenario: Owner updates their PC's currency
- **WHEN** a PC's owner sends `PUT /api/pcs/:id` with `currency: { "pp": 1, "gp": 15, "ep": 0, "sp": 4, "cp": 12 }`
- **THEN** the server SHALL save the updated `currency` object on the PC document and return HTTP 200

#### Scenario: Currency edit rejected for a PC owned by someone else
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` with a `currency` change for a PC whose `owner_user_id` does not match the requester
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: New PC starts with zeroed currency
- **WHEN** a PC is created via `POST /api/pcs` without a `currency` field
- **THEN** the server SHALL store `currency` with all five denominations set to `0`

#### Scenario: Negative currency values are rejected
- **WHEN** an authenticated client sends `PUT /api/pcs/:id` with any `currency` field less than `0`
- **THEN** the server SHALL return HTTP 400 and make no change to the stored PC

### Requirement: Inventory is viewable and editable from an InventoryPanel reachable from Dashboard and an active room
A shared `InventoryPanel` UI component SHALL display a PC's `items` and `currency` and allow the PC's owner to edit them, backed by the existing PC REST endpoints. The panel SHALL be launchable from the PC list in Dashboard and from a PC's entity row in DMView/PlayerView during an active combat session, without emitting any WebSocket message or altering `RoomState`.

#### Scenario: Panel opened from Dashboard
- **WHEN** a user clicks the inventory action on one of their own PCs in Dashboard
- **THEN** the client SHALL open the InventoryPanel for that PC, fetching its current `items` and `currency` via REST

#### Scenario: Panel opened mid-combat
- **WHEN** a user clicks the inventory action on a PC's entity row inside an active room (DMView or PlayerView)
- **THEN** the client SHALL open the InventoryPanel for that PC via REST without sending any WebSocket message or affecting the live `RoomState`

#### Scenario: Item rows are edited as a batch, not per-row requests
- **WHEN** a user adds, edits, or removes rows in the InventoryPanel's item list and saves
- **THEN** the client SHALL submit the entire updated `items` array in a single `PUT /api/pcs/:id` request
