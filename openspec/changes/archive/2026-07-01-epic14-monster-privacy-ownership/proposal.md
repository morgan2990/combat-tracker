## Why

Epic 14 asks for private, owner-scoped custom monsters so a DM's campaign secrets stay hidden from other DMs. Today every monster — official (scrubber-imported) and DM-authored custom alike — lives in one MongoDB collection keyed only by `{name, edition}`, with no owner in the write key at all. Simply bolting `private`/`owner_id` fields onto that model (the epic's literal AC list) would not actually deliver privacy safely: two different DMs creating custom monsters with the same name and edition would silently overwrite each other via the existing upsert path, and "customizing an official monster" already means overwriting the one shared document every DM sees. Investigation surfaced this as the real problem to solve, not just the two fields named in the epic.

## What Changes

- **BREAKING**: Split monster storage into two MongoDB collections — `monsters` (official, scrubber-only, unchanged `{name, edition}` key) and `custom_monsters` (DM-authored, ID-keyed like `PC` documents, `owner_id` required). This removes the cross-DM name-collision risk structurally instead of patching around it, and retires the `shouldPreserveCustom` guard (no longer reachable — scrubber and DMs never contend for the same document anymore).
- Add `private`, `owner_id`, and `owner_display_name` to custom monster documents and to the Typesense mirror, alongside `is_custom` (currently missing from the Typesense schema entirely).
- `GET /api/search/monsters` now requires authentication and applies a composite filter so a DM only ever sees: official monsters, public custom monsters, and their own private custom monsters. Search results merge official and custom hits into one list, distinguished by `is_custom` and `owner_display_name` (e.g. "Goblin — Official" vs. "Goblin — by Alice") so a DM can tell which one they want when names collide.
- New ID-keyed endpoints for custom monsters: get-by-id (403 if private and not owned by requester), update-by-id (edit), delete-by-id (removes from MongoDB and the Typesense mirror), and PDF streaming by id with the same ownership check. The existing name-keyed endpoints (`GET /api/monsters/:name`, `GET /api/monsters/:name/pdf`) remain but now only resolve official monsters.
- `MonsterForm.tsx` gains a "Mark as Private Campaign Content" toggle with an explanatory tooltip, and an edit mode (mirroring `CharacterForm.tsx`'s `useParams`-based pattern) so an existing custom monster's privacy state loads correctly for editing.
- Dashboard's "As DM" panel gains a "My Monsters" list, scoped to the requesting DM's own custom monsters, each with **Edit** and **Delete** actions. Delete requires a confirm step — the app's first delete-of-persisted-data operation.
- Typesense collection is wiped and recreated once (no data worth preserving pre-launch) to pick up the new schema fields; the scrubber is re-run afterward to backfill official monsters, per the existing "re-running the scrubber fully backfills the index" behavior.

Explicitly not changing: `Entity.display_name` / per-room-instance aliasing (Epic 16, separate and orthogonal), any migration tooling for pre-existing custom monster data (confirmed disposable), and Typesense schema-alter code (wipe-and-recreate is sufficient with no data to preserve).

## Capabilities

### New Capabilities
_None._ Everything below extends the purpose of existing capabilities rather than introducing a new domain.

### Modified Capabilities
- `monster-repository`: the DM-authored monster path moves from `{name, edition}` upsert to ID-keyed CRUD in a separate `custom_monsters` collection, with `owner_id`/`private`/`owner_display_name`; gains update-by-id, delete-by-id, get-by-id, and PDF-stream-by-id; name-keyed lookup/stream requirements narrow to official monsters only.
- `monster-search`: search now requires authentication, applies the owner/privacy filter, and returns a richer hit shape (`is_custom`, `owner_display_name`); the DM panel's follow-up statblock fetch routes to the ID-keyed endpoint for custom hits instead of always fetching by name.
- `monster-search-index`: Typesense schema gains `is_custom`, `private`, `owner_id`, `owner_display_name`; both `monsters` and `custom_monsters` mirror into the same index; deleting a custom monster removes its Typesense document too.
- `monster-form`: adds the private/public toggle with tooltip and an edit mode that loads an existing custom monster's saved privacy state.
- `dashboard-monster-creation`: adds an owner-scoped "My Monsters" list with Edit and Delete (confirm-gated) actions alongside the existing entry point and post-save behavior.

## Impact

- **Backend**: `store/monster.go` (collection split, drop `shouldPreserveCustom`, new CRUD functions), `store/typesense.go` (schema + mirror fields, delete-from-index), `api/handler.go` (auth on search, ID-keyed routes, ownership checks), `main.go` (new route registrations).
- **Frontend**: `frontend/src/components/MonsterForm.tsx` (privacy toggle, edit mode), `frontend/src/components/Dashboard.tsx` (My Monsters list), `frontend/src/components/DMView.tsx` (statblock-preview fetch routes by id for custom hits), `frontend/src/App.tsx` (new `/monsters/custom/:id/edit` route), `frontend/src/types.ts` (hit/monster shape updates).
- **Ops**: one-time manual drop of the Typesense `monsters` collection before/at deploy, followed by a scrubber re-run to backfill official monsters.
