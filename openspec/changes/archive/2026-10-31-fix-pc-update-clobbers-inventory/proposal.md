## Why

`PUT /api/pcs/{id}` overwrites `items`, `currency`, and `shares_initiative` with empty values whenever the request omits them, and the Edit Character screen sends only `name` and `max_hp`. Editing a character's name or HP therefore silently erases its inventory and purse.

## What Changes

- `PUT /api/pcs/{id}` updates `items`, `currency`, and `shares_initiative` only when the request includes them; an omitted field keeps its stored value.
- An explicit empty `items` array still clears the list, so `InventoryPanel` can remove every item.
- `name` and `max_hp` stay required on every update.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `player-profile-management`: updating a PC or companion changes only the optional fields present in the request, rather than overwriting all editable fields.

## Impact

- `store/mongo.go`: `PCUpdateFields` distinguishes omitted from present fields; `UpdatePC` builds its `$set` from the fields present.
- `api/pcs.go`: `pcPayload` passes omitted fields through as absent.
- `api/pcstore_fake_test.go`, `api/pcs_handlers_test.go`, `store/pc_integration_test.go`: the fake mirrors the new semantics; regression tests added.
- No frontend change: existing callers keep working, and `CharacterForm` stops erasing data without being edited.
- Inventories already erased by this bug are not recoverable through this change.
