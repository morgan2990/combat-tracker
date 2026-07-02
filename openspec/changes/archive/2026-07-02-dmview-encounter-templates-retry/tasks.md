## 1. Fix retry behavior

- [x] 1.1 In `frontend/src/components/EncounterTemplatesControl.tsx`'s `toggleOpen`, restructured the fetch chain so `setLoaded(true)` only fires when the response is ok and the body parses successfully — a non-ok response resolves to `undefined` (instead of `[]`) so it's handled the same way as the `.catch()` network-error path, both leaving `loaded` false.
- [x] 1.2 `tsc -b` and `oxlint` clean (same pre-existing warnings as before, no new ones).

## 2. Verify

- [x] 2.1 Verified via Playwright with route interception: a successful fetch lists the encounter and a second dropdown open makes no new request (call count stayed at 1).
- [x] 2.2 Verified via Playwright, both failure modes: (a) forced HTTP 500 on first call → empty-state shown, reopening retries and succeeds (2 calls total); (b) simulated network abort on first call → same result, retries and succeeds on reopen (2 calls total).
- [x] 2.3 GitHub issue #7 auto-closed on merge, referencing merge commit `98b08d47e7f6aee3c2701e7572757811ae112508` (PR #13).
