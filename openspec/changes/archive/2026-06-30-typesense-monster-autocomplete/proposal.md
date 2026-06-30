## Why

Monster search today (`GET /api/search/monsters`) is an exact `{name, edition}` lookup against MongoDB — a DM must type a creature's full, correctly-spelled name and tab away before anything happens, returning at most one result. With ~7,000 monsters already imported from the 5etools bestiaries, this isn't usable as a discovery tool. Epic 12 introduces Typesense as a dedicated, typo-tolerant, prefix-matching search layer mirrored from MongoDB, and reworks the DM's monster search into a live, debounced autocomplete dropdown.

## What Changes

- Add a Typesense `monsters` collection (schema: `id`, `name`, `max_hp`, `initiative_modifier`, `edition`), kept in sync with MongoDB. `id` is MongoDB's own `_id` — `store.Monster` gains an explicit `ID` field so it round-trips on reads (currently unset/unused anywhere in the app).
- Unify monster persistence behind a single `store.MonsterStore.UpsertMonster` used by both the live API (manual monster creation) and the scrubber CLI: switches from `ReplaceOne` to `FindOneAndReplace` (so the resulting `_id` is always available), performs the MongoDB write, then best-effort upserts into Typesense. A Typesense failure is logged, not surfaced as a request/entry failure — MongoDB remains the source of truth, consistent with the scrubber's existing per-entry log-and-continue behavior.
- Add a guard in the shared upsert: a non-custom (scrubber-sourced) write SHALL NOT overwrite an existing document already marked `is_custom: true`. This makes re-running the scrubber a safe, permanent backfill/refresh mechanism that can't clobber DM customizations, now or later.
- `cmd/scrubber` is rewired to import `combatapp/store` instead of maintaining its own duplicated `Monster` struct and MongoDB connector; its 5etools-specific parsing/normalization logic is unchanged.
- Backfill of the ~7,000 already-imported monsters is operational, not code: re-running the scrubber once against the existing local 5etools checkout populates Typesense for all of them via the same upsert path.
- **BREAKING (spec-level)**: `GET /api/search/monsters` changes its query engine (MongoDB exact match → Typesense typo-tolerant/prefix search) and its response shape (single-or-empty array of full `Monster` docs → a top-N array of lightweight hits: `id`, `name`, `max_hp`, `initiative_modifier`). This directly supersedes `monster-search`'s current Purpose note, which states the response shape stays identical through Epic 12 — that note is now incorrect and is corrected as part of this change.
- DM Combat Panel's monster search is reworked from an on-blur exact lookup into a live dropdown: no request fires below 3 characters; a ~175ms debounce governs requests at 3+ characters; dropping back below 3 characters instantly clears the dropdown with no request. Selecting a result (click or Enter) clears the search box, autofills the staging Name/Max HP fields, and fires one follow-up `GET /api/monsters/{name}` call (existing endpoint, unchanged) to recover `source_type`/`reference_url`/`pdf_object_key` for statblock-reference linking — fields intentionally absent from the lean Typesense schema and search response.

## Capabilities

### New Capabilities
- `monster-search-index`: the Typesense `monsters` collection schema, the shared dual-write upsert path (MongoDB + Typesense) with its custom-monster-preservation guard, and the re-run-scrubber-to-backfill mechanism.

### Modified Capabilities
- `monster-search`: the query endpoint switches from MongoDB exact match to Typesense typo-tolerant/prefix/edition-filtered search with a narrowed response shape; the DM Combat Panel requirement is rewritten for the debounced dropdown UX (3-char threshold, debounce, instant-clear, selection autofill + statblock-reference follow-up fetch).
- `monster-repository`: `store.Monster` gains an `id` field surfaced in `POST`/`GET /api/monsters` responses; the manual-creation upsert now also dual-writes to Typesense via the shared upsert path.
- `monster-scrubber`: the tool now imports `combatapp/store` rather than maintaining its own MongoDB connector and `Monster` struct; each upsert also dual-writes to Typesense, subject to the custom-monster-preservation guard.

## Impact

- `store/monster.go`: add `ID` field to `Monster`; rework `UpsertMonster` to use `FindOneAndReplace`, add the custom-monster guard, add the Typesense dual-write call.
- New `store/typesense.go` (or similar): Typesense client init, collection schema setup, upsert/search helpers.
- `cmd/scrubber/main.go`: drop local `Monster` struct and Mongo connector; import `combatapp/store`; keep 5etools parsing/normalization logic.
- `api/handler.go`: `SearchMonsters` rewritten to query Typesense instead of `store.SearchMonsters`; `UpsertMonster`/`GetMonster` responses now include `id`.
- `go.mod`: new Typesense Go client dependency.
- New `docker-compose.typesense.yml` (or equivalent), following the existing `docker-compose.mongodb.yml` pattern; new `TYPESENSE_*` env vars mirroring `MONGODB_URI`/`MINIO_*` conventions.
- `frontend/src/components/DMView.tsx` (`AddCreatureForm`): replaces on-blur exact lookup with a debounced search-box + dropdown, separate from the staging Name/Max HP/Qty fields; adds the selection-time follow-up fetch for statblock-reference fields.
