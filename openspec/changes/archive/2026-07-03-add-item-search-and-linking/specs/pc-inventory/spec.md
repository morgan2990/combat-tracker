## MODIFIED Requirements

### Requirement: A PC has a personal item list
Each PC document SHALL include an `items` array of `{ name: string, quantity: number, catalog_item_id?: string, reference_url?: string }` entries, editable only by the PC's owner via the existing `PUT /api/pcs/:id` endpoint. `name` is a free-text label with no uniqueness constraint; multiple entries with the same `name` MAY exist. `catalog_item_id` and `reference_url` are optional and both present only when the entry was created or last edited via a catalog selection (see `item-search`); a free-text entry omits both.

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

#### Scenario: An item saved with catalog reference fields round-trips them
- **WHEN** a PC's owner sends `PUT /api/pcs/:id` with an item entry including `catalog_item_id` and `reference_url`
- **THEN** the server SHALL save both fields on that entry and return them unchanged on the next fetch

#### Scenario: An item saved without catalog reference fields omits them
- **WHEN** a PC's owner sends `PUT /api/pcs/:id` with an item entry that has no `catalog_item_id` or `reference_url`
- **THEN** the server SHALL save the entry with both fields absent, not defaulted to empty strings

## ADDED Requirements

### Requirement: InventoryPanel's item-name field offers catalog search with a reference link
`InventoryPanel`'s item-name input SHALL offer a debounced search dropdown against `GET /api/search/items`, using the same interaction conventions as monster search (no request below 3 characters, ~175ms debounce, dropdown selectable by click or Enter). The input SHALL remain freely editable as free text at all times, whether populated by a selection or typed directly.

Selecting a dropdown result SHALL: fill the row's `name` from the hit; issue a follow-up `GET /api/items/{id}` to recover `reference_url`; and set the row's `catalog_item_id` and `reference_url` accordingly. If a row in the same list already has a matching `catalog_item_id`, the selection SHALL increment that row's `quantity` instead of adding a new row. Editing a row's `name` directly after a catalog selection SHALL clear that row's `catalog_item_id` and `reference_url`.

Any row with a non-empty `reference_url` SHALL display a link icon that opens the URL in a new tab.

#### Scenario: No request below 3 characters
- **WHEN** the user has typed 2 characters into an item-name field
- **THEN** the frontend SHALL NOT dispatch any request to `GET /api/search/items`

#### Scenario: Selecting a new catalog item adds a row
- **WHEN** the user selects a dropdown result for an item not already present (by `catalog_item_id`) in the current list
- **THEN** the client SHALL add a new row with that item's name, `catalog_item_id`, and `reference_url`, and issue `GET /api/items/{id}` to populate `reference_url`

#### Scenario: Selecting an already-present catalog item increments quantity
- **WHEN** the user selects a dropdown result whose `id` matches an existing row's `catalog_item_id` in the current list
- **THEN** the client SHALL increment that row's `quantity` and SHALL NOT add a new row

#### Scenario: Free-text entries never auto-merge
- **WHEN** the user types a name matching an existing free-text row's name, without selecting a dropdown result
- **THEN** the client SHALL add a new row rather than merging with the existing one

#### Scenario: Editing a catalog-linked row's name clears its reference
- **WHEN** the user directly edits the `name` of a row that has a `catalog_item_id`
- **THEN** the client SHALL clear that row's `catalog_item_id` and `reference_url`

#### Scenario: Link icon shown only for catalog-linked rows
- **WHEN** a row has a non-empty `reference_url`
- **THEN** the client SHALL display a link icon next to that row that opens `reference_url` in a new tab

#### Scenario: Free-text row has no link icon
- **WHEN** a row has no `reference_url`
- **THEN** the client SHALL NOT display a link icon for that row
