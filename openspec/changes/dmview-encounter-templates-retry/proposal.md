## Why

`EncounterTemplatesControl`'s dropdown (`frontend/src/components/EncounterTemplatesControl.tsx`, extracted from `DMView.tsx` during `split-dmview-component`) fetches `GET /api/encounters?edition=<edition>` on first open. On a non-ok HTTP response it marks the list as loaded (so a later open never retries, even though the empty list reflects a fetch problem, not the DM's actual saved encounters), while on a network error it does not mark it loaded (so a later open does retry). This asymmetry was flagged during `code-cleanup` and tracked as GitHub issue #7, deliberately left as a product decision rather than folded into either the extraction or the shared `fetchJSON` consolidation.

## What Changes

- `EncounterTemplatesControl` retries the fetch on every open until it has genuinely succeeded at least once — a non-ok HTTP response and a network error are both treated the same way (list stays "not loaded", next open retries).
- Once a fetch actually succeeds (parses a JSON array, whether empty or not), the list is marked loaded and won't re-fetch on subsequent opens, as today.
- Closes GitHub issue #7.

## Capabilities

### Modified Capabilities
- `encounter-injection`: the "DM Panel Encounter Templates Dropdown" requirement gains a scenario specifying retry-on-failure behavior for the phone-tier dropdown's fetch, previously unspecified.

## Impact

- `frontend/src/components/EncounterTemplatesControl.tsx` only.
- No API or data model changes. No change to the successful-fetch path (still lists the DM's saved encounters for the room's edition).
