## Context

Today every monster document — official (scrubber-imported) and DM-authored custom — lives in one MongoDB `monsters` collection, identified only by `{name, edition}`. `UpsertMonster` (`store/monster.go`) writes via `FindOneAndReplace` with upsert enabled, keyed on that pair. The only collision guard, `shouldPreserveCustom`, stops a non-custom (scrubber) write from clobbering an existing custom document — it says nothing about two different DMs both writing custom documents with the same name and edition, which today silently overwrite each other. "Customizing an official monster" and "creating a private monster" are the same code path writing to the same document identity.

Typesense mirrors this collection into a single search index, but the mirrored document (`typesenseMonsterDoc`, `store/typesense.go`) doesn't even carry `is_custom` today, let alone ownership fields. `ensureMonsterCollection` only creates the collection if missing — it does not migrate an existing collection's schema.

This is early-stage development (per Epic 13 archived just before this change): there is no production data that needs to survive a storage-model change, and no existing "edit a saved monster" UI to preserve compatibility with — `MonsterForm.tsx` is create-only.

## Goals / Non-Goals

**Goals:**
- Make privacy actually safe: a DM's private monster must never be exposed to, or overwritten by, another DM.
- Eliminate the cross-DM name collision at the source (structural fix), not with a conflict-rejection patch.
- Give custom monsters a real edit/delete lifecycle, since Epic 14's own AC (privacy state must load correctly when editing) presupposes one.

**Non-Goals:**
- Preserving pre-existing custom monster or Typesense data through this change (confirmed disposable).
- Building Typesense schema-alter/migration tooling.
- Touching `Entity.display_name` / per-room aliasing (Epic 16's concern, unrelated to monster document identity).
- Building any "official monster override" or lineage-tracking concept — a DM customizing an official monster always produces an independent custom document with no link back to the official one.

## Decisions

**Split into two MongoDB collections: `monsters` (official) and `custom_monsters` (DM-authored, ID-keyed).**
This was chosen over two alternatives considered during exploration:
- *Fold `owner_id` into the existing collection's upsert key* (`{name, edition, owner_id}`) — works but leaves scrubber and DM writes sharing one collection and one code path, and raises an unresolved question of whether a DM's override should "shadow" the official document in search results.
- *Keep one collection, reject conflicting writes* (first DM to use a name "claims" it, second gets an error) — cheaper, but produces a confusing wall for common homebrew names and doesn't allow two DMs to independently have their own "Bandit Leader."

The two-collection split avoids both problems: custom monsters are identified by MongoDB `_id` (the same pattern `PC` documents already use — `GetPCByID`), so two DMs' same-named monsters are simply two different documents. `shouldPreserveCustom` becomes unreachable dead code and is removed, since the scrubber can now only ever write to `monsters` and DMs can now only ever write to `custom_monsters`.

**Single merged Typesense index across both collections.**
Considered mirroring into two separate Typesense collections and merging client-side or server-side per query. Rejected: a DM's search should show official and custom hits interleaved by relevance in one request, and the existing search code is already built around one collection — adding `is_custom`/`owner_id`/`owner_display_name`/`private` fields to the one existing schema is far less churn than standing up a second collection and a merge step.

**Wipe-and-recreate the Typesense collection instead of writing schema-alter logic.**
`ensureMonsterCollection`'s create-if-missing check means it will not pick up new fields on an already-existing collection. Since there's no data worth preserving, the plan is: drop the collection once (ops step), let the existing create-if-missing code recreate it with the new schema on next boot, then re-run the scrubber to backfill official monsters — reusing the already-documented "re-running the scrubber fully backfills the index" behavior (`monster-search-index` spec) instead of writing new migration code.

**Custom monster identity is a new capability, not a repurposed name-based one.**
`GET /api/monsters/:name` and `GET /api/monsters/:name/pdf` remain but resolve official monsters only going forward. New ID-keyed siblings (`GET /api/custom-monsters/:id`, `GET /api/custom-monsters/:id/pdf`, `PUT`/`DELETE /api/custom-monsters/:id`) handle the custom side, each enforcing: 403 if `private` and `owner_id` doesn't match the authenticated requester.

These routes live under `/api/custom-monsters`, a distinct top-level path from `/api/monsters/{name}`, rather than nested under `/api/monsters/custom/...` as first implemented. Discovered when the server failed to start on deploy: Go's `net/http.ServeMux` rejects `GET /api/monsters/{name}/pdf` and `GET /api/monsters/custom/{id}` as conflicting patterns at registration time, since both have 3 path segments and the mux can't statically prove a request like `/api/monsters/custom/pdf` resolves unambiguously (is `custom` the `{name}` with a literal `/pdf` suffix, or is `custom` literal with `pdf` as the `{id}`?). Moving custom-monster routes to their own top-level prefix removes the structural overlap entirely.

**Delete requires a confirm step; no soft-delete.**
This is the app's first delete-of-persisted-data operation (PCs and rooms are never deleted; the closest analog — removing an unsaved companion row in `CharacterForm` — discards client-side state before it's ever saved, not a real delete call). A simple confirm-before-delete interaction was chosen over a full modal/undo system to stay consistent with the app's generally lightweight UI. Delete is hard: it removes the MongoDB document and best-effort removes the Typesense mirror (same log-don't-fail pattern `syncMonsterToTypesense` already uses), so a deleted monster stops appearing in search immediately in the common case, with the same best-effort caveat the rest of the sync layer already accepts.

