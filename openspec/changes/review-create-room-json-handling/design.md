## Context

`CreateRoom` (`api/rooms.go`) is the one endpoint that doesn't use the shared `decodeJSON` helper introduced during `code-cleanup` — it ignores the JSON decode error entirely (`//nolint`) and proceeds with a zero-value body, which happens to resolve to the spec'd `"5e"` default edition. Every other handler in `api/` rejects a malformed body with HTTP 400 `"invalid json"`. This was a deliberate product decision, now made: tighten `CreateRoom` to match.

## Goals / Non-Goals

**Goals:**
- `CreateRoom` rejects a malformed (non-JSON) request body with HTTP 400 `"invalid json"`, using the same shared `decodeJSON` helper every other handler uses.

**Non-Goals:**
- No change to the existing "omitted or invalid `edition` value, in an otherwise-valid JSON body" leniency — that still resolves to `"5e"` via `resolveEditionOrDefault`. This proposal is scoped only to bodies that fail to parse as JSON at all.
- No change to any other endpoint's decode behavior.

## Decisions

**`CreateRoom` keeps its own decode step rather than reusing the shared `decodeJSON` helper.** `decodeJSON` treats any decode error — including `io.EOF` from a genuinely empty body — as "invalid json" and rejects with 400. But `room-creation`'s spec explicitly requires an empty body to succeed (`edition` defaults to `"5e"`), a contract no other endpoint has. Reusing `decodeJSON` as-is would have silently broken that spec'd scenario. `CreateRoom`'s decode step treats `io.EOF` as "valid, empty body" and only rejects on other decode errors (i.e. an actually malformed, non-empty body) — verified against all four scenarios (well-formed body, empty body, unrecognized edition value, malformed JSON) via manual `curl` requests.

**Keep `resolveEditionOrDefault` unchanged.** The decode step and the edition-resolution step are separate concerns — decode failure is now a hard 400, while a valid-but-unrecognized `edition` value (or an omitted one) within a successfully-decoded body still defaults to `"5e"`, per the existing spec'd behavior. This proposal only removes the silent-swallow of decode errors.

## Risks / Trade-offs

- [Any existing client that currently sends a malformed body to `POST /api/rooms` and relies on the current default-to-"5e" leniency would start getting a 400] → No known client does this (the DM Dashboard always sends a well-formed `{ "edition": ... }` body); this is a narrowing of previously-undefined behavior, not a documented contract being broken.

## Migration Plan

Frontend-only-unaffected backend change, no data migration. Single commit: swap `CreateRoom` to `decodeJSON`, update `room-creation`'s spec, close GitHub issue #8.

## Open Questions

None — the decision to tighten (vs. keep lenient) was made explicitly before this design was written.
