## Context

Monster search already solved "typo-tolerant, prefix-matching, edition-filtered autocomplete over a large reference dataset, with free-text entry preserved and reference-link fields fetched on selection rather than indexed" (`openspec/specs/monster-search/spec.md`, `monster-search-index/spec.md`, and their original design doc at `openspec/changes/archive/2026-06-30-typesense-monster-autocomplete/design.md`). Item entry today (`pc-inventory`) is a bare `{ name, quantity }` array with no search, no reference linking, and no distinction between a real catalog item and any string a player types.

5etools' item data (available locally via the same `SCRUBBER_SOURCE_5E`/`SCRUBBER_SOURCE_5_5E` checkouts already used for the bestiary) is split across two files with different shapes: `items-base.json`'s `baseitem[]` and `items.json`'s `item[]` (generic equipment categories like "Longsword"/"Breastplate", and named/magic items like "Rope of Climbing"). Both need scrubbing for "standard equipment + magic items" to actually be searchable — a player looking for a mundane longsword and a DM looking for a specific magic sword are searching two different source tables.

Unlike the bestiary, item data doesn't split cleanly across the two checkouts: `SCRUBBER_SOURCE_5_5E` is a strict superset (230 `baseitem` / 2,428 `item` entries, vs. `SCRUBBER_SOURCE_5E`'s 124 / 1,649), containing both 2014-sourced entries (`PHB`, `DMG`, `MM`, ...) and 2024-sourced entries (`XPHB`, `XDMG`, `XMM`) side by side, often linked via `reprintedAs`. There is exactly one checkout worth scrubbing for items.

Items also don't have static per-item pages the way bestiary entries do. The site's own routing (`UrlUtil.PG_ITEMS` in its `js/utils.js`) shows item pages are addressed via a hash fragment on `items.html`, built from `UrlUtil.encodeArrayForHash(name, source)` — not a `/items/{slug}.html` static path. A `reference_url` has to follow that hash-routed form to actually open on the item.

## Goals / Non-Goals

**Goals:**
- Debounced search/autocomplete on the item-name field in both `InventoryPanel` (player, own PC) and `PartyInventoryPanel` (DM, any linked-party member), reusing the monster-search UX conventions (175ms debounce, 3-char minimum, dropdown, Enter-selects-top).
- A small external-link icon next to any item row backed by a catalog selection, opening its 5etools page in a new tab.
- Selecting a catalog item already present on the same list (matched by stable catalog id) increments its quantity instead of adding a duplicate row.
- Free-text entry stays fully intact — the dropdown and link icon are purely additive, exactly as established for monsters.

**Non-Goals:**
- No "custom item" entity, no ownership/privacy model, no per-item CRUD flow. An unmatched typed name is just free text, saved exactly as `pc-inventory` already saves it today — nothing new is persisted or made searchable beyond what a player types into their own list.
- No embedded item-page viewer. Unlike monster statblocks (needed live, mid-combat), an item lookup is occasional reference — a plain link is enough.
- No changes to `monster-search`, `monster-search-index`, or any monster-related code. This is a parallel, independent catalog and search endpoint, not a shared one.

## Decisions

**1. One capability covers the catalog, index, and search endpoint.**
`item-search` covers the Typesense schema, the shared upsert path, the scrubber's item-parsing code, and the `GET /api/search/items` endpoint in a single spec file — one cohesive unit, rather than split across separate catalog/index and consumer capabilities the way `monster-search-index`/`monster-search` are.

**2. The item catalog is simpler than the monster catalog: no `is_custom`, no `private`, no `owner_id`.**
Directly follows from the Non-Goal above. The new `store.CatalogItem` type carries `id, name, edition, category, rarity, reference_url, five_etools_id, source_book` — no authorship fields at all, since there's no custom-item concept to protect or filter by. This also means the custom-monster-preservation guard `monster-search-index` has for its own catalog has no analog here: there's nothing for a scrubber re-run to accidentally clobber, because every document in the item catalog always comes from the scrubber.

**3. Typesense schema stays lean; reference fields are fetched on selection, not indexed.**
Indexed: `id, name (facet), edition (facet), category (facet), rarity (facet, optional)`. Not indexed: `reference_url`, `five_etools_id`, `source_book` — these aren't needed for ranking/filtering/display in the dropdown. On selection, the frontend makes one follow-up call, `GET /api/items/{id}`, to recover them — the same shape as `GET /api/monsters/{name}`, except keyed by id (the item catalog's Mongo `_id`, so the catalog always exposes a real opaque id) rather than name, since the two source tables (`baseitem`/`item`) could plausibly produce name collisions across categories in a way bestiary entries didn't.

**4. `Item` (the PC-embedded type) gains two optional fields: `catalog_item_id` and `reference_url`.**
Both absent for free-text entries. Both set together, at selection time, from the `GET /api/items/{id}` follow-up call — never re-resolved live on render, so a stale/renamed catalog entry doesn't retroactively break an already-saved inventory row. Editing an item's `name` field directly after a catalog selection clears both fields (mirrors monster search: editing the staging Name field clears `monsterRef`), since a hand-edited name no longer reliably corresponds to the selected catalog entry.

