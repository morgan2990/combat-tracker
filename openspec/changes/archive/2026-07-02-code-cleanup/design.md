## Context

The proposal lists ten extraction/consolidation items spanning both the Go backend and the React frontend, none of which change observable behavior. This design covers how to sequence and verify them safely, since "pure refactor" claims are only as good as the verification behind them — several near-identical findings from the initial survey turned out to already be intentional, spec'd behavior (`CreateRoom`'s edition default, `CreatePC` ignoring `items`/`currency` at creation), so each item here was cross-checked against `openspec/specs/` before being scoped in.

## Goals / Non-Goals

**Goals:**
- Reduce duplication and file size in the areas identified, with zero observable behavior change.
- Each extraction independently verifiable (type-check/lint for frontend, `go build`/`go vet` for backend) so a regression in one item doesn't block the rest.
- Preserve every endpoint's and component's current behavior exactly, including behavior that looks inconsistent across sibling endpoints/components but is intentional (e.g. `CreateRoom`'s edition default vs. other endpoints' strict rejection).

**Non-Goals:**
- Introducing a shared theme/design-token module (deferred — large mechanical pass, better scoped as its own change).
- Splitting `DMView.tsx` into multiple files (deferred — better done after the `EntityRow`/`PlayerView` duplication is resolved).
- Changing any endpoint's validation strictness or error-handling behavior (deferred where flagged — those are product decisions, not refactors).

## Decisions

**Backend handler split mirrors `store/`'s existing domain boundaries.** `store/` already has `room.go`, `monster.go`, `party.go`, `encounter.go`, `user.go`, `membership.go`, `custom_monster.go`. Splitting `api/handler.go` along the same lines (rather than inventing a new grouping) keeps the two layers' file structure legible together and requires no judgment calls about where a handler belongs.

**Shared request helpers take the narrowest useful shape**: `requireUser(w, r) (userID string, ok bool)`, `decodeJSON(w, r, &body) bool`, `writeJSON(w, status, v)`. Each wraps exactly the boilerplate that's byte-identical across call sites today (auth-required 401, decode-or-400, encode-with-status), and nothing more — handlers keep their own validation and store calls inline. Alternative considered: a larger per-domain "controller" abstraction; rejected as speculative — the actual duplication is in these three small operations, not in overall handler structure.

**ID generation helper preserves per-call-site byte length as a parameter** (`NewID(n int) string`), since `room/room.go`'s `newToken` is already parameterized and the other two call sites both use 8 bytes — a fixed-length helper would silently constrain a caller that later needs a different length.

**Edition-validation helper takes an explicit behavior parameter, not a single unified function.** `CreateRoom` defaulting invalid/omitted edition to `"5e"` is spec'd (`openspec/specs/room-creation/spec.md`); every other endpoint rejects with 400. A single helper that changed either behavior to match the other would be a spec violation. The shared helper is `resolveEdition(raw string, onInvalid EditionPolicy) (string, error)` (or equivalent), called with each site's existing policy.

**Frontend `fetchJSON` helper only wraps the fetch/parse/fallback mechanics, not error surfacing.** Call sites currently diverge on whether a failed fetch silently falls back to `[]`/`null` or sets a visible error message. The shared helper returns a result the caller still branches on, rather than baking in one error-handling policy — otherwise "consolidating" would silently change behavior at whichever call sites don't already match the helper's chosen policy.

**Label-style consolidation picks the more common value where sizes differ (e.g. 12px over 11px), not an average or a new value.** This is called out in the proposal as a minor visual normalization rather than a pure no-op, so the rule for resolving drift is: match the value used by the majority of the 6 sites, don't introduce a new one.

**Sequencing**: backend items (handler split, then shared helpers, then ID/edition consolidation, then `gofmt`) and frontend items (finish pill dedup, then `EntityRow`/`PlayerView` extraction, then `fetchJSON`, then segmented-toggle, then label-style) are independent of each other and can be done in either order or in parallel; within each stack, later items are ordered after the file-structure changes they'd otherwise conflict with (e.g. the handler split happens before introducing shared helpers, so the helpers land in their final per-domain files directly).

## Risks / Trade-offs

- [Mechanical extraction accidentally changes behavior at one call site (e.g. a handler that looks identical but has one subtly different validation branch)] → Diff each extracted call site against its original inline code line-by-line before deleting the original; run existing lint/type-check/build after every item, not just at the end.
- [Consolidating `labelStyle`/`fieldStyle` picks the "wrong" majority value for a site that actually needed its distinct value for a reason not visible in the code] → Screenshot the affected forms before/after (mirrors the verification approach used for the last frontend change) to confirm the 1px/width difference isn't user-visible in practice.
- [Backend handler split is a large mechanical diff that's easy to skim past in review] → Keep it as its own commit with zero logic changes, separate from the helper-introduction commits, so reviewers can diff-review the move alone.

## Migration Plan

No deployment or data migration — internal-only refactor. Land backend and frontend items as separate commits per item (matching this repo's existing convention of small, single-purpose commits), verify each with the relevant lint/type-check/build command, and defer the three larger items in the proposal's "Deferred" section to future changes.

## Open Questions

None outstanding — each item's scope and behavior-preservation constraints are settled above; the deferred items are explicitly out of scope pending separate decisions.
