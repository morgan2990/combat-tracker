## Why

`CreateRoom` (`api/handler.go`) currently swallows a malformed-JSON request body (`json.NewDecoder(r.Body).Decode(&body)` with the error ignored) and proceeds with a zero-value body, which happens to resolve to the spec'd `"5e"` default edition (`openspec/specs/room-creation/spec.md`). Every other handler in `api/handler.go` returns HTTP 400 `"invalid json"` on a decode failure. This is scoped separately from `code-cleanup` because changing it would alter `CreateRoom`'s observable behavior for malformed request bodies — a product decision, not a pure refactor.

## What Changes

This proposal does not pre-decide the outcome. When picked up, it should:
- Confirm whether `CreateRoom`'s current leniency (accepting a malformed body and falling back to defaults) is intentional or an oversight.
- If it should be tightened to match sibling handlers (reject malformed JSON with 400), update `room-creation`'s spec accordingly — this would be a **Modified Capability**, not an internal-only refactor, since it changes an observable response for malformed requests.
- If the leniency is intentional (e.g. to keep room creation maximally permissive for clients), no code change is needed — document the decision so it isn't re-flagged as an inconsistency in a future cleanup pass.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `room-creation`: possibly modified, pending the decision above — `CreateRoom`'s handling of a malformed (non-JSON) request body may change from silently defaulting to rejecting with HTTP 400, to match every other handler's behavior.

## Impact

- `api/handler.go`'s `CreateRoom` handler, if changed.
- `openspec/specs/room-creation/spec.md`, if the behavior changes.
- No data model changes.
