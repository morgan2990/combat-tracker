## 1. Backend: split api/handler.go by domain

- [x] 1.1 Move `CreateRoom`/`GetRoom` into `api/rooms.go`; move `SignUp`/`Login`/`Logout`/`Me` into `api/auth.go`. No logic changes.
- [x] 1.2 Move PC/companion handlers into `api/pcs.go`, party handlers into `api/parties.go`, custom-monster handlers into `api/custom_monsters.go`, encounter handlers into `api/encounters.go`, official-monster/search handlers into `api/monsters.go`. `api/handler.go` removed (empty after the split).
- [x] 1.3 Run `go build ./...` and `go vet ./...` to confirm the split compiles with no behavior change.

## 2. Backend: shared request helpers

- [x] 2.1 Add `requireUser(w, r) (userID string, ok bool)`, `decodeJSON(w, r, &body) bool`, and `writeJSON(w, status, v)` helpers (in `api/` alongside the split handler files).
- [x] 2.2 Replace the repeated auth-check / decode-or-400 / encode-with-status boilerplate in each handler with calls to the new helpers, one file at a time, verifying `go build ./...` after each file. `CreateRoom` intentionally left untouched by `decodeJSON` (its lenient decode-swallow behavior is out of scope, tracked by `review-create-room-json-handling`). `SignUp`'s response now gets a `Content-Type: application/json` header via `writeJSON` that it previously lacked (every sibling handler already set it) — a header-only normalization, not a status/body change.
- [x] 2.3 Confirm the `"database error"` 500 response text and status codes are unchanged at every call site after the swap (38 occurrences, matching the pre-change count).

## 3. Backend: consolidate ID generation and edition validation

- [x] 3.1 Introduce a single `NewID(n int) string` helper (random hex ID generator, in `store/user.go`) and update `store/custom_monster.go`, `store/encounter.go`, `store/mongo.go`, `store/party.go`, `room/room.go`, and `api/custom_monsters.go` to call it with 8 bytes, matching every prior call site's byte length. `room/room.go`'s local `newToken` removed (was already byte-identical logic).
- [x] 3.2 Introduce `isValidEdition`/`requireValidEdition`/`resolveEditionOrDefault` in `api/helpers.go` and update all 6 call sites (`rooms.go`, `custom_monsters.go` x3, `monsters.go`, `encounters.go` x2) to use them — `CreateRoom` keeps its default-to-5e policy via `resolveEditionOrDefault`, every other endpoint keeps its reject-with-400 policy via `requireValidEdition`. `encounters.go`'s local `validEncounterEdition` removed.
- [x] 3.3 Run `go build ./...` and `go vet ./...` to confirm no behavior change.

## 4. Backend: formatting sweep

- [x] 4.1 Ran `gofmt -l .`; most flagged files use CRLF line endings, which makes `gofmt -d`/`gofmt -w` treat every line as changed (line-ending normalization noise, not real formatting issues) — running `gofmt -w` on those would silently convert them to LF repo-wide, well outside this task's scope. Instead: hand-realigned `room/room.go`'s `Entity` struct fields to match gofmt's actual output (verified by running gofmt on the struct in isolation), preserving the file's CRLF; ran `gofmt -w` on `store/user.go` (LF-only, genuinely small diff — `User`/`Session` struct field alignment).

## 5. Frontend: finish My Creatures quick-pick extraction

- [x] 5.1 The tablet/desktop row-list was already shared via `CustomMonsterList`/`DMNavColumn` from the prior change — `DMView.tsx`'s `AddCreatureForm` never rendered a row-list itself (that's `DMNavColumn`'s job, elsewhere in the layout), only the phone-tier pill. Extracted the pill row markup into `CustomMonsterPillList.tsx` and used it in both `EncounterForm.tsx`'s phone branch and `AddCreatureForm`, removing the duplicated pill JSX from both. Each caller keeps its own "My Creatures" label wrapper (styling drift between them is task 9's scope).
- [x] 5.2 Manually verified via Playwright: `EncounterForm` at phone width renders the pill and clicking adds to the staging list; `DMView`'s in-room `AddCreatureForm` at phone width renders the pill and clicking populates the Name/Max HP fields — both unchanged from before the extraction.

## 6. Frontend: shared entity vital-state and condition helpers

- [x] 6.1 Extracted `CONDITIONS`, `entityVitalState(dead, currentHP)`, `vitalRowBg(vitalState, isActive, isMe?)`, and `vitalTextColor(vitalState)` into `frontend/src/entityVitals.ts`. `entityVitalState` standardized on `PlayerView`'s primitive-based signature (`dead`, `currentHP`) rather than `DMView`'s Entity-object one, since it's the more general shape; `DMView`'s call site updated to `entityVitalState(entity.dead, entity.current_hp)`. `vitalRowBg` takes an `isMe` parameter (defaulting false) to preserve `PlayerView`'s extra highlight branch that `DMView` doesn't have — not unified away, since it's a real behavioral difference.
- [x] 6.2 Extracted `ConditionToggles` (`frontend/src/components/ConditionToggles.tsx`) — the flex-wrap row of condition-toggle buttons — used by both `DMView.tsx`'s `EntityRow` and `PlayerView.tsx`. Button padding normalized to `4px 10px` (was `3px 9px` in DMView, `4px 10px` in PlayerView) — a 1px visual normalization, not a pure no-op. Each caller keeps its own outer wrapper spacing (DMView's `marginBottom: 12`, PlayerView's none).
- [x] 6.3 Verified via Playwright: DM side — added a creature, toggled "Prone" (condition tag rendered correctly), killed it (dead-state row: dark background, grayed text, "💀 Dead" indicator, all correct). Player side — joined the room with a PC, toggled "Stunned" (active-state red border/background rendered correctly). Screenshots confirm no regression.

## 7. Frontend: shared fetchJSON helper

- [ ] 7.1 Introduce a `fetchJSON<T>(url, fallback)` helper that only wraps the fetch/parse/fallback mechanics, returning a result callers still branch on for their own error-surfacing behavior.
- [ ] 7.2 Replace the ~10 hand-rolled call sites across `DMNavColumn.tsx`, `Dashboard.tsx`, `DMView.tsx`, `EncounterForm.tsx`, `CharacterForm.tsx`, and `MonsterForm.tsx` one file at a time, preserving each site's current fallback/error-surfacing behavior.
- [ ] 7.3 Run `tsc -b` and `oxlint` after each file to confirm no regressions.

## 8. Frontend: shared segmented-toggle component

- [ ] 8.1 Extract a shared segmented-toggle component for the "5e"/"5.5e" edition picker and use it in both `EncounterForm.tsx` and `MonsterForm.tsx`.
- [ ] 8.2 Verify both pickers render and select identically to before.

## 9. Frontend: consolidate label/field style constants

- [ ] 9.1 Introduce a shared module for `labelStyle`/`labelText`/`fieldStyle`, picking the majority value where the 6 current sites differ (e.g. 12px over 11px label text).
- [ ] 9.2 Update the 6 form components to use the shared constants.
- [ ] 9.3 Screenshot the affected forms before/after to confirm the value normalization isn't visually disruptive.

## 10. Verify

- [ ] 10.1 Run `tsc -b` and `oxlint` for the full frontend.
- [ ] 10.2 Run `go build ./...` and `go vet ./...` for the full backend.
- [ ] 10.3 Manually smoke-test the app end-to-end (sign up, create room as DM, add creatures, join as player, view/edit inventory) to confirm no regressions across all the touched areas.
