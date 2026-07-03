## Why

Adding an inventory item today means typing a bare name into a free-text field, with no way to find the right item quickly, no link to its actual rules text, and no distinction between "Healing Potion" the real item and any other string a player happens to type. Monster search already solved an equivalent problem for the bestiary — this reuses that pattern for the 5etools item catalog (standard equipment and magic items) while keeping free-text entry fully intact for homebrew items.

## What Changes

- A new item catalog, scrubbed from 5etools' item data (`items-base.json`'s generic equipment categories and `items.json`'s named/magic items) into MongoDB and mirrored into a lean Typesense `items` index, following the same shared-upsert, best-effort-dual-write, non-fatal-startup pattern already established for monsters.
- A new `GET /api/search/items` endpoint: typo-tolerant, prefix-matching, edition-filtered search over the catalog.
- The item-name field in both `InventoryPanel` (a player editing their own inventory) and `PartyInventoryPanel` (a DM editing a linked party's inventory) gains a debounced search dropdown. Free-text entry stays fully available — the dropdown is purely additive, exactly as it is for `AddCreatureForm`.
- Selecting a catalog result stores a stable catalog reference (id) and `reference_url` directly on that `Item` entry, and a small external-link icon next to the row opens the item's 5etools page in a new tab. No embedded viewer — item lookups are occasional reference, not live-combat-urgent the way monster statblocks are.
- Selecting a catalog item that's already present on the same list (matched by catalog id, not name string) increments its quantity instead of adding a duplicate row. Free-text entries never auto-merge with each other or with catalog picks.
- No new "custom item" entity, ownership, or privacy model. An item that doesn't match the catalog is just free text, saved exactly as it is today — nothing new is persisted, searched, or reused beyond the PC's own `items` array.

## Capabilities

### New Capabilities
- `item-search`: the item catalog (MongoDB + Typesense), its 5etools-sourced scrubber path, and the `GET /api/search/items` endpoint.

### Modified Capabilities
- `pc-inventory`: `Item` gains optional catalog-reference fields; `InventoryPanel`'s item-name field gains search/autocomplete, a reference-link icon, and merge-on-select-by-catalog-id behavior.
- `dm-party-inventory`: `PartyInventoryPanel`'s per-member item-name fields gain the same search/autocomplete, link icon, and merge behavior as `InventoryPanel`.

## Impact

- **Backend**: new `store.CatalogItem` collection/type (distinct from the existing `store.Item` embedded on a PC) and Typesense schema/init; `cmd/scrubber` gains an items code path alongside its existing monster path; new `api/items.go`-equivalent search handler; `store.Item` (the embedded PC item type) gains optional fields.
- **Frontend**: `InventoryPanel.tsx` and `PartyInventoryPanel.tsx` both gain the debounced-search + link-icon + merge behavior; `types.ts`'s `Item` type gains the new optional fields; a new shared item-search hook/component is likely worth factoring out given two call sites need identical behavior (unlike `AddCreatureForm`, which is the only monster-search consumer).
- **Data dependency**: relies on a local 5etools source checkout (`SCRUBBER_SOURCE_5E`/`SCRUBBER_SOURCE_5_5E`, already configured) containing `items-base.json` and `items.json`; no new external dependency.
- **No changes** to `monster-search`, `monster-search-index`, or any monster-related code — this is a parallel, independent catalog, not a shared one.
