## MODIFIED Requirements

### Requirement: DM can fetch the aggregated inventory of the room's linked party
The system SHALL provide `GET /api/rooms/{room_id}/party-inventory`, restricted to the requester matching the room's `owner_user_id`. The response SHALL include: `items`, an array of `{ owner_pc_id, owner_name, name, quantity, catalog_item_id?, reference_url? }` covering every item instance across every member of the room's linked party (the last two fields present only for entries created or last edited via a catalog selection); `members`, an array of `{ pc_id, name, currency }` giving each member's identity and personal currency (the data source the client filters against — see "Aggregated view is filterable by party member"); `party_currency`, the party's pooled currency (`Party.Currency`, unchanged); and `members_currency_total`, the sum across all members of each member's personal `PC.Currency`, denomination-wise.

#### Scenario: DM fetches the aggregated view
- **WHEN** the room's owner sends `GET /api/rooms/{room_id}/party-inventory` for a room whose `party_id` is set
- **THEN** the server SHALL respond with HTTP 200, an `items` array covering every member PC's items with owner references, a `members` array with each member's identity and personal currency, a `party_currency` object, and a `members_currency_total` object

#### Scenario: Non-owner cannot fetch the aggregated view
- **WHEN** a user who is not the room's `owner_user_id` sends `GET /api/rooms/{room_id}/party-inventory`
- **THEN** the server SHALL respond with HTTP 403 and include no inventory data

#### Scenario: Room has no linked party
- **WHEN** the room's owner sends `GET /api/rooms/{room_id}/party-inventory` for a room whose `party_id` is unset
- **THEN** the server SHALL respond with a response indicating no linked party, without error, and with an empty item list

#### Scenario: Catalog-linked items include their reference in the aggregated view
- **WHEN** a party member has an item entry with `catalog_item_id` and `reference_url` set
- **THEN** the corresponding row in the aggregated `items` array SHALL include both fields

## ADDED Requirements

### Requirement: PartyInventoryPanel's item-name fields offer the same catalog search as InventoryPanel
Each party member's item-name input in `PartyInventoryPanel` SHALL offer the same debounced search dropdown against `GET /api/search/items`, with the same selection, merge-by-`catalog_item_id`, name-edit-clears-reference, and link-icon behavior specified for `InventoryPanel` in `pc-inventory`, scoped independently to each member's own item list — a catalog selection made while editing one member SHALL NOT affect any other member's rows.

#### Scenario: Selecting a catalog item for one member does not affect another
- **WHEN** the DM selects a catalog result while editing one party member's items
- **THEN** only that member's item list SHALL be affected; other members' item lists SHALL remain unchanged

#### Scenario: Merge-by-catalog-id is scoped per member
- **WHEN** the DM selects a catalog result that matches an existing `catalog_item_id` on a *different* member's row, while editing a member who does not yet have that item
- **THEN** the client SHALL add a new row to the member currently being edited rather than incrementing the other member's row

#### Scenario: Link icon shown per row regardless of filter state
- **WHEN** the DM views the aggregated panel, filtered or unfiltered, and a row has a non-empty `reference_url`
- **THEN** the client SHALL display a link icon next to that row that opens `reference_url` in a new tab
