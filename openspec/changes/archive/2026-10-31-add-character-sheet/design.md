## Context

`store.PC` holds `name`, `max_hp`, `items`, and `currency`; companions are `PC` documents with `parent_pc_id`. `GET /api/me` and each WebSocket connection load whole `PC` documents, so whatever lives on `PC` is paid for on every one of those reads.

Players keep their sheets in the MPMB PDF (v14.0.8-beta). Reading two real sheets shaped this model: many values are not numeric (limited-use max `2d6`, companion AC `13+Prof`, to-hit `DC 12`), spell lists come from feats and items as well as classes, and a companion may have no name. The import itself comes in a later change, but this model must be able to hold what it will produce.

Depends on `fix-pc-update-clobbers-inventory`: saving `name` or `max_hp` from a section goes through `UpdatePC`, which must not erase the inventory.

## Goals / Non-Goals

**Goals:**
- Hold everything the sheet displays, section by section.
- Let the owner edit each section without affecting the others.
- Let the DM read the sheets of the party in their room.
- Leave room for the import without another migration.

**Non-Goals:**
- Importing the PDF, the raw field dump, or the reimport diff.
- Any rules calculation or validation of 5e legality.
- Spent state, or syncing the sheet with a live room.
- Linking the child and adult sheets of one character; they are separate PCs for now.
- Changing inventory or currency.

## Decisions

### A separate collection, one document per PC

`character_sheets` has a unique index on `pc_id`. Companions get their own document. Embedding in `PC` was rejected because it would add the whole sheet to `/api/me` and to every WebSocket connection, which only need a name and HP.

### `name` and `max_hp` live only on `PC`

The sheet does not duplicate them, so there is nothing to keep in sync and combat keeps reading the same fields. Responses include the `PC` next to the sheet. `PUT .../sheet/identity` accepts `name` and `PUT .../sheet/combat` accepts `max_hp`; the handler writes those to `PC`, sending the stored value for the other required field.

### Snapshot values, not derived ones

Modifiers and bonuses are stored as displayed, since homebrew scripts change them. Fields the sheet can fill with free text are strings. Numbers that may be blank are nullable, because blank is not zero.

### Sections

| Section | Content |
|---|---|
| `identity` | `player`, `classes_text`, `character_level`, `classes[{class, subclass, level}]`, `species`, `background`, `size`, `alignment`, `xp` |
| `abilities` | `str` to `cha`: `{score, mod, save_bonus, save_proficient}` |
| `skills` | `entries[{key, ability, bonus, proficient, expertise}]`, `passive_perception`, `passive_insight`, `passive_investigation` |
| `combat` | `ac`, `initiative`, `proficiency_bonus`, `speed_text`, `speeds{walk, climb, swim, fly, burrow}`, `hit_dice[{die, total}]` |
| `defenses` | `vision`, `resistances[]`, `save_notes` |
| `attacks` | `[{name, ability, range, to_hit, damage, damage_type, notes}]` |
| `actions` | `action[]`, `bonus_action[]`, `reaction[]`, each `{name, source, is_default}` |
| `limited_uses` | `[{name, max, recovery, source}]` |
| `features` | `[{kind, name, text, source, level, book, page, uses_note}]` |
| `weapon_masteries` | `[{weapon, mastery, notes}]` |
| `proficiencies` | `armor[]`, `weapons[]`, `tools[]`, `languages[]`, each `{name, pending}` |
| `magic_items` | `[{name, stage, attunement_required, attuned, text, source, catalog_item_id}]` |
| `spellcasting` | `spell_lists[]`, `slots[9]` (totals), `pact_magic{slot_level, total}` |
| `wild_shapes` | `uses`, `duration`, `limitations`, `forms[creature]` |
| `notes` | `appearance`, `personality`, `ideals`, `bonds`, `flaws`, `backstory`, `free_text` |
| `creature` | `species`, `type`, `size`, `alignment`, `ac`, `speed_text`, `senses`, `abilities`, `saves[]`, `skills[]`, `attacks[]`, `traits`, `gear[]`, `notes` |

