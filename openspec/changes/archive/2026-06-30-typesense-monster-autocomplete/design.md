## Context

MongoDB's `monsters` collection (~7,000 documents from 5etools bestiary imports, plus future DM-custom entries) is queried today via an exact `{name, edition}` `FindOne` with a unique index on those same two fields — no text index, no fuzzy/prefix matching. Monster persistence currently has two independent write paths: `store.MonsterStore.UpsertMonster` (used by the live API's manual-creation endpoint) and `cmd/scrubber/main.go`'s own hand-rolled `Monster` struct and MongoDB connector (fully decoupled from the `store` package). Neither path has any concept of an application-level document `id` — MongoDB auto-assigns `_id` but nothing in the app reads it back.

## Goals / Non-Goals

**Goals:**
- Typo-tolerant, prefix-matching, edition-filtered autocomplete search over the full monster set, fast enough for live-as-you-type use.
- A single, shared write path for monster persistence (replacing the scrubber's duplicated logic) that keeps MongoDB and Typesense in sync for both the scrubber and the live API.
- A backfill story for the ~7,000 already-imported monsters that doesn't require new tooling.
- Preserve the existing statblock-reference linking (`source_type`/`reference_url`/`pdf_object_key`) through the new search flow without bloating the search index with it.

**Non-Goals:**
- No transactional/two-phase-commit guarantee between MongoDB and Typesense — Typesense is a best-effort mirror, MongoDB is the sole source of truth.
- No dedicated reindex/migration tool — backfill is "re-run the existing scrubber."
- No request de-duplication/race-guarding for out-of-order debounced search responses on the frontend (acceptable given local-network latency at this project's scale).
- No change to the existing `GET /api/monsters/{name}` or `GET /api/monsters/{name}/pdf` endpoints' contracts.

## Decisions

### 1. Typesense `id` = MongoDB's real `_id`
`store.Monster` gains `ID string \`bson:"_id,omitempty" json:"id,omitempty"\`` so it round-trips on reads. This is exposed in `POST`/`GET /api/monsters` JSON responses (previously absent entirely). The alternative — using the existing `{name, edition}` composite as a synthetic id — was considered and rejected: it's a smaller diff, but the epic's AC1 explicitly asks for "the MongoDB document ID," and having a real opaque id available is generally useful for any future per-monster reference from the frontend, not just Typesense's internal bookkeeping.

### 2. Unify monster persistence behind one `store.MonsterStore.UpsertMonster`
Both AC2 (scrubber) and AC3 (manual creation) need to (a) reliably obtain the resulting `_id` after an upsert and (b) keep Typesense in sync — identical requirements. Today's `ReplaceOne`-based upsert only returns the new `_id` on a fresh insert (`result.UpsertedID`), not when replacing an existing document, so it can't satisfy (a) as written. The fix: switch to `FindOneAndReplace` with `options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)`, which always returns the full resulting document (including `_id`) regardless of insert-vs-replace. This single method becomes the one place responsible for the MongoDB write, the `_id` retrieval, and the Typesense dual-write — used by both the scrubber and the live API, rather than two independent implementations that could silently diverge.

`cmd/scrubber/main.go` is rewired to import `combatapp/store` and call this shared method, dropping its local `Monster` struct and `connectMongo()` function. Its 5etools-specific parsing (`_copy` resolution, two-pass dependency resolution, URL/`five_etools_id` construction) stays in the scrubber — that logic doesn't belong in `store`.

### 3. Dual-write failure mode: best-effort, MongoDB-first
`UpsertMonster` writes to MongoDB first; only on success does it attempt the Typesense upsert. A Typesense failure is logged but does **not** fail the MongoDB write or the caller's request/entry. Rationale: there is no real cross-database transaction available here, and "failing" the caller after MongoDB already committed would be reporting failure for an operation that partially succeeded. This also matches the scrubber's existing per-entry philosophy (`log.Printf("warning: upsert failed..."); return` already lets one bad entry not abort the whole batch) and the precedent from the prior room-state-persistence work (secondary/derived stores are allowed to be eventually consistent; the primary store is never blocked on them). Typesense holds no data that doesn't already exist authoritatively in MongoDB, so a missed write only ever produces "temporarily absent or stale in search," never data loss or corruption — and it self-heals on the next successful write to that monster.

### 4. Custom-monster preservation guard
Inside the shared `UpsertMonster`, before writing: if a document already exists at `{name, edition}` with `is_custom: true`, and the incoming write has `is_custom: false` (i.e., it originates from the scrubber, not the manual-creation API), the write is skipped (logged, not applied) and the existing document is returned unchanged. DM edits to their own custom monsters (`is_custom: true` incoming) always proceed normally. This makes re-running the scrubber a permanently safe operation — today, with zero custom monsters in the live database, this guard is inert, but it prevents a real, silent data-loss scenario the moment any custom monster is created and a scrubber re-run (e.g., for a future bestiary update) follows.

### 5. Backfill is operational, not code
Re-running `go run ./cmd/scrubber --source <path> --edition <edition>` against the same local 5etools checkout already used (per Epic 8/9) re-processes and re-upserts all ~7,000 entries through the now-Typesense-aware shared upsert path — a complete backfill with zero new tooling. Safe by construction once Decision 4's guard is in place. This needs to happen once, manually, after this change ships; it is not triggered automatically.

### 6. Search index stays lean; statblock-reference fields are fetched on selection, not indexed
Typesense's schema (AC1) and the search response (AC2 AC4) are deliberately limited to `id`, `name`, `max_hp`, `initiative_modifier`, `edition` — no `source_type`/`reference_url`/`pdf_object_key`. These fields aren't needed for ranking/filtering/display in the dropdown, and keeping them out of Typesense avoids re-syncing PDF/URL metadata into a system that doesn't need it. When a DM selects a dropdown result, the frontend makes one additional call to the existing `GET /api/monsters/{name}` endpoint (unchanged, already returns the full document) to recover those fields before finalizing the staged creature — preserving today's "statblock ready" linking without widening Typesense's schema or the search endpoint's response shape. This call happens once per selection, not per keystroke.

This decision directly corrects `openspec/specs/monster-search/spec.md`'s current Purpose note, which states Epic 12 will leave the response shape identical — that was written speculatively before this design existed and is now wrong; the modified spec replaces that claim.

### 7. Typesense startup failure is non-fatal
Unlike `store.Init()`'s `log.Fatalf` on a MongoDB connection failure, a Typesense connection/init failure at startup SHALL be logged but SHALL NOT prevent the server from starting. This follows directly from Decision 3 (MongoDB is the source of truth; Typesense is a derived, best-effort layer) — the same reasoning that makes a single failed write non-fatal also means a fully unavailable Typesense shouldn't take down room/combat functionality that has nothing to do with monster search. `GET /api/search/monsters` SHALL return an empty result set (not a 500) if Typesense is unreachable at query time, rather than degrading to the old Mongo exact-match behavior (which would reintroduce two different search code paths to maintain).

### 8. New infrastructure follows existing conventions
A new `docker-compose.typesense.yml` mirrors the existing `docker-compose.mongodb.yml` pattern (self-hosted, not Typesense Cloud, consistent with how Mongo and MinIO are run). Connection config follows the existing `MONGODB_URI`/`MINIO_ENDPOINT` convention: env vars with hardcoded LAN-IP fallback defaults for local development, overridable via environment in deployment.

## Risks / Trade-offs

- **[Risk]** Best-effort dual-write means Typesense can drift from MongoDB under sustained Typesense unavailability (every write during that window is silently missed). → **Mitigation**: re-running the scrubber is always available as a full resync, and any individual monster self-heals on its next successful write; acceptable at this project's scale per Decision 3.
- **[Risk]** `FindOneAndReplace` is a slightly heavier operation than `ReplaceOne` (returns the full document rather than just a write result). → **Mitigation**: negligible at this collection size and write frequency; not a meaningful performance concern here.
- **[Risk]** The custom-monster guard depends on the incoming write correctly setting `is_custom`. If the scrubber or any future write path ever sent `is_custom: true` for bulk-imported data, the guard would not protect anything. → **Mitigation**: `is_custom` is already a hardcoded, single-purpose field (always `false` from the scrubber, always `true` from the manual-creation API) — no behavior change needed here, just relying on an invariant that already exists.
- **[Risk]** Splitting the search box from the staging Name field (frontend) is a UX behavior change for an existing, working flow. → **Mitigation**: free-text/homebrew entry remains fully intact (staging fields stay always-editable); the dropdown is purely additive.

## Migration Plan

1. Ship the backend changes (Typesense client, schema init, shared `UpsertMonster`, rewritten search endpoint, scrubber rewire) and frontend changes together — the search endpoint's response-shape change is a breaking change to its existing single caller (`AddCreatureForm`), so there's no safe partial-rollout split between them.
2. After deploy, manually re-run the scrubber once per edition (`--edition 5e` and `--edition 5.5e`) against the existing local 5etools checkout to backfill Typesense for all ~7,000 existing monsters.
3. Rollback: revert the deploy. MongoDB data is untouched by any of this (Typesense is purely additive), so no data migration needs to be undone.

## Open Questions

None outstanding — all forks surfaced during exploration were resolved during that conversation.
