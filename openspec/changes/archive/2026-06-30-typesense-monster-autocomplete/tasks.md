## 1. Typesense infrastructure

- [x] 1.1 N/A — an existing self-hosted Typesense instance is already running (`192.168.0.94:8108`, API key `xyz`); no new docker-compose file needed.
- [x] 1.2 Add the official Typesense Go client to `go.mod`.
- [x] 1.3 Add `store/typesense.go`: client init (`TYPESENSE_URL`/`TYPESENSE_API_KEY` env vars, falling back to `http://192.168.0.94:8108` / `xyz` — mirroring the hardcoded-LAN-IP-default convention already used for `MONGODB_URI`/`MINIO_ENDPOINT`), `monsters` collection schema creation (id, name [facet], max_hp, initiative_modifier [optional], edition [facet]) if not already present, called from `store.InitTypesense()`, invoked from `main.go` alongside `store.InitMinio()`. A connection/init failure SHALL be logged, not fatal (no `log.Fatalf`).

## 2. Shared monster persistence (MongoDB + Typesense)

- [x] 2.1 Add `ID bson.ObjectID \`bson:"_id,omitempty" json:"id,omitempty"\`` to `store.Monster` (uses `bson.ObjectID`, not a plain string, so it round-trips correctly through the MongoDB driver; it marshals to JSON as a hex string automatically).
- [x] 2.2 Rework `store.MonsterStore.UpsertMonster` to use `FindOneAndReplace` with `SetUpsert(true)` and `SetReturnDocument(options.After)` instead of `ReplaceOne`, so the returned `Monster` always carries a populated `id`.
- [x] 2.3 Add the custom-monster preservation guard: before writing, if an existing document at `{name, edition}` has `IsCustom: true` and the incoming `Monster` has `IsCustom: false`, skip the write (log it) and return the existing document unchanged.
- [x] 2.4 After a successful MongoDB upsert (and not skipped by 2.3), upsert the same document into the Typesense `monsters` collection (mapping to its schema fields) keyed by the MongoDB `id`. Log failures; do not return them as errors from `UpsertMonster`.
- [x] 2.5 Add `SearchMonsters(query, edition string) ([]MonsterHit, error)` to `MonsterStore` (delegates to a Typesense-backed package function) with typo tolerance + prefix matching on `name`, filtered by `edition`, returning lightweight hits (`id`, `name`, `max_hp`, `initiative_modifier`). Returns an empty slice (not an error) if Typesense is unreachable.

## 3. Scrubber unification

- [x] 3.1 Update `cmd/scrubber/main.go` to import `combatapp/store` instead of declaring its own `Monster` struct and `connectMongo()` function.
- [x] 3.2 Replace the scrubber's local `upsert()` closure with calls to `store.GlobalMonsters.UpsertMonster`, keeping the existing 5etools-specific parsing/normalization (`_copy` resolution, two-pass dependency resolution, `reference_url`/`five_etools_id` construction) unchanged.
- [x] 3.3 Verify the scrubber's existing per-entry log-and-continue behavior still applies when a write is skipped by the custom-monster guard (2.3) or when the Typesense mirror fails (2.4) — neither should abort the run or be miscounted in the completion summary in a misleading way. Implementation note: `UpsertMonster` now returns an `UpsertOutcome` (`Inserted`/`Updated`/`SkippedCustomProtected`) so the scrubber can keep its `processed`/`inserted`/`updated` summary counters byte-compatible with the existing (unchanged) "completion summary" requirement — skipped entries are logged (inside `UpsertMonster`) but don't increment any summary counter, same as how Typesense-mirror failures are logged internally without affecting the MongoDB-write counts.

## 4. Search endpoint

- [x] 4.1 Rewrite `api.SearchMonsters` to call the new Typesense-backed search (2.5) instead of the old Mongo exact match, returning the top matching hits as a JSON array of `{id, name, max_hp, initiative_modifier}`.
- [x] 4.2 Update `api.UpsertMonster` (both JSON and multipart paths) and `api.GetMonster` to include the document's `id` in their JSON responses — `api.GetMonster` needed no code change since `store.Monster` now carries `ID` directly and `json.NewEncoder(w).Encode(m)` already serializes the full struct; `api.UpsertMonster` now encodes the `saved` document returned by `UpsertMonster` instead of echoing back the request body.

## 5. Frontend: debounced autocomplete dropdown

- [x] 5.1 In `AddCreatureForm` (`DMView.tsx`), split state: a transient search-query string (cleared on selection) separate from the staging Name/Max HP/Quantity fields.
- [x] 5.2 Implement the 3-character threshold: no request fires below 3 characters; dropping back below 3 characters instantly clears/closes the dropdown with no request.
- [x] 5.3 Implement a ~175ms debounce (hand-rolled `useEffect`/`setTimeout`, no new dependency) for requests at 3+ characters, including cleanup of pending timers on each keystroke.
- [x] 5.4 Render the dropdown: each result shows name, edition badge, and `max_hp`.
- [x] 5.5 Implement selection (click or `Enter`): clear the search input, close the dropdown, autofill staging Name/Max HP from the selected hit, then fire `GET /api/monsters/{name}` to recover `source_type`/`reference_url`/`pdf_object_key` and populate `monsterRef` (preserving the existing "statblock ready" indicator behavior). `Enter` selects the top result (no arrow-key highlight navigation — not required by the ACs).
- [x] 5.6 Confirm the staging Name/Max HP fields remain freely editable regardless of whether a dropdown selection was made (free-text/homebrew entry must keep working) — unchanged from the prior implementation, verified by inspection and `tsc -b`/`vite build`/`oxlint` all passing clean.

## 6. Backfill (operational)

- [ ] 6.1 After deploying, manually re-run `go run ./cmd/scrubber --source <path> --edition 5e` and `--edition 5.5e` against the existing local 5etools checkout to backfill Typesense for the ~7,000 already-imported monsters. (No code task — documented here so it isn't forgotten.)

## 7. Tests

- [x] 7.1 Unit tests for the custom-monster preservation guard. Extracted the decision into a pure `shouldPreserveCustom(existingIsCustom, incomingIsCustom bool) bool` function (no Mongo dependency) and added table-driven tests in `store/monster_test.go` covering all four cases.
- [ ] 7.2 Unit tests for `FindOneAndReplace`-based upsert returning a populated `id` on both insert and replace paths. **Deferred** — same reason as the Epic 11 persistence work: no Mongo mocking/interface layer or separate test database exists in this codebase; this specifically tests MongoDB driver behavior (not pure application logic), so it can't be made dependency-free the way 7.1 was. The only reachable MongoDB is the live instance the running app uses. Verify manually if needed.
- [ ] 7.3 Frontend tests (or manual verification) for the 3-character threshold, debounce timing, instant-clear on drop-below-3, and selection autofill + follow-up fetch behavior. **Partially covered**: `tsc -b`, `vite build`, and `oxlint` all pass clean, confirming type-correctness, but there is no test runner configured in this frontend (no vitest/jest in `package.json`) to exercise actual runtime behavior (debounce timing, dropdown interaction). Setting one up is a larger lift than this task implies — deferred; recommend interactive verification in a browser instead.
