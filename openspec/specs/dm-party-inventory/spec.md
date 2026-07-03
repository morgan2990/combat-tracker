# Spec: DM Party Inventory

## Purpose

Defines DM-only, room-scoped read/write access to the aggregated inventory of a room's linked party: every member's items in a flat, owner-tagged list, the party's pooled currency, and the sum of members' personal currency, with per-member filtering in the DM panel and a single-member shortcut from an entity row. Edits made through this view are silent — they persist directly to the affected PC's document without any WebSocket broadcast, matching how player-initiated inventory edits already behave.

## Requirements

### Requirement: DM can fetch the aggregated inventory of the room's linked party
The system SHALL provide `GET /api/rooms/{room_id}/party-inventory`, restricted to the requester matching the room's `owner_user_id`. The response SHALL include: `items`, an array of `{ owner_pc_id, owner_name, name, quantity }` covering every item instance across every member of the room's linked party; `members`, an array of `{ pc_id, name, currency }` giving each member's identity and personal currency (the data source the client filters against — see "Aggregated view is filterable by party member"); `party_currency`, the party's pooled currency (`Party.Currency`, unchanged); and `members_currency_total`, the sum across all members of each member's personal `PC.Currency`, denomination-wise.

#### Scenario: DM fetches the aggregated view
- **WHEN** the room's owner sends `GET /api/rooms/{room_id}/party-inventory` for a room whose `party_id` is set
- **THEN** the server SHALL respond with HTTP 200, an `items` array covering every member PC's items with owner references, a `members` array with each member's identity and personal currency, a `party_currency` object, and a `members_currency_total` object

#### Scenario: Non-owner cannot fetch the aggregated view
- **WHEN** a user who is not the room's `owner_user_id` sends `GET /api/rooms/{room_id}/party-inventory`
- **THEN** the server SHALL respond with HTTP 403 and include no inventory data

#### Scenario: Room has no linked party
- **WHEN** the room's owner sends `GET /api/rooms/{room_id}/party-inventory` for a room whose `party_id` is unset
- **THEN** the server SHALL respond with a response indicating no linked party, without error, and with an empty item list

### Requirement: DM can edit a party member's items and currency from the aggregated view
The system SHALL provide `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}`, restricted to requests where the requester matches the room's `owner_user_id`, the room's `party_id` is set, and `pc_id` is a member of that party. A successful request SHALL overwrite the target PC's `items` and/or `currency` the same way `PUT /api/pcs/:id` does for the PC's own owner.

#### Scenario: DM edits a member's items
- **WHEN** the room's owner sends `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` with an updated `items` array, for a `pc_id` in the room's linked party
- **THEN** the server SHALL save the updated `items` array on that PC's document and return HTTP 200

#### Scenario: DM edits a member's personal currency
- **WHEN** the room's owner sends `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` with an updated `currency` object, for a `pc_id` in the room's linked party
- **THEN** the server SHALL save the updated `currency` on that PC's document and return HTTP 200; the room's linked party's pooled `Party.Currency` SHALL NOT be affected

#### Scenario: Edit rejected for a PC outside the linked party
- **WHEN** the room's owner sends `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` for a `pc_id` that is not a member of the room's linked party
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Edit rejected for a non-owner of the room
- **WHEN** a user who is not the room's `owner_user_id` sends `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}`
- **THEN** the server SHALL respond with HTTP 403 and make no change

#### Scenario: Negative currency values are rejected
- **WHEN** the room's owner sends `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` with any `currency` field less than `0`
- **THEN** the server SHALL respond with HTTP 400 and make no change

### Requirement: Aggregated view is filterable by party member
The DM-facing panel SHALL let the DM filter the item list to a single party member. Filtering SHALL be applied client-side against the single fetched response, without an additional request. When filtered to a single member, the displayed "members' currency" figure SHALL show only that member's personal currency; the party's pooled currency figure SHALL remain visible and unchanged by the filter.

#### Scenario: DM filters to one player
- **WHEN** the DM selects a specific party member in the panel's filter control
- **THEN** the client SHALL display only that member's item rows and that member's personal currency, alongside the unfiltered party pooled currency, without issuing a new request to the server

#### Scenario: DM clears the filter
- **WHEN** the DM clears the player filter after having selected one
- **THEN** the client SHALL display item rows for every member again and revert the members' currency figure to the sum across all members

### Requirement: Party inventory edits are silent
Edits made through `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` SHALL NOT trigger any WebSocket broadcast and SHALL NOT alter `RoomState`. The affected player's client SHALL only reflect the change the next time it independently fetches that PC's data (e.g. opening their own `InventoryPanel`).

#### Scenario: Edit does not broadcast to the room
- **WHEN** the DM edits a party member's items or currency via the aggregated view
- **THEN** no WebSocket message SHALL be sent to any client in the room as a result of that edit

### Requirement: Entity row inventory icon reflects party-membership access
The 🎒 inventory icon on a `pc`-type entity row in DMView SHALL be enabled only when that PC is a member of the room's currently linked party. When enabled and clicked by the DM, it SHALL open `InventoryPanel` sourced from the room-scoped endpoints defined above rather than `GET /api/pcs/:id`. A player opening their own inventory via this icon uses `/api/pcs/:id`.

#### Scenario: Icon enabled for a linked-party member
- **WHEN** the DM views an entity row for a PC that is a member of the room's linked party
- **THEN** the 🎒 icon SHALL be enabled, and clicking it SHALL open `InventoryPanel` populated via the room-scoped endpoint for that PC

#### Scenario: Icon disabled when no party is linked
- **WHEN** the DM views a `pc`-type entity row in a room whose `party_id` is unset
- **THEN** the 🎒 icon SHALL be disabled (or hidden) for that row

#### Scenario: Icon disabled for a PC outside the linked party
- **WHEN** the DM views a `pc`-type entity row for a PC that is not a member of the room's linked party
- **THEN** the 🎒 icon SHALL be disabled (or hidden) for that row
