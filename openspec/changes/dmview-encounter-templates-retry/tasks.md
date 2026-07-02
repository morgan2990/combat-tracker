## 1. Fix retry behavior

- [ ] 1.1 In `frontend/src/components/EncounterTemplatesControl.tsx`'s `toggleOpen`, restructure the fetch chain so `setLoaded(true)` only fires when the response is ok and the body parses successfully — a non-ok response and a network/parse error both leave `loaded` false.
- [ ] 1.2 Run `tsc -b` and `oxlint` to confirm no regressions.

## 2. Verify

- [ ] 2.1 Manually test: with the backend running normally, open the dropdown, confirm the encounter list loads and a second open doesn't re-fetch (e.g. via network tab / no visible reload flicker).
- [ ] 2.2 Manually test (simulated failure): temporarily point the fetch at a bad URL or stop the backend, open the dropdown, confirm the empty-state message shows; restart the backend, open again, confirm it retries and succeeds this time.
- [ ] 2.3 Close GitHub issue #7 referencing the merged change.
