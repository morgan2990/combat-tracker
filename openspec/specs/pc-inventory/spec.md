# Spec: PC Inventory

## Purpose

Defines the personal item list and currency purse carried by each PC, and the InventoryPanel UI used to view and edit them from Dashboard and from an active room.

## Requirements

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

### Requirement: CurrencyEditor offers a delta-entry modal alongside absolute-value editing

`CurrencyEditor` SHALL offer a delta-entry modal, opened via a trigger next to its existing absolute-value inputs (which remain unchanged and always available). The modal SHALL present one signed text field per denomination (`pp`, `gp`, `ep`, `sp`, `cp`), accepting an optional leading `+` or `-` followed by digits (bare digits with no sign SHALL be treated as positive). The modal SHALL have a single "Apply" and a single "Cancel" action for the whole modal, not a per-field commit.

Applying SHALL compute each denomination's new absolute value as `current + delta`, clamped to a minimum of `0` (matching the clamp `CurrencyEditor`'s absolute-value inputs already apply), for every denomination with a non-empty, validly-formatted delta field. A denomination whose delta field is left empty SHALL be unchanged. The modal SHALL call `onChange` exactly once with the resulting `Currency` object, then close. Canceling SHALL close the modal without calling `onChange`.

#### Scenario: Applying a single-denomination delta
- **WHEN** a user opens the delta modal, enters `+10` in the `gp` field, leaves all other fields empty, and clicks Apply
- **THEN** `CurrencyEditor` SHALL call `onChange` once with `gp` equal to its previous value plus `10` and every other denomination unchanged

#### Scenario: Applying deltas across multiple denominations at once
- **WHEN** a user enters deltas in more than one denomination field and clicks Apply
- **THEN** `CurrencyEditor` SHALL call `onChange` once with all entered deltas applied simultaneously to their respective denominations

#### Scenario: A delta that would go negative clamps to zero
- **WHEN** a user enters a delta for a denomination whose current value plus that delta would be less than `0` (e.g. `-8` against a current value of `5`)
- **THEN** `CurrencyEditor` SHALL set that denomination to `0` in the resulting `Currency` passed to `onChange`, without an error

#### Scenario: Canceling discards all entered deltas
- **WHEN** a user enters one or more deltas and clicks Cancel instead of Apply
- **THEN** `CurrencyEditor` SHALL close the modal without calling `onChange`, leaving `currency` unchanged

#### Scenario: Bare digits are treated as a positive delta
- **WHEN** a user enters a denomination's delta field with digits only and no leading sign (e.g. `10`)
- **THEN** `CurrencyEditor` SHALL treat it identically to `+10`

#### Scenario: Absolute-value inputs remain available and unaffected
- **WHEN** a user has the delta modal available but instead edits a denomination directly in `CurrencyEditor`'s existing absolute-value input
- **THEN** the existing absolute-value editing behavior SHALL be unaffected by the presence of the delta modal
