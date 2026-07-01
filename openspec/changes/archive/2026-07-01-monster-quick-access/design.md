## Context

Both consumers already know the current edition: `AddCreatureForm` receives `edition` as a prop from `roomState.edition`; `EncounterForm` holds `edition` as local state (the DM's own selector). Both already have a monster-selection path from search (`selectMonster` in `AddCreatureForm`, `addMonster` in `EncounterForm`) that this change parallels rather than replaces.

The key asymmetry driving this design: `GET /api/custom-monsters` (unlike a Typesense search hit) already returns the *full* document — `source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`, everything `selectMonster`'s follow-up `GET /api/custom-monsters/:id` fetch exists purely to obtain. Picking from the quick-access list is therefore strictly simpler than picking from search, not just an alternate entry point to the same code path.

## Goals / Non-Goals

**Goals:**
- A DM can add one of their own custom monsters to the in-room tracker or an encounter blueprint in one click, no typing required.
- The quick-pick list only ever shows the current edition's monsters, matching what search already filters to.
- Existing search-based selection is untouched — this is a parallel path, not a replacement.

**Non-Goals:**
- No change to how official (non-custom) monsters are found — this feature is custom-monster-only, since the DM only "owns" custom monsters.
- No caching/invalidation logic — the list is fetched fresh each time the form mounts (`AddCreatureForm`) or the edition changes (`EncounterForm`), matching how the Dashboard's existing "My Monsters" fetch behaves (fetch on mount, no refetch triggers).
- No visual redesign of either form — the new section is additive, styled consistently with the existing search dropdown.

## Decisions

**Server-side edition filter, not client-side.** `ListCustomMonstersByOwner(ownerID, edition string)` filters at the Mongo query level (`bson.M{"owner_id": ownerID, "edition": edition}` when non-empty), matching the precedent just set by `ListEncountersByOwner` in the previous change. The parameter is optional and additive — `ListMyCustomMonsters` reads `r.URL.Query().Get("edition")`; an empty string means "no filter," preserving the Dashboard's existing all-editions behavior with zero changes to that call site.

**Quick-pick selection populates state directly, no fetch.** `selectMonster` (search path) needs a follow-up fetch because a `MonsterSearchHit` is a lean projection. A `CustomMonster` list item is already the full document, so the new handler (`selectCustomMonster` in `AddCreatureForm`, `addCustomMonster` in `EncounterForm`) sets the same target state (`name`, `maxHP`, `monsterRef` / staged group) directly from the already-fetched object. This is fewer round-trips than the search path, not equivalent to it.

**List fetched once per form-mount / edition-change, not per keystroke.** Unlike the debounced search (which queries on every qualifying keystroke), the quick-pick list has no query input — it's a `useEffect` keyed on `edition` (and, for `AddCreatureForm`, mount, since a room's edition is fixed for that component's lifetime).

**Section placement: above the existing search box in both forms**, styled as a horizontally-wrapping chip list (name + HP), not a dropdown — it's meant to be glanceable without typing, unlike the search results dropdown which only appears after input.

## Risks / Trade-offs

- **[Risk]** A DM with many custom monsters could get an unwieldy wall of chips. → **Mitigation**: accepted for this change; the existing "My Monsters" Dashboard list has the same unbounded-growth characteristic and hasn't needed pagination — if it becomes a real problem, a future change can add search-within-quick-pick or pagination then.
- **[Risk]** `ListCustomMonstersByOwner`'s signature change (`ownerID` → `ownerID, edition`) touches every call site. → **Mitigation**: there is exactly one existing call site (`ListMyCustomMonsters`), so this is a one-line update, not a broad refactor.

## Open Questions

None outstanding.
