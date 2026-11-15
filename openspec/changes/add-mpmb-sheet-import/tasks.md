## 1. PDF reader (`mpmb/`)

- [x] 1.1 Add `github.com/pdfcpu/pdfcpu` to `go.mod`, and call `api.DisableConfigDir()` so nothing is written to the home directory.
- [x] 1.2 Implement `mpmb.Read`: `api.ReadContext` without validation, walking the AcroForm by qualified `/T`, keeping `/TM`, inherited `/FT`, stream values, and non-empty values only.
- [x] 1.3 Resolve a choice field's `display` through `/I` and `/Opt`, and read the sheet version, build, type, and title from the document info.
- [x] 1.4 Add the limits: decode limit per value, field count, depth, `recover`, and never decoding `User_Imported_Files.Stringified`. Return distinct errors for no form fields, no MPMB metadata, and unreadable files.
- [x] 1.5 Add a small synthetic PDF fixture covering `/Type /Template`, `/TM`, stream values, and `/I` with `/Opt`, and reader tests including each rejection and the limits.

## 2. Literal parser and spells

- [x] 2.1 Implement the recursive-descent literal parser: unquoted keys, both quote styles, arrays, numbers, booleans; an error for anything else. Add tests, including a function value and malformed input.
- [x] 2.2 Map spell lists from `spellshead.*` and spell rows in order: list changes, level dividers, skipped control rows, key, and each `check` value. Add tests, including lists on continuation pages.
- [x] 2.3 Take list kinds from `CurrentSpells`, and split Pact Magic using the Warlock slot table. Add tests, including a parse failure and a negative result.

## 3. Direct mapping

- [x] 3.1 Add the `mpmb.Map` result type, warnings, and `Template.extras.*` prefix resolution.
- [x] 3.2 Map identity (classes from headers, base class from features, size from its label), abilities, saves, skills, combat, and defenses.
- [x] 3.3 Map attacks, actions (source from `/TM`), limited uses (source from the matching magic item), proficiencies (with `pending`), magic items (with `stage`), and notes.
- [x] 3.4 Add synthetic field-list tests for every mapping scenario in the spec, including that spent state is left out.

## 4. Features

- [x] 4.1 Parse feature rows from `Class Features`, `Extra.Notes`, and the notes pages into name, source, level, book, page, uses note, and text, removing markers. Add tests for rows with and without level or book, and for text that is not a row.
- [x] 4.2 Map feats, merging a same-named notes row, and split `Racial Traits` into species features. Add tests.
- [x] 4.3 Extract weapon masteries from their rows, leaving them out of features. Add tests.

## 5. Companions, wild shapes, inventory

- [x] 5.1 Map companions across every `AScomp` page, with name fallback and the HP warning. Add tests, including two companions and one without a name.
- [x] 5.2 Map wild shapes: uses, duration, limitations, and forms from `Wildshape.Race.N` and `Wildshape.N.*`. Add tests.
- [x] 5.3 Map inventory from the gear and coin fields, including non-numeric amounts and the "has inventory" result. Add tests.
- [x] 5.4 Add the end-to-end test against real sheets, which runs only when `MPMB_SAMPLES_DIR` is set, and run it locally.

## 6. Store

- [x] 6.1 Add `CharacterSheetImportStore` for `character_sheet_imports`, with a TTL index on `expires_at` and create, get, mark applied, delete, and delete-older operations.
- [x] 6.2 Add `ReplaceSheet` to `CharacterSheetStore`: a compare-and-set on revision that replaces all sections and sets `source`, creating the sheet at revision 0.
- [x] 6.3 Add integration tests using the shared testcontainer, including field names with dots, a conflict, `source` kept by later section saves, and draft expiry fields.

## 7. Differences

- [x] 7.1 Implement the diff per section on the clean model, including name and max HP, pairing list rows by their title field. Add tests for changes, removals, reordering, and no differences.
- [x] 7.2 Add companions and inventory to the diff, including the case where the inventory is left alone. Add tests.

## 8. API

- [x] 8.1 Add a store interface with a test fake, the upload handling (streaming to a temp file, hash, 64 MiB limit, removal afterward), and the one-at-a-time guard.
- [x] 8.2 Implement `POST /api/pcs/{id}/sheet/import` and `POST /api/sheet-imports`, and map reader errors to their responses. Register the routes. Add handler tests for each response in the spec.
- [x] 8.3 Implement the confirm: revision check, then sheet, PC, new companions, old companions, and marking applied, including the inventory rule and catalog links. Add tests, including a 409 that changes nothing and a repeated confirm.
- [x] 8.4 Add the new-character path to the confirm, including its validation. Add tests.
- [x] 8.5 Implement `DELETE /api/sheet-imports/{id}` and the access rules. Add tests for another user and an expired draft.

## 9. Frontend

- [x] 9.1 Add the import types and an import dialog: choose a file, upload, show warnings and differences by section, and confirm or cancel (discarding the draft). Add tests.
- [x] 9.2 Handle a rejected file, a 429, and a 409 on confirm in the dialog. Add tests.
- [x] 9.3 Add the import action to the sheet page, for the owner of a top-level PC only, reloading the sheet after confirming. Add tests, including no action in the DM view.
- [x] 9.4 Add "New character from PDF" to the Dashboard, opening the new sheet and refreshing the list after confirming. Add tests.

## 10. Verification

- [x] 10.1 Run `go test ./...`, the store integration tests, and `npm test` and `npm run build` in `frontend/`.
- [x] 10.2 In the running app, import Puck Adulto and Puck Niño into new characters, and compare each section against the PDF.
- [x] 10.3 Reimport a sheet after a manual edit and confirm the diff shows the edit being lost. Confirm that a character with items keeps them after importing a sheet with no inventory.
- [x] 10.4 Upload a flattened or non-MPMB PDF and confirm the message.
- [ ] 10.5 After deploying, confirm that a real upload passes through the Cloudflare tunnel and that no pdfcpu directory appears in the container.
