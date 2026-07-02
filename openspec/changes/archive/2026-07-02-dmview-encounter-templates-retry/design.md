## Context

`EncounterTemplatesControl.tsx`'s `toggleOpen` currently has two different failure paths that behave differently:

```ts
fetch(`/api/encounters?edition=${encodeURIComponent(edition)}`)
  .then(res => res.ok ? res.json() : [])
  .then((data: Encounter[]) => { setEncounters(data); setLoaded(true) })
  .catch(() => setEncounters([]))
```

A non-ok HTTP response resolves to `[]` in the first `.then` and flows into the second `.then`, which sets both `encounters` and `loaded`. A network/parse error skips straight to `.catch`, which only sets `encounters`, leaving `loaded` false. The decision made: always retry on either failure mode, only marking `loaded` true on a genuine success.

## Goals / Non-Goals

**Goals:**
- Both failure modes (non-ok response, network/parse error) leave `loaded` false, so the next dropdown open retries the fetch.
- A genuinely successful fetch (parses to a JSON array, even an empty one) still marks `loaded` true and is not re-fetched on subsequent opens.

**Non-Goals:**
- No change to what's fetched, how results are displayed, or the injection flow once an encounter is selected.
- Not switching this call site to the shared `fetchJSON` helper — `fetchJSON<T>(url, fallback)` can't distinguish "genuinely succeeded with a value equal to the fallback" from "failed and returned the fallback," which this call site needs to distinguish (an empty-but-successful list must still mark `loaded`). This was the same reason `code-cleanup` left this call site out of the `fetchJSON` consolidation.

## Decisions

**Only call `setLoaded(true)` inside the branch that has real parsed data**, not inside the `res.ok ? ... : []` ok-check. Restructure so a non-ok response short-circuits to the same "leave loaded false" outcome as a network error, rather than synthesizing a fallback array that flows through as if it were a success.

## Risks / Trade-offs

- [A DM who opens the dropdown repeatedly during a genuine outage re-fetches every time] → Acceptable — this is a low-frequency, DM-only, manually-triggered action (not a poll loop), and the alternative (silently stuck on an empty list until the page reloads) is worse UX for a transient blip.

## Migration Plan

Frontend-only, no data migration. Single commit: fix the retry logic, sync the `encounter-injection` delta spec, verify.

## Open Questions

None — the retry-vs-no-retry decision was made explicitly before this design was written.
