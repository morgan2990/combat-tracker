## 1. Backend: split api/handler.go by domain

- [x] 1.1 Move `CreateRoom`/`GetRoom` into `api/rooms.go`; move `SignUp`/`Login`/`Logout`/`Me` into `api/auth.go`. No logic changes.
- [x] 1.2 Move PC/companion handlers into `api/pcs.go`, party handlers into `api/parties.go`, custom-monster handlers into `api/custom_monsters.go`, encounter handlers into `api/encounters.go`, official-monster/search handlers into `api/monsters.go`. `api/handler.go` removed (empty after the split).
- [x] 1.3 Run `go build ./...` and `go vet ./...` to confirm the split compiles with no behavior change.

## 2. Backend: shared request helpers

- [ ] 2.1 Add `requireUser(w, r) (userID string, ok bool)`, `decodeJSON(w, r, &body) bool`, and `writeJSON(w, status, v)` helpers (in `api/` alongside the split handler files).
- [ ] 2.2 Replace the repeated auth-check / decode-or-400 / encode-with-status boilerplate in each handler with calls to the new helpers, one file at a time, verifying `go build ./...` after each file.
- [ ] 2.3 Confirm the `"database error"` 500 response text and status codes are unchanged at every call site after the swap.

## 3. Backend: consolidate ID generation and edition validation

- [ ] 3.1 Introduce a single `NewID(n int) string` helper (random hex ID generator) and update `store/user.go`, `store/custom_monster.go`, and `room/room.go` to call it, preserving each site's current byte length.
- [ ] 3.2 Introduce a shared edition-validation helper that takes an explicit policy parameter (default-to-5e vs. reject-with-400), and update the ~6 call sites to use it, preserving each endpoint's current behavior exactly.
- [ ] 3.3 Run `go build ./...` and `go vet ./...` to confirm no behavior change.

## 4. Backend: formatting sweep

- [ ] 4.1 Run `gofmt -l .` to find inconsistently formatted files (including `room/room.go`'s `Entity` struct) and `gofmt -w` them.

## 5. Frontend: finish My Creatures quick-pick extraction

- [ ] 5.1 Extract a shared component covering both the phone-tier pill markup and the tablet/desktop `CustomMonsterList` row-list, and use it in both `EncounterForm.tsx` and `DMView.tsx`'s `AddCreatureForm`, removing the duplicated pill JSX from both.
- [ ] 5.2 Manually verify (as in the prior change) that both tiers still render and add-to-staging/add-to-combat works identically in `EncounterForm` and in `DMView`'s in-room panel.

## 6. Frontend: shared entity vital-state and condition helpers

- [ ] 6.1 Extract the shared `entityVitalState`-style dead/unconscious/alive classifier, the `CONDITIONS` array, and the row-color/text-color mapping into a shared module, used by both `DMView.tsx`'s `EntityRow` and `PlayerView.tsx`.
- [ ] 6.2 Extract the shared condition-toggle button styling/behavior into a shared component, used by both.
- [ ] 6.3 Verify DMView and PlayerView render combatant rows and condition toggles identically to before (screenshot compare or manual check for a dead, unconscious, and active entity).

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