- **Spell lists:** `{kind: class|feat|item|other, name, ability, save_dc, attack, prepare_count, spells[]}`.
- **Spells:** `{key, name, level, preparation, mark, save, school, casting_time, range, components, duration, book, page, description}`.
- **`preparation`:** `always`, `prepared`, `unprepared`, `known`, `at_will`, `limited`, or `other`. `mark` keeps the original value when it is `other`.
- **Feature `kind`:** `class`, `species`, `background`, `feat`, or `other`. A block of text that cannot be split is a single row.
- **Magic item uses:** these are not on the item, since the sheet tracks them as limited uses. A limited use names its item in `source`, and the page shows them together.

A PC accepts every section except `creature`. A companion accepts only `creature`. Wild shape forms and companions share the `creature` type.

### Saving by section, with a revision check

`PUT /api/pcs/{id}/sheet/{section}` takes `{revision, data}` and returns the new `revision`. The store runs a compare-and-set on `{pc_id, revision}`, setting only that section. With `revision: 0` it upserts; a duplicate key on `pc_id` means someone else created the document first. A mismatch returns 409.

Saving the whole document was rejected: it would repeat the bug fixed in `fix-pc-update-clobbers-inventory`, where saving one part erases another.

The sheet is written first, then `PC`. If the `PC` write fails, the client gets a 500 and reloads; `name` and `max_hp` keep their old values and no sheet data is lost.

### Typed sections, validated on save

Each section decodes into a Go struct, so unknown keys are dropped. The handler returns 400 for an unknown section, a section that does not apply to the document's type, an invalid enum, a spell level outside 0-9, or an empty `name` or non-positive `max_hp`. The body is limited to 1 MiB with `http.MaxBytesReader`, well above the largest section seen in real sheets.

### Access

| Route | Who | Otherwise |
|---|---|---|
| `GET /api/pcs/{id}/sheet` | Owner | 404, like `GetPC` |
| `PUT /api/pcs/{id}/sheet/{section}` | Owner | 403, through `requireOwnedPC` |
| `GET /api/rooms/{room_id}/pcs/{pc_id}/sheet` | DM of the room, if the PC is in its linked party | 403, through `requireOwnedRoom` and `resolveLinkedParty`, like `UpdatePartyInventoryPC` |

Both reads return `{pc, sheet, companions: [{pc, sheet}]}`. A PC with no sheet yet returns an empty sheet with `revision: 0`, not a 404. The DM's response includes `pc.items` and `pc.currency`, which the DM can already see.

### Prepared for the import

The document carries `schema_version` and an optional `source` block: `{kind, sheet_version, sheet_build, imported_at, file_name, file_sha256, mapper_version, import_id}`. This change never writes `source`, and a manual save leaves it in place. The raw dump will live in its own `character_sheet_imports` collection, created by the import change.

### Page

`/characters/:id/sheet` shows the sections with the existing `CollapsibleSection`, and the same page serves the DM as `/characters/:id/sheet?room=<room_id>`, read-only. The DM link opens in a new tab so combat stays on screen. Companions and wild shapes render through one shared creature component. Most sections are lists of rows, so editing shares one row-editor component rather than having a form per section.

## Risks / Trade-offs

- [Edits do not recalculate, so a changed score can leave a stale modifier] → Stated in the UI next to the ability editor. Recalculating would be wrong for homebrew.
- [Two sheet types share one collection] → Section rules by `PC.type` are enforced in one place in the handler.
- [Deleting a PC writes to two collections without a transaction] → The `PC` is deleted first, then its sheets. A failure leaves an unreachable sheet and is logged; it does no harm to users.
- [The model has not yet been exercised by a real import] → It was checked against two real sheets, and `schema_version` allows later changes.

## Migration Plan

No migration. The store creates the index on startup and PCs without a sheet read as empty. Rollback leaves an unused collection.
