## Why

`CreateRoom` (`api/rooms.go`) currently swallows a malformed-JSON request body (`json.NewDecoder(r.Body).Decode(&body)` with the error ignored) and proceeds with a zero-value body, which happens to resolve to the spec'd `"5e"` default edition (`openspec/specs/room-creation/spec.md`). Every other handler in `api/` returns HTTP 400 `"invalid json"` on a decode failure, via the shared `decodeJSON` helper (`api/helpers.go`). This inconsistency was flagged during the `code-cleanup` change and deliberately left as a product decision rather than folded into that refactor.

## What Changes

- `CreateRoom` rejects a malformed-JSON request body with HTTP 400 `"invalid json"`, matching every other endpoint, instead of silently falling back to a zero-value body.
- `CreateRoom` switches to the shared `decodeJSON` helper for this, removing its `//nolint` decode-error suppression.
- An omitted `edition` field (valid JSON, just no `edition` key, e.g. `{}` or an empty body) continues to default to `"5e"` — this proposal only changes behavior for a body that fails to parse as JSON at all, not the existing "omitted or invalid edition value" leniency.

## Capabilities

### Modified Capabilities
- `room-creation`: `CreateRoom`'s handling of a malformed (non-JSON) request body changes from silently defaulting to `"5e"` to rejecting with HTTP 400 `"invalid json"`.

## Impact

- `api/rooms.go`'s `CreateRoom` handler.
- `openspec/specs/room-creation/spec.md`.
- Closes GitHub issue #8.
- No data model changes. No change to the "omitted/invalid edition value in an otherwise-valid body" behavior (still defaults to `"5e"`).
