## 0. Prerequisite

- [x] 0.1 Confirm `fix-pc-update-clobbers-inventory` has shipped: `UpdatePC` with only `name` and `max_hp` keeps `items` and `currency`.

## 1. Store

- [x] 1.1 Add `store/character_sheet.go` with the sheet, section, and shared `creature` types, including `schema_version` and the optional `source` block.
- [x] 1.2 Add `CharacterSheetStore` with a unique index on `pc_id`, and register it in `store.Init`.
- [x] 1.3 Implement `GetByPCID`, `GetByPCIDs`, and `DeleteByPCIDs`.
- [x] 1.4 Implement `SaveSection` as a compare-and-set on `{pc_id, revision}`, upserting at `revision: 0` and reporting a conflict on a mismatch or duplicate key.
- [x] 1.5 Add integration tests using the shared testcontainer: round trip, saving one section leaves the others intact, first save, stale revision, concurrent first save, delete, and `source` preserved.

## 2. API for the owner

- [x] 2.1 Add a `sheetStore` interface and a test fake, following `api/pcstore.go`.
- [x] 2.2 Implement `GET /api/pcs/{id}/sheet`, returning the PC, sheet, and companions, with an empty sheet when none is stored and 404 for non-owners.
- [x] 2.3 Implement `PUT /api/pcs/{id}/sheet/{section}`: ownership, section allowed for the document's type, 1 MiB limit, enum and spell level validation, and 409 on a stale revision.
- [x] 2.4 Write `name` from `identity` and `max_hp` from `combat` to the PC through `UpdatePC`, rejecting invalid values before any write.
- [x] 2.5 Delete sheets in `DeletePC`: a PC's and its companions', or only the companion's.
- [x] 2.6 Register the routes in `main.go`.
- [x] 2.7 Add handler tests for every scenario in the owner, validation, and deletion requirements.

## 3. API for the DM

- [x] 3.1 Implement `GET /api/rooms/{room_id}/pcs/{pc_id}/sheet` with `requireOwnedRoom`, `resolveLinkedParty`, and the party membership check, and register the route.
- [x] 3.2 Add handler tests: party member, PC outside the party, room with no party, non-owner of the room, and unauthenticated.

## 4. Sheet page: base and core sections

- [x] 4.1 Add the sheet types to `frontend/src/types.ts`.
- [x] 4.2 Add a data hook that loads through the owner or DM route depending on `?room=`.
- [x] 4.3 Add the `/characters/:id/sheet` route and page, using `CollapsibleSection`, with empty states.
- [x] 4.4 Add the identity, abilities and saves, skills, combat, and defenses views.
- [x] 4.5 Add the sheet link to the Dashboard's character list and to `PlayerView`.
- [x] 4.6 Add the sheet link to party members' rows in `EntityRow`, opening in a new tab with `?room=`.
- [x] 4.7 Add component tests for the page, empty states, read-only mode, and link visibility.

## 5. Sheet page: remaining views

Each group can ship independently once section 4 is done.

- [x] 5.1 Add the attacks, actions, limited uses, and proficiencies views, marking `pending` proficiencies, with tests.
- [x] 5.2 Add the features, feats, weapon masteries, magic items, and notes views, showing each item's related limited uses, with tests.
- [x] 5.3 Add the spellcasting view: lists by source, slots, separate Pact Magic, and a distinct marker per preparation state, with tests.
- [x] 5.4 Add the shared creature component and use it for companions and wild shapes, with tests.

## 6. Manual editing

- [x] 6.1 Add a section editor wrapper that handles edit, cancel, and save, shows validation errors, and on 409 keeps the input and offers a reload. Add a row editor for list sections. Include tests.
- [x] 6.2 Add editing for identity (including `name`), abilities, skills, combat (including `max_hp`), defenses, and notes, with the note that values do not recalculate. Include tests.
- [x] 6.3 Add editing for attacks, actions, limited uses, features, proficiencies, weapon masteries, and magic items, with tests.
- [x] 6.4 Add editing for spellcasting, wild shapes, and the companion creature section, with tests.

## 7. Verification

- [x] 7.1 Run `go test ./...`, and `npm test` and `npm run build` in `frontend/`.
- [x] 7.2 In the running app, enter the Puck Adulto sheet by hand, including its spell lists and Pact Magic, and confirm it survives a reload.
- [x] 7.3 As DM in a room with a linked party, open a member's sheet read-only. As another player, confirm access is denied.
- [x] 7.4 Edit a section in two tabs and confirm the second save reports the conflict without losing input.
- [x] 7.5 Delete a character and confirm its sheets are gone from `character_sheets`.