**5. Merge-on-select matches by `catalog_item_id`, never by name string.**
When the user selects a catalog result, the client checks the current item list (whichever list is being edited — a single PC's in `InventoryPanel`, or one member's in `PartyInventoryPanel`) for an existing row whose `catalog_item_id` equals the selected item's id. If found, increment that row's `quantity`; if not, append a new row. Two free-text rows that happen to share a name never merge — there's no stable id to match on, and silently merging based on string equality would surprise a player who typed the same name for two narratively distinct things.

**6. The scrubber gains an items code path on the same `cmd/scrubber` binary, not a new tool.**
Unlike the monster path, the items path takes only `--source` (pointed at `SCRUBBER_SOURCE_5_5E`) — no `--edition` flag, since edition is derived per entry (Decision 7), not selected by which checkout is scrubbed. Parses both `items-base.json`'s `baseitem[]` (translating `type` codes — stripping any trailing `|SOURCE` suffix, e.g. `"AF|DMG"` → `"AF"` — via `items-base.json`'s own `itemType` lookup table into a human-readable category) and `items.json`'s `item[]` (resolving `_copy` inheritance the same two-pass way the bestiary parser already does, at similarly low prevalence — ~2% of entries). Both feed the same shared `UpsertItem` path into one unified `items` collection, keyed by `{name, edition}` as monsters are.

**7. Edition is derived per item from its 5etools source code, not from which checkout is scrubbed.**
Only three source codes are 2024/5.5e core-rulebook content: `XPHB`, `XDMG`, `XMM` (the 2024 Player's Handbook/DMG/Monster Manual). Every other source code — including similarly-`X`-prefixed but unrelated ones like `XGE` (Xanathar's Guide to Everything, 2017) — is 5e content. An entry's `edition` is `"5.5e"` if its `source` is one of those three codes, else `"5e"`. This is a source-code allowlist, not a prefix match, since prefix alone (`X...`) produced a false positive on `XGE`.

**8. `reference_url` is a hash-routed link into `items.html`, always on the `5e.tools` domain.**
Confirmed against the site's own routing code: `UrlUtil.URL_TO_HASH_BUILDER["item"]`/`["baseitem"]` both resolve to `it => UrlUtil.encodeArrayForHash(it.name, SourceUtil.getEntitySource(it))`, and `encodeArrayForHash` lowercases and URL-encodes each part before joining with `_`. So `reference_url` is built as `https://5e.tools/items.html#{urlencode(lowercase(name))}_{lowercase(source)}` — e.g. "Longsword" (source `PHB`) → `https://5e.tools/items.html#longsword_phb`. Always `5e.tools`, never `2014.5e.tools`, since only the 5.5e checkout is scrubbed (Decision 7) and that site serves both 5e- and 5.5e-sourced items.

## Risks / Trade-offs

- **[Risk] `baseitem[]` and `item[]` could theoretically produce a name collision** (a base category and a specific item sharing a name) since they're merged into one `{name, edition}`-keyed collection. → **Mitigation**: last-write-wins via the shared upsert, same as any other same-key collision in this codebase; not worth a dedup/conflict-detection mechanism at this data scale — acceptable per the same reasoning the monster catalog already accepts for its own key collisions.
- **[Risk] The 5e/5.5e source-code allowlist (Decision 7) could go stale** if a future 5etools data update introduces another 2024-line core book. → **Mitigation**: the allowlist is a 3-element list in one place in the scrubber; low-cost to extend if that happens, and no existing item's edition tag would silently change since the allowlist only adds new "5.5e" matches, never removes existing ones.
- **[Risk] Two independent frontend call sites (`InventoryPanel`, `PartyInventoryPanel`) need identical search/select/merge/link behavior.** Unlike monster search (`AddCreatureForm` is the only consumer), duplicating this logic in two components risks drift. → **Mitigation**: factor the debounce/search/select/merge logic into a shared hook or component during implementation rather than copy-pasting; this is an implementation detail for tasks.md, not a spec-level requirement, since both panels' existing specs already describe the same *behavior*, just not shared *code*.

## Migration Plan

1. Ship backend (Typesense schema/init, shared `UpsertItem`, scrubber items path, `GET /api/search/items`, `GET /api/items/{id}`) and frontend (search UX in both panels, link icon, merge logic) together — same reasoning as the monster precedent: the frontend has nothing to call until the backend exists, so there's no safe partial rollout.
2. After deploy, run the scrubber's items path once against `SCRUBBER_SOURCE_5_5E` (the only checkout items are scrubbed from — see Decision 7) to populate the catalog and its Typesense mirror.
3. No migration needed for existing `PC.Items` data — the two new fields are optional and simply absent on every pre-existing item.
4. Rollback: revert the deploy. No destructive changes to `PC.Items` or any other existing collection.

## Open Questions

- Exact CLI shape for telling the scrubber to process items vs. monsters (new flag vs. separate subcommand) — an implementation detail, not a spec-level concern, left for tasks.md.
