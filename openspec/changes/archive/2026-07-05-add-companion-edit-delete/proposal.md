## Why

The Edit Character screen's "Existing Companions / Pets" section is read-only after creation — no way to edit a companion's name, max HP, or shares-initiative flag, and no way to delete one. Separately, there is no way to delete a character (PC) at all: the Dashboard's "My Characters" list only offers an Edit link. Both gaps stem from the persistent PC/companion profile system (`player-profile-management`) having only ever grown create/update/fetch operations, never delete.

## What Changes

- Add `DELETE /api/pcs/{id}`, owner-checked against the document's own `owner_user_id` (works for either a `"pc"` or `"companion"` type document — both are rows in the same `pcs` collection). Deleting a `"pc"`-type document cascades to delete all companion documents whose `parent_pc_id` matches it.
- Extend `UpdatePC`/`PUT /api/pcs/{id}` to also persist `shares_initiative` (currently accepted in the request payload but silently dropped — the store method never writes it).
- Widen the `refresh_from_profile` WebSocket sync (`room.RefreshFromProfile`) to also propagate a companion's `name` and `shares_initiative` into any live room entity, alongside the `max_hp` sync it already does. Still pull-based, still only runs when the client sends `refresh_from_profile` (typically on reconnect) — no push/broadcast-on-edit.
- Delete is unconditional: no check against whether the PC or companion is currently instantiated in an active room. This matches existing delete precedent elsewhere (room deletion, encounter deletion, custom-monster deletion) and keeps the API layer from having to reach into the in-memory room registry.
- Frontend (`CharacterForm.tsx`): the "Existing Companions / Pets" section becomes editable (name, max HP, shares-initiative checkbox) with a per-row Delete action, replacing the current plain read-only list. Edits save via `PUT /api/pcs/{companion_id}` per row; delete via `DELETE /api/pcs/{companion_id}`.
- Frontend (`Dashboard.tsx`): the "My Characters" list gains a Delete action next to the existing Edit link, with a confirm dialog whose copy warns that deleting a character currently active in a room means that room can no longer be rejoined (the server re-fetches the PC on every join/reconnect and rejects with "forbidden" if it's gone).

## Capabilities

### Modified Capabilities
- `player-profile-management`: adds `DELETE /api/pcs/{id}` (with cascade delete of companions when deleting a PC), fixes `shares_initiative` persistence in the existing update path, and widens the `refresh_from_profile` sync to cover `name` and `shares_initiative` in addition to `max_hp`.

## Impact

- `store/mongo.go`: new `DeletePC` method (cascade delete when `type == "pc"`); `UpdatePC` gains a `sharesInitiative` parameter.
- `api/pcs.go`: new `DeletePC` handler; `UpdatePC`/`CreateCompanion` payload handling updated for `shares_initiative` write-through.
- `main.go`: register `DELETE /api/pcs/{id}`.
- `room/room.go`: `RefreshFromProfile` gains `Name`/`SharesInitiative` sync alongside `MaxHP`.
- `frontend/src/components/CharacterForm.tsx`: existing-companions section becomes editable + deletable.
- `frontend/src/components/Dashboard.tsx`: "My Characters" list gains a Delete action with confirm-dialog copy.
