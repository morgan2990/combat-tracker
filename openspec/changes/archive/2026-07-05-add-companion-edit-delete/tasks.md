## 1. Store layer (`store/mongo.go`)

- [x] 1.1 Add `sharesInitiative bool` parameter to `UpdatePC`, and include `shares_initiative` in the `$set` document.
- [x] 1.2 Add `DeletePC(id string) (deletedType string, err error)` (or equivalent): fetch the document first to read its `type`; if `type == "pc"`, delete all companions where `parent_pc_id == id` (e.g. `DeleteMany`) before/alongside deleting the document itself; if `type == "companion"`, delete just that one document. Return enough information (or a not-found sentinel) for the handler to distinguish "not found" from "deleted".

## 2. API layer (`api/pcs.go`, `main.go`)

- [x] 2.1 Update `UpdatePC` handler to pass `body.SharesInitiative` through to the store call added in 1.1.
- [x] 2.2 Add a `DeletePC` handler: resolve the document by `id`, 404 if missing, 403 if `OwnerUserID != userID`, otherwise call the store's delete (1.2) and return 200/204.
- [x] 2.3 Register `DELETE /api/pcs/{id}` in `main.go` alongside the existing `POST/PUT/GET /api/pcs...` routes.

## 3. Room sync (`room/room.go`)

- [x] 3.1 In `RefreshFromProfile`, alongside the existing `MaxHP` sync for matched companion entities (matched via `e.PCID`), also set `e.Name` and `e.SharesInitiative` from the fetched companion document.

## 4. Frontend — companion edit/delete (`frontend/src/components/CharacterForm.tsx`)

- [x] 4.1 Replace the read-only `existingCompanions.map(...)` block with editable rows (name input, max HP input, shares-initiative checkbox), mirroring the styling already used for new-companion rows.
- [x] 4.2 Wire each existing-companion row's edits to `PUT /api/pcs/{companion_id}` with `{ name, max_hp, shares_initiative }` (decide: save-on-blur/per-field, or an explicit per-row Save button — pick whichever matches the rest of this form's interaction feel). Implemented as an explicit per-row Save button (draft state per companion, committed on click).
- [x] 4.3 Add a per-row Delete button calling `DELETE /api/pcs/{companion_id}`; on success, remove the row from `existingCompanions` locally (optimistic update, matching the pattern used for `handleDeleteMonster`/`handleDeleteEncounter`/`handleDeleteRoom` elsewhere in this codebase).

## 5. Frontend — character delete (`frontend/src/components/Dashboard.tsx`)

- [x] 5.1 Add a `handleDeleteCharacter(pcId)` function: `window.confirm` guard with copy warning that deleting a character currently active in a room means that room can no longer be rejoined, then `DELETE /api/pcs/{pcId}`.
- [x] 5.2 On success, remove the character from the rendered "My Characters" list (optimistic update, same pattern as `handleDeleteRoom`).
- [x] 5.3 Add the Delete action next to the existing Edit link in the "My Characters" row markup (`Dashboard.tsx` around line 295).

## 6. Verification

- [x] 6.1 Manually verify: edit a companion's name/HP/shares-initiative, confirm it persists via `GET /api/pcs/{id}`.
- [x] 6.2 Manually verify: delete a companion, confirm it's gone from `GET /api/pcs/{parent_id}`'s `companions` array.
- [x] 6.3 Manually verify: delete a character with companions, confirm both the PC and its companions are gone (check the `pcs` collection or re-fetch `GET /api/me`).
- [x] 6.4 Manually verify: join a room, edit a companion's name on the profile, send `refresh_from_profile`, confirm the live entity's name updates in the broadcast `RoomState`.
- [x] 6.5 Manually verify ownership checks: attempting `PUT`/`DELETE /api/pcs/{id}` as a non-owner returns 403.
