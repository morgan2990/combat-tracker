## Why

A character is little more than a name, max HP, and an inventory, so players keep the real sheet in an MPMB PDF that the DM and the app cannot see. Storing a full snapshot of the sheet puts it in reach of the owner and the DM during play, and prepares for importing it from the PDF later.

## What Changes

- Add a character sheet per PC or companion, stored as a snapshot of what the sheet displays. The app does not calculate rules: changing an ability score does not update its modifier.
- Cover identity, abilities and saves, skills and passive senses, combat, defenses, attacks, actions, limited uses, features and feats, weapon masteries, proficiencies, magic items, spellcasting, wild shapes, companions, and notes.
- Spellcasting supports lists from classes, feats, and items; Pact Magic separately from other slots; and a preparation state on each spell, including always prepared.
- Add a sheet page that shows every section, with manual editing by section for the owner.
- Add read-only access for the DM of a room whose linked party includes the PC.
- Remove the sheet when its PC or companion is deleted.
- Reserve a `source` block for the future import. Importing from a PDF is out of scope for this change.
- Spent state is deliberately not stored: current and temporary HP, spent hit dice, death saves, and spent uses and slots. Live combat state stays in the room.

## Capabilities

### New Capabilities

- `character-sheet`: storing, viewing, and editing a character's full sheet, including its access rules and lifecycle.

### Modified Capabilities

None. `PUT /api/pcs/{id}` already gains the partial-update behavior this change relies on through `fix-pc-update-clobbers-inventory`, which must ship first.

## Impact

- `store/`: new `character_sheet.go` with the sheet types and the `character_sheets` collection; `store/mongo.go` registers the store and its unique index on `pc_id`.
- `api/`: new `character_sheets.go` and a `sheetStore` interface with a test fake; `DeletePC` also removes sheets.
- `main.go`: three new routes.
- `frontend/src/types.ts`: sheet types.
- `frontend/src/components/`: new sheet page and section components; links from `Dashboard`, `PlayerView`, and `EntityRow`.
- `frontend/src/App.tsx`: new route.
- No change to combat or to `RoomState`. No data migration: the collection starts empty.
