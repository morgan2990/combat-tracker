## 1. Schema & Index Migration

- [x] 1.1 Add `Edition string`, `InitiativeModifier int`, and `IsCustom bool` fields to the `Monster` struct in `store/monster.go`
- [x] 1.2 Update `UpsertMonster` in `store/monster.go` to use `bson.M{"name": m.Name, "edition": m.Edition}` as the upsert filter
- [x] 1.3 Add compound unique index creation to `store.Init()` in `store/mongo.go`: drop any existing `{ name: 1 }` unique index on the `monsters` collection, then create `{ name: 1, edition: 1 }` unique index

## 2. API Handler Updates

- [x] 2.1 Update the JSON path of `UpsertMonster` in `api/handler.go`: require `edition` in the request body (return HTTP 400 if missing or not `"5e"`/`"5.5e"`), and set `m.IsCustom = true` before upserting
- [x] 2.2 Update the multipart path of `UpsertMonster` in `api/handler.go`: read `edition` from form values (return HTTP 400 if missing or invalid), and set `IsCustom: true` on the `Monster` before upserting

## 3. CLI Scrubber — Setup & Flag Parsing

- [x] 3.1 Create `cmd/scrubber/main.go` with `--source` and `--edition` flags; validate that both are present and `--edition` is one of `"5e"` or `"5.5e"`, exiting with a non-zero status and a descriptive message on failure
- [x] 3.2 Connect to MongoDB in the scrubber using the same `MONGODB_URI` env var logic as `store.Init()`

## 4. CLI Scrubber — Core Logic

- [x] 4.1 Implement bestiary file discovery: glob all files matching `bestiary-*.json` under `<source>/data/bestiary/`; log a warning and exit non-zero if no files are found
- [x] 4.2 Implement JSON parsing: for each file, unmarshal the top-level `monster` array; skip and log a warning for files that fail to parse
- [x] 4.3 Implement field normalization per monster entry: `hp.average → MaxHP`, `name → Name`, `source → SourceBook`; handle missing `hp` or `average` fields gracefully (skip entry with warning)
- [x] 4.4 Implement `InitiativeModifier` calculation: `int(math.Floor(float64(dex-10) / 2))`; default to `0` with a warning log if the `dex` field is absent
- [x] 4.5 Implement `FiveEToolsID` and `ReferenceURL` generation: `five_etools_id = "{name-kebab}-{source-lower}"`, base URL from `--edition` flag (`https://2014.5e.tools/bestiary/` for `5e`, `https://5e.tools/bestiary/` for `5.5e`), final URL = `{base}{five_etools_id}.html`
- [x] 4.6 Set `SourceType = "url"`, `IsCustom = false`, `Edition` from the `--edition` flag on every normalized document
- [x] 4.7 Upsert each document using `{ name, edition }` as the filter; track inserted vs updated counts using the `UpsertedCount` and `MatchedCount` fields on the MongoDB result
- [x] 4.8 Print completion summary to stdout: `Done: N processed, N inserted, N updated`

## 5. Verification

- [x] 5.1 Run `go run ./cmd/scrubber --source <path-to-5e-repo> --edition 5e` and confirm documents land in MongoDB with correct `edition`, `initiative_modifier`, `is_custom`, `reference_url`, and `five_etools_id` values on a sample of entries
- [x] 5.2 Run `go run ./cmd/scrubber --source <path-to-5.5e-repo> --edition 5.5e` and confirm separate `edition: "5.5e"` documents exist alongside the 5e ones for shared monster names (e.g. Goblin)
- [x] 5.3 Re-run the 5e scrub and confirm the document count in MongoDB does not increase (idempotency check)

<!-- Tasks 5.1–5.3 require local 5etools repos and a running MongoDB instance — run manually -->