**Custom monster PDFs are keyed by MongoDB id in MinIO, not by name.**
Discovered during implementation: `store.UploadPDF`/`StreamPDF` key objects purely by monster name (`"monsters/" + name + ".pdf"`), globally, with no owner scoping — the same collision class as the original MongoDB problem, just in MinIO, and missed during exploration. Fixed by adding id-keyed variants (`custom-monsters/{id}.pdf`) used only by the new custom-monster endpoints; the official path stays name-keyed and untouched since official names remain globally unique. Since the MongoDB `id` doesn't exist until the document is inserted but the PDF needs to be uploaded (and its key known) before the document can be written with `pdf_object_key` set, the id is now pre-generated by the handler (via an exported `store.NewID()`) before the PDF upload, then passed into `CreateCustomMonster` instead of being generated inside it.

## Risks / Trade-offs

- **[Risk]** Dropping the Typesense collection loses all currently-indexed monsters (official and custom) until the scrubber is re-run. → **Mitigation**: sequence the ops steps as drop → deploy → restart (recreates schema) → re-run scrubber, done as one maintenance window; acceptable given no production traffic yet.
- **[Risk]** `POST /api/monsters` currently serves DM-authored creation; splitting it into a distinct custom-monster path changes its request/response contract (id semantics, collection target). → **Mitigation**: this is a pre-launch app with one frontend client under our control (`MonsterForm.tsx`), updated in the same change — no external consumers to break.
- **[Risk]** Best-effort Typesense delete-on-delete can fail silently (network blip), leaving a stale, deleted-in-Mongo document visible in search until the next write to that id. → **Mitigation**: same accepted trade-off the existing sync layer already makes for upserts; log the failure per existing convention.

## Migration Plan

1. Ship code: collection split, new fields, new routes, frontend changes (this change's `tasks.md`).
2. Ops (one-time, at/before deploy): drop the existing Typesense `monsters` collection.
3. Deploy / restart the server — `ensureMonsterCollection` recreates the collection fresh with the new schema (create-if-missing path, unchanged code).
4. Re-run the scrubber against the existing source/edition to backfill official monsters into the recreated index.
5. Any custom monsters created before this change shipped are gone (both the old single-collection Mongo documents are superseded by the new `custom_monsters` collection, and the Typesense mirror was wiped) — confirmed acceptable; affected DMs recreate them via the (now privacy-aware) form.

No rollback path is defined beyond reverting the deploy — acceptable given the confirmed lack of production data at stake.

## Open Questions

None outstanding — all decisions below were resolved during exploration prior to this proposal.
