## Why

The frontend and backend have each grown feature-by-feature without a dedicated pass to consolidate accumulated duplication and boilerplate. None of this affects current behavior, but it's making individual files larger and repeated patterns more likely to drift out of sync when only one copy gets updated.

## What Changes

Pure internal refactors — no observable behavior change, no API contract change, no new capabilities. Each item below is scoped to preserve current behavior exactly; risk is mitigated by running the existing lint/type-check (frontend) and `go build`/`go vet` (backend) after each extraction, and spot-checking against `openspec/specs/` where an item's current behavior is spec'd.

**Backend (Go):**
- Split `api/handler.go` (~900 lines, all endpoint domains in one file) into per-domain files (`auth.go`, `rooms.go`, `pcs.go`, `parties.go`, `monsters.go`, `custom_monsters.go`, `encounters.go`), mirroring the per-domain split `store/` already uses. Pure file move, no logic change.
- Introduce shared request helpers (auth-required check, JSON decode-or-400, JSON response encode) to collapse the same 3-4 line boilerplate currently repeated across ~20+ handlers.
- Consolidate the three separate reimplementations of "generate a random hex ID" (`store/user.go`, `store/custom_monster.go`, `room/room.go`) into one shared helper, preserving each call site's current ID length.
- Consolidate the ~6 duplicated inline `"5e"`/`"5.5e"` edition-validation checks into one shared helper — preserving each endpoint's current accept/reject behavior exactly (`CreateRoom` defaults invalid/omitted edition to `"5e"` per its spec; other endpoints reject invalid values with 400 — the helper takes a parameter for which behavior to use, it does not unify them into one behavior).
- Run `gofmt` across files with inconsistent formatting (e.g. uneven struct tag alignment in `room/room.go`'s `Entity` struct).

**Frontend (React/TypeScript):**
- Extract a shared component covering both the phone-tier pill markup (currently duplicated verbatim between `EncounterForm.tsx` and `DMView.tsx`'s `AddCreatureForm`) and the tablet/desktop row-list presentation.
- Extract a shared `entityVitalState`-style classifier and row-color/condition-toggle helpers between `DMView.tsx`'s `EntityRow` and `PlayerView.tsx`, which currently redefine the same dead/unconscious/alive logic, the same `CONDITIONS` array, and the same state-to-color mapping independently.
- Introduce a shared `fetchJSON`-style helper for the "GET, fall back on non-ok, catch swallows" pattern currently hand-rolled at ~10 call sites across 6 files. Each call site keeps its own error-surfacing behavior on top (some silently fall back, some set an error message) — the helper only consolidates the mechanical fetch/parse/fallback part.
- Extract a shared segmented-toggle component for the "5e"/"5.5e" edition picker, currently duplicated verbatim between `EncounterForm.tsx` and `MonsterForm.tsx`.
- Consolidate the near-identical `labelStyle`/`labelText`/`fieldStyle` constants redefined across 6 form components into one shared module. These have drifted slightly (e.g. 11px vs 12px label text, fixed vs 100% width) — consolidating means picking one canonical value, which is a very small visual normalization, not a pure no-op mechanical change; called out here so it isn't missed during review.

## Deferred (not in this change)

- **Shared theme/design-token module**: no shared color/spacing constants exist anywhere in the repo; the same ~9 hex colors are repeated across all 17 component files (~250 occurrences). Worth doing, but it's a large, purely mechanical pass better done as its own change once the extractions above reduce how many files still hand-roll these values.
- **Splitting `DMView.tsx`** (832 lines) into separate files for `EntityRow`, `AddCreatureForm`, `EncounterTemplatesControl`, and the layout shell. Better done after the `EntityRow`/`PlayerView.tsx` duplication above is resolved, so there's less to move.
- **`CreateRoom`'s malformed-JSON handling**: it currently swallows a JSON decode error and proceeds with a zero-value body (which happens to resolve to the spec'd `"5e"` default), while every other handler in `api/handler.go` returns 400 on decode failure. Whether to tighten this needs a product decision (it would change `CreateRoom`'s current behavior for malformed request bodies), so it's out of scope for a no-behavior-change cleanup pass.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
(none — every item above preserves current behavior; anything that would change behavior is listed under Deferred instead)

## Impact

- `api/handler.go` and `store/`, `room/room.go` (backend) — internal restructuring only, no endpoint contract changes.
- `frontend/src/components/DMView.tsx`, `PlayerView.tsx`, `EncounterForm.tsx`, `MonsterForm.tsx`, `DMNavColumn.tsx`, `Dashboard.tsx`, `CharacterForm.tsx`, and other form components (frontend) — internal restructuring, no rendered-output changes except the label-style value normalization called out above.
- No database schema, API contract, or dependency changes.
