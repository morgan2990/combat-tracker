## No Capability Changes

This change is a pure internal refactor (see `proposal.md`'s Capabilities section): no new capabilities, no requirement-level behavior changes to any existing capability. There are no delta spec files because there is nothing to add, modify, remove, or rename in `openspec/specs/`.

Anything in the codebase survey that would have required a behavior change (and therefore a delta spec) — e.g. tightening `CreateRoom`'s malformed-JSON handling — is explicitly listed under `proposal.md`'s "Deferred" section instead of being folded into this change.
