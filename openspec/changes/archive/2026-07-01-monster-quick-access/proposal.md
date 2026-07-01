## Why

A DM with several custom monsters currently has to type 3+ characters into search every time they want to add one they've already created, even though the app already knows exactly which monsters belong to them. Surfacing the DM's own custom monsters as a one-click list — alongside, not instead of, search — removes that friction in both places a DM picks a monster: the in-room Add Creature form and the Encounter Builder.

## What Changes

- Add an optional `edition` query parameter to `GET /api/custom-monsters`, filtering the returned list to the requester's own custom monsters of that edition. Existing callers (the Dashboard's "My Monsters" list) that omit the parameter are unaffected — they still get every edition.
- Add a "My Creatures" quick-pick section to the DM Panel's Add Creature form (`AddCreatureForm` in `DMView.tsx`), listing the DM's own custom monsters for the room's current edition. Clicking one populates the form exactly as clicking a search result does, but without a network round-trip for the full monster doc — the list response already carries every field (`source_type`, `reference_url`, `pdf_object_key`, `initiative_modifier`) that `selectMonster`'s follow-up fetch exists solely to obtain for search hits.
- Add the same "My Creatures" quick-pick section to the Encounter Builder screen (`EncounterForm.tsx`), appending a staged monster group on click exactly as picking a search result does.
- Both quick-pick lists are populated by the same edition-filtered `GET /api/custom-monsters?edition=` fetch and stay independent of the search input — a DM can use either, or both, interchangeably.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `monster-repository`: "DM can list their own custom monster templates" gains an optional `edition` filter.
- `initiative-ui`: the Add Creature Form gains a "My Creatures" quick-pick section alongside its existing search.
- `encounter-builder`: the Encounter Builder Screen gains the same quick-pick section alongside its existing search.

## Impact

- `api/handler.go`: `ListMyCustomMonsters` reads an optional `edition` query param.
- `store/custom_monster.go`: `ListCustomMonstersByOwner` gains an edition filter parameter.
- `frontend/src/components/DMView.tsx`: `AddCreatureForm` fetches and renders the DM's custom monsters for the room's edition; a new selection handler populates form state directly from the already-fetched data (no follow-up fetch, unlike `selectMonster`).
- `frontend/src/components/EncounterForm.tsx`: same quick-pick section and direct-populate selection handler, appending a staged group.
- No changes to `monster-search`, Typesense, or the WS protocol — this is purely an alternate, faster path to data the app already exposes.
