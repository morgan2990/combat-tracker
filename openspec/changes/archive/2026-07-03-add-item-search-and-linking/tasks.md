## 1. 5etools items URL pattern and edition classification

- [x] 1.1 Confirmed: item pages are hash-routed on `items.html` (`https://5e.tools/items.html#{urlencoded-lowercase-name}_{lowercase-source}`), not a `/items/{slug}.html` static path — see design.md Decision 8
- [x] 1.2 Confirmed: `SCRUBBER_SOURCE_5_5E` is a strict superset of `SCRUBBER_SOURCE_5E` for items; only that checkout is scrubbed, with edition derived per entry from a `{XPHB, XDMG, XMM}` source-code allowlist — see design.md Decisions 6-7

## 2. Item catalog data model and Typesense schema

- [x] 2.1 Add `store.CatalogItem` (`id, name, edition, category, rarity, reference_url, five_etools_id, source_book`), a new MongoDB-backed type/collection distinct from the existing PC-embedded `store.Item`
- [x] 2.2 Add a Typesense `items` collection schema (`id, name, edition, category, rarity`) and initialize it at startup alongside the existing `monsters` collection, non-fatal on failure
- [x] 2.3 Implement a shared `UpsertItem` using `FindOneAndReplace` (upsert, `ReturnDocument: After`), keyed by `{name, edition}`, mirroring `UpsertMonster`
- [x] 2.4 Implement best-effort MongoDB→Typesense mirroring on every item upsert, logging (not failing) on Typesense errors

## 3. Scrubber items path

- [x] 3.1 Add an items-processing path to `cmd/scrubber` (flag/subcommand TBD) reading `items-base.json`'s `baseitem[]` and `items.json`'s `item[]` from the `--source` checkout (no `--edition` flag — edition is derived per entry, task 3.2)
- [x] 3.2 Derive each entry's `edition` from its `source` code (`"5.5e"` if `source` is `XPHB`/`XDMG`/`XMM`, else `"5e"`)
- [x] 3.3 Build a `type`-code → human-readable `category` lookup from `items-base.json`'s `itemType` table (stripping any trailing `|SOURCE` from the entry's `type` before lookup) and apply it to both `baseitem` and `item` entries
- [x] 3.4 Resolve `_copy` inheritance in `item[]` entries (two-pass, mirroring the existing bestiary `_copy` resolution)
- [x] 3.5 Build each entry's `reference_url` as `https://5e.tools/items.html#{urlencoded-lowercase-name}_{lowercase-source}`
- [x] 3.6 Call the shared `UpsertItem` path for every parsed entry from both source tables

## 4. Search and lookup endpoints

- [x] 4.1 Add `GET /api/search/items?q=&edition=` (auth required, both params required, edition validated, typo-tolerant/prefix Typesense query, empty array on Typesense-unavailable, returns `{id, name, category, rarity}[]`)
- [x] 4.2 Add `GET /api/items/{id}` (auth required, full catalog document including `reference_url`, 404 if not found)
- [x] 4.3 Register both routes in `main.go`

## 5. PC-embedded Item type gains catalog reference fields

- [x] 5.1 Add optional `catalog_item_id`/`reference_url` fields to `store.Item` (the PC-embedded type) and the frontend `Item` type in `types.ts`
- [x] 5.2 Confirm `PUT /api/pcs/:id` and `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` round-trip both fields unchanged (no special handling needed if they're just additional struct/type fields flowing through existing `$set` writes — verify, don't assume)

## 6. Shared search/select/merge/link frontend logic

- [x] 6.1 Factor the debounced-search, selection, merge-by-`catalog_item_id`, name-edit-clears-reference, and link-icon behavior into a shared hook or component (per design.md's risk about code drift between two consumers)
- [x] 6.2 Wire it into `InventoryPanel.tsx`'s item-name field
- [x] 6.3 Wire it into `PartyInventoryPanel.tsx`'s per-member item-name fields, confirming merge/selection stays scoped to the member currently being edited

## 7. Aggregated party-inventory response carries reference fields

- [x] 7.1 Add `catalog_item_id`/`reference_url` to `partyInventoryItemRow` in `api/rooms.go`'s `GetPartyInventory`
- [x] 7.2 Update `PartyInventoryPanel.tsx`'s item-grouping logic (currently strips extra fields down to `{name, quantity}` when building `itemsByOwner`) to preserve both fields through to rendering

## 8. Verification

- [x] 8.1 Run the scrubber's items path against `SCRUBBER_SOURCE_5_5E`; spot-check a `baseitem`-sourced entry and an `item`-sourced (including a `_copy`-resolved) entry in MongoDB and via `GET /api/search/items`, in both `5e` and `5.5e` editions
- [x] 8.2 Manually verify: typing fewer than 3 characters issues no request; typo-tolerant and prefix search both return expected results; results are edition-scoped
- [x] 8.3 Manually verify in `InventoryPanel`: selecting a catalog item adds a row with a working link icon; selecting the same item again increments quantity instead of duplicating; typing a free-text name never gets a link icon; editing a catalog-linked row's name clears its link
- [x] 8.4 Manually verify the same in `PartyInventoryPanel`, including that a selection made for one party member does not affect another member's list
- [x] 8.5 Manually verify Typesense-unavailable behavior: stop Typesense, confirm search returns an empty array (HTTP 200) rather than an error, and that server startup is unaffected
- [x] 8.6 Confirm no changes to monster search behavior (`GET /api/search/monsters`, `AddCreatureForm`) — this change should be fully additive and parallel
