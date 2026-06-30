## Context

Epic 7 introduced the monster repository — a MongoDB-backed store where a DM manually registers creatures one at a time via `POST /api/monsters`. The `monsters` collection currently uses `name` as a unique key and holds stat defaults plus an optional statblock reference (URL or PDF).

To support edition-aware autocomplete (Epic 12) and make the app usable out of the box, the collection needs thousands of pre-populated creatures from both the 2014 5e and 2024 5.5e rulesets. The 5etools project maintains structured JSON bestiary files for both editions, making it the practical data source for a personal tool.

The current `Monster` struct already includes `five_etools_id` and `source_book` fields (added during Epic 7 implementation) but they are not formally scoped or used. This change formalizes them.

## Goals / Non-Goals

**Goals:**
- Extend the monster schema with `edition`, `initiative_modifier`, and `is_custom`
- Replace the `{ name }` unique index with `{ name, edition }` so both editions coexist
- Build a CLI tool that reads local 5etools bestiary JSON and bulk-upserts into MongoDB
- Auto-generate `reference_url` for every scraped monster so the statblock drawer works immediately

**Non-Goals:**
- No edition filtering on `GET /api/monsters/:name` — that ambiguity is deferred to Epic 9 (room edition context)
- No Typesense integration — that is Epic 12's responsibility
- No scheduled or server-triggered scrubbing — CLI only
- No frontend changes

## Decisions

### Decision: CLI tool over admin HTTP endpoint
**Chosen:** Standalone `cmd/scrubber/main.go`, invoked manually via `go run ./cmd/scrubber --source <path> --edition <edition>`.  
**Rejected:** `POST /api/admin/scrub` endpoint.  
**Rationale:** The scrubber runs once per edition against a locally cloned repo. It has no need for the HTTP server, authentication, or multipart handling. A CLI avoids that coupling and is simpler to run, test, and re-run independently.

### Decision: `is_custom` bool over separate collections
**Chosen:** Single `monsters` collection with `is_custom: bool` distinguishing scrubbed (`false`) vs DM-created (`true`).  
**Rejected:** `monsters` (scrubbed) + `custom_monsters` (DM-created) as two separate collections.  
**Rationale:** Downstream consumers (Epic 12 Typesense sync, autocomplete endpoint) query one collection. Splitting collections doubles the query surface for no behavioral gain at this stage.

### Decision: Compound index `{ name, edition }` as unique key
**Chosen:** Drop `{ name: 1, unique: true }`, create `{ name: 1, edition: 1, unique: true }`.  
**Rejected:** Keeping `{ name }` unique and appending edition as a suffix to the name (e.g., `"Goblin (5e)"`).  
**Rationale:** Name-suffixing pollutes display names and breaks the autocomplete match against the 5etools URL slug. A compound index keeps names clean and makes the upsert filter unambiguous.

### Decision: Auto-generate `reference_url` from name + source_book
**Chosen:** Scrubber derives `reference_url` deterministically: name lowercased with spaces as hyphens, source code lowercased, joined as `{name-kebab}-{source-lower}.html` under the edition-specific base URL.  
**Rejected:** Leaving `reference_url` null for scraped monsters (DM fills in manually).  
**Rationale:** Auto-generation means the statblock drawer (Epic 7) works for all 3000+ scraped monsters with zero manual effort. The `five_etools_id` field stores the slug fragment (`{name-kebab}-{source-lower}`) so it can be regenerated if the URL format changes.

### Decision: Edition-specific base URLs
- `"5e"` → `https://2014.5e.tools/bestiary/`
- `"5.5e"` → `https://5e.tools/bestiary/`

The base URL is determined by the `--edition` flag, not inferred from the source filename. This makes the mapping explicit and keeps the scrubber independent of 5etools' internal file naming conventions.

### Decision: `initiative_modifier` derived at import time
**Chosen:** Compute `floor((dex - 10) / 2)` during scrub and store the result.  
**Rejected:** Store raw `dex` and compute at query time.  
**Rationale:** Epic 12's Typesense schema indexes `initiative_modifier` directly. Pre-computing avoids a derived-field problem in the search index. Note: Go integer division truncates toward zero — use `math.Floor(float64(dex-10) / 2)` to correctly handle odd dex values below 10.

## Risks / Trade-offs

- **5etools URL format changes** → generated `reference_url` values break silently. Mitigation: `five_etools_id` is stored on every document, so a re-scrub regenerates all URLs without data loss.
- **Index migration on a populated collection** → In development this collection is likely empty or small; the migration is safe. If the collection has manual entries from Epic 7 testing, they will lack `edition` and fail the new compound index. Mitigation: set `edition: "5e"` as a default on pre-existing documents before creating the new index, or drop and re-seed.
- **`GetMonsterByName` ambiguity** → After this change, two documents can share the same name. The existing `GET /api/monsters/:name` endpoint returns whichever MongoDB finds first. This is a known gap: it will remain ambiguous until Epic 9 introduces room-level edition context. No mitigation needed now — the endpoint is not on the critical path for the scrubber.

## Migration Plan

1. Drop the existing `{ name: 1 }` unique index on the `monsters` collection.
2. For any pre-existing documents without an `edition` field, backfill `edition: "5e"` and `is_custom: true`.
3. Create the new compound unique index `{ name: 1, edition: 1 }`.
4. Run the scrubber for each edition: `go run ./cmd/scrubber --source <path> --edition 5e` and `--edition 5.5e`.

Rollback: The scrubber is additive (upserts only). Rolling back means dropping the new index and re-creating the old `{ name: 1 }` unique index, after removing any duplicate-name documents added by the scrubber.

## Open Questions

- Should DM-created monsters with `is_custom: true` be excluded from Typesense indexing (Epic 12), or should custom monsters also be searchable? (Deferred to Epic 12.)
- Should the scrubber skip monsters with no `dex` field (some 5etools entries for objects/traps), or assign `initiative_modifier: 0` as a default? (Recommend: assign 0 with a warning log.)
