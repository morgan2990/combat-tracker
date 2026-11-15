## Why

Players keep their sheets in the MPMB PDF, and entering a full sheet by hand takes a long time and goes stale after every level-up. Importing the PDF fills the character sheet from the source of truth the table already uses.

## What Changes

- Add importing an unflattened MPMB v14 PDF: read its form fields, map them to the character sheet, and show the differences before anything changes.
- Offer two entry points: importing into an existing character from its sheet page, and "New character from PDF" on the Dashboard.
- Accepting an import overwrites the whole sheet, with no merging. There is no synchronization with the PDF.
- Accepting also updates the PC's name and max HP, and replaces its companions. Companions are deleted and recreated, not matched.
- Accepting replaces items and currency only if the PDF carries at least one item or coin, so importing a sheet with an empty inventory cannot erase the one in the app.
- Keep the raw field dump, sheet version, file hash, and import date alongside the sheet, so a sheet can be re-mapped without another upload. Keep only the latest applied import per character; unconfirmed drafts expire.
- Reject flattened and non-MPMB PDFs with a clear message. Accept uploads up to 64 MiB, one import at a time.
- Add `github.com/pdfcpu/pdfcpu` as a dependency.

## Capabilities

### New Capabilities

- `mpmb-sheet-import`: uploading an MPMB PDF, previewing the differences, confirming or discarding, and the data kept about each import.

### Modified Capabilities

None. The sheet's `source` block already exists for this purpose, and section saves already leave it alone.

## Impact

- `mpmb/` (new package): PDF field reader, literal parser, and mapper, with no database or HTTP dependencies.
- `store/`: `character_sheet_imports` collection with a TTL index, and a whole-sheet replace in `CharacterSheetStore`.
- `api/`: upload, confirm, and discard handlers, and a store interface with a test fake.
- `main.go`: new routes.
- `frontend/src/`: import dialog with the diff view, a button on the sheet page, and an entry on the Dashboard.
- `go.mod`: pdfcpu and its dependencies.
- Live rooms: a recreated companion's entity keeps its old `pc_id` and stops refreshing until the character rejoins. Nothing breaks.
- Recreated companions lose `shares_initiative`, which the PDF does not carry.
