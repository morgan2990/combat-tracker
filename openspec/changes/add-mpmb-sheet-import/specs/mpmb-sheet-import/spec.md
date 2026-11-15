## ADDED Requirements

### Requirement: An owner can upload an MPMB PDF for preview without changing anything
An authenticated user SHALL be able to upload an MPMB PDF for a top-level PC they own with `POST /api/pcs/{id}/sheet/import`, or to start a new character with `POST /api/sheet-imports`. The server SHALL read the form fields, map them, store a draft, and return the import ID, source details, warnings, and differences. An upload SHALL NOT change the sheet, the PC, its companions, or its inventory.

#### Scenario: Upload for an existing character
- **WHEN** the owner uploads a valid MPMB PDF to `POST /api/pcs/{id}/sheet/import`
- **THEN** the server SHALL return HTTP 200 with an import ID and the differences, and the stored sheet and PC SHALL be unchanged

#### Scenario: Upload for a new character
- **WHEN** an authenticated user uploads a valid MPMB PDF to `POST /api/sheet-imports`
- **THEN** the server SHALL return HTTP 200 with an import ID and a preview, and SHALL NOT create a PC

#### Scenario: Upload for someone else's character
- **WHEN** a user uploads to `POST /api/pcs/{id}/sheet/import` for a PC they do not own
- **THEN** the server SHALL return HTTP 403 and store nothing

#### Scenario: Upload for a companion
- **WHEN** the owner uploads to `POST /api/pcs/{id}/sheet/import` for a `type: "companion"` document
- **THEN** the server SHALL return HTTP 400 and store nothing

#### Scenario: Upload without a session
- **WHEN** a client without a valid session uploads a PDF
- **THEN** the server SHALL return HTTP 401 and store nothing

### Requirement: Unusable files are rejected with a clear message
The server SHALL reject a file it cannot import with HTTP 400 and a message naming the reason, and SHALL store nothing. The upload limit SHALL be 64 MiB, returning HTTP 413 above it. The server SHALL process one import at a time and SHALL return HTTP 429 to a concurrent one. The uploaded file SHALL be deleted when the request ends, and SHALL NOT be kept.

#### Scenario: Flattened PDF
- **WHEN** a user uploads an MPMB PDF with no form fields
- **THEN** the server SHALL return HTTP 400 with a message saying the sheet must not be flattened

#### Scenario: Not an MPMB sheet
- **WHEN** a user uploads a PDF without MPMB sheet version metadata
- **THEN** the server SHALL return HTTP 400 with a message saying it is not an MPMB sheet

#### Scenario: Corrupt file
- **WHEN** a user uploads a file that cannot be read as a PDF, including one that makes the PDF library fail
- **THEN** the server SHALL return HTTP 400 and keep running

#### Scenario: Oversized file
- **WHEN** a user uploads a file larger than 64 MiB
- **THEN** the server SHALL return HTTP 413

#### Scenario: Concurrent import
- **WHEN** a user uploads a PDF while another upload is being processed
- **THEN** the server SHALL return HTTP 429 with a message asking to retry

#### Scenario: Unknown sheet version
- **WHEN** a user uploads an MPMB PDF whose major version is not 14
- **THEN** the server SHALL process it and SHALL include a warning about the version

### Requirement: Nothing embedded in the PDF is executed
The system SHALL treat every value in the PDF as data. Embedded JavaScript SHALL NOT be executed or evaluated, and the stored script bundle (`User_Imported_Files.Stringified`) SHALL NOT be decoded or stored. Stream values SHALL be decoded with a size limit, and walking the fields SHALL be limited in depth and count.

#### Scenario: Script bundle left out
- **WHEN** a sheet with loaded scripts is imported
- **THEN** the stored raw fields SHALL NOT include `User_Imported_Files.Stringified`

#### Scenario: A literal containing a function
- **WHEN** a parsed field holds a function instead of plain data
- **THEN** the parser SHALL return an error, and the import SHALL continue with a warning

#### Scenario: Excessive nesting
- **WHEN** a PDF's fields exceed the depth or count limit
- **THEN** the server SHALL return HTTP 400 and store nothing

### Requirement: The sheet is mapped from the PDF's form fields
The mapper SHALL fill every section of the character sheet from the form fields, resolving page prefixes from `Template.extras.*`. It SHALL NOT import spent state. A value that cannot be mapped SHALL NOT fail the import; it SHALL produce a warning and remain in the raw fields.

#### Scenario: Field named through its hierarchy
- **WHEN** a field has a `/TM` mapping name
- **THEN** the field SHALL be identified by its qualified `/T` name, with `/TM` kept separately

#### Scenario: Pages at different positions
- **WHEN** two sheets have their spell pages under `P6.SSfront.` and `P7.SSfront.`
- **THEN** both SHALL map the same way

#### Scenario: Multiclass identity
- **WHEN** `Class Features` has the headers `Warlock (Archfey Patron), level 1` and `Druid (Circle of Dreams), level 2`
- **THEN** `identity.classes` SHALL hold both, with class, subclass, and level

#### Scenario: Class without a base name
- **WHEN** `Class and Levels` is `Beast Master`, the header is `Beast Master, level 3`, and its features cite `Ranger 1`
- **THEN** the class SHALL be `Ranger`, the subclass `Beast Master`, and the level 3

#### Scenario: Size resolved from its label
- **WHEN** `Size Category` holds the value `1` with the option `Small` selected
- **THEN** the size SHALL be `Small`, not `Medium`

#### Scenario: Action source
- **WHEN** `Action 9` has the `/TM` `Toma wacho!#!#Kid Tricks (feat)`
- **THEN** the action SHALL have the name `Toma wacho!` and the source `Kid Tricks (feat)`

#### Scenario: Default action
- **WHEN** an action's `/TM` source is `Default action`
- **THEN** the action SHALL have `is_default: true`

#### Scenario: Feature row split into fields
- **WHEN** `Class Features` holds `#◆ Favored Enemy# (Ranger 1, P24 119) #[2× per LR or SS]#` followed by text
- **THEN** a feature SHALL have the name `Favored Enemy`, source `Ranger`, level 1, book `P24`, page `119`, uses note `2× per LR or SS`, and that text, with no marker characters

#### Scenario: Feat present in two places
- **WHEN** `Kid Tricks` is both a `Feat Name` and a row on the notes page
- **THEN** there SHALL be one `feat` feature, with the longer text

#### Scenario: Weapon mastery
- **WHEN** `Extra.Notes` holds `#◆ Sling# (Weapon Mastery)` with text naming the `'Slow' Mastery`
- **THEN** `weapon_masteries` SHALL hold `Sling` with `Slow`, and no feature SHALL be created for it

#### Scenario: Unresolved proficiency choice
- **WHEN** `Language 3` is `from Human`
- **THEN** that language SHALL have `pending: true`

#### Scenario: Progressing magic item and its uses
- **WHEN** the item is `Ring of Jaedear (Level 1)` and a limited use is `Ring of Jaedear (Level 1) (Shield)`
- **THEN** the item SHALL have the name `Ring of Jaedear` and stage `Level 1`, and the use SHALL have the source `Ring of Jaedear`

#### Scenario: Spell lists by source
- **WHEN** `CurrentSpells` marks lists with `refType` `class`, `feat`, and `item`
- **THEN** each spell list SHALL have that kind

#### Scenario: Spell level from row order
- **WHEN** a spell row follows `setdivider##1##1`
- **THEN** that spell SHALL have level 1, and rows before the first divider SHALL have level 0

#### Scenario: Spell preparation
- **WHEN** spell rows have `check` values of `markedbox`, `checkedbox`, `checkbox`, blank, `atwill`, and `oncelr`
- **THEN** their preparation SHALL be `always`, `prepared`, `unprepared`, `known`, `at_will`, and `limited`

#### Scenario: Unknown spell marker
- **WHEN** a spell row has an unknown `check` value
- **THEN** its preparation SHALL be `other`, with the original in `mark`, and a warning SHALL be added

#### Scenario: Pact Magic separated
- **WHEN** the sheet shows 4 first-level slots for Warlock 1 and Druid 2
- **THEN** `pact_magic` SHALL have 1 slot of level 1, and `slots` SHALL have 3 first-level slots

#### Scenario: Unreadable spell data
- **WHEN** `CurrentSpells` cannot be parsed
- **THEN** spell lists SHALL default to `class`, slots SHALL stay combined, and a warning SHALL be added

#### Scenario: Wild shapes
- **WHEN** the sheet has a wild shape page with the forms `Rat` and `Cat`
- **THEN** `wild_shapes.forms` SHALL hold both, each named and using the creature fields

#### Scenario: Spent state not imported
- **WHEN** a sheet has values in `HP Current` and `Limited Feature Used 1`
- **THEN** the mapped sheet SHALL NOT include those values

### Requirement: The user sees the differences before accepting
The response to an upload SHALL list the changes per section between the stored and imported data, including name and max HP. It SHALL list the companions to remove and create, and whether items and currency would be replaced. Raw fields SHALL NOT be compared.

#### Scenario: Changed value
- **WHEN** the imported `combat.ac` differs from the stored one
- **THEN** the differences SHALL show the section, field, old value, and new value

#### Scenario: Reordered list
- **WHEN** the same attacks appear in a different order
- **THEN** the differences SHALL NOT report those attacks as changed

#### Scenario: Manual edits are shown as losses
- **WHEN** the stored sheet has an attack that the PDF does not
- **THEN** the differences SHALL show that attack as removed

#### Scenario: No differences
- **WHEN** the imported data matches what is stored
- **THEN** the differences SHALL be empty

#### Scenario: Companions listed
- **WHEN** the character has a companion and the PDF has another
- **THEN** the differences SHALL list the first as removed and the second as added

### Requirement: Confirming replaces the sheet as a whole
`POST /api/sheet-imports/{id}/confirm` SHALL replace the whole sheet with the draft, set `source`, and update the PC's name and max HP. Sections absent from the PDF SHALL be cleared. If the sheet's revision changed since the upload, the server SHALL return HTTP 409 and change nothing. There SHALL be no partial acceptance.

#### Scenario: Confirm an import
- **WHEN** the owner confirms a draft for an existing character
- **THEN** the sheet SHALL match the draft, `revision` SHALL increase, and `name` and `max_hp` SHALL be updated on the PC

#### Scenario: Source recorded
- **WHEN** an import is confirmed
- **THEN** `source` SHALL hold kind `mpmb`, the sheet version and build, import date, file name, file hash, mapper version, and import ID

#### Scenario: Sheet changed after the upload
- **WHEN** the sheet is saved after the upload and the owner then confirms
- **THEN** the server SHALL return HTTP 409 and change nothing

#### Scenario: Confirm twice
- **WHEN** an already applied import is confirmed again
- **THEN** the server SHALL return HTTP 409 and change nothing

#### Scenario: A manual save after an import
- **WHEN** the owner saves a section after an import
- **THEN** `source` SHALL remain unchanged

#### Scenario: Invalid HP on an existing character
- **WHEN** `HP Max` in the PDF is not a positive number
- **THEN** the PC SHALL keep its current `max_hp`, and the preview SHALL include a warning

### Requirement: A new character can be created from a PDF
Confirming a draft from `POST /api/sheet-imports` SHALL create a PC owned by the requester, using the sheet's name and max HP, and then apply the import to it.

#### Scenario: Create from a PDF
- **WHEN** a user confirms a draft for a new character
- **THEN** a PC SHALL be created with the sheet, companions, and inventory from the draft, and the response SHALL include its `pc_id`

#### Scenario: Sheet with no usable name or HP
- **WHEN** a draft for a new character has no `PC Name` or no valid `HP Max`
- **THEN** the confirm SHALL return HTTP 400 and SHALL NOT create a PC

### Requirement: Confirming replaces the character's companions
Confirming SHALL create a companion for each one in the PDF, and SHALL delete the character's existing companions and their sheets. Companions SHALL NOT be matched between the two. A companion with no name SHALL use its species. New companions SHALL have `shares_initiative: false`.

#### Scenario: Companions replaced
- **WHEN** a character with a companion confirms an import with another one
- **THEN** only the imported companion SHALL remain, with its creature sheet, and the old companion's sheet SHALL be deleted

#### Scenario: Companion without a name
- **WHEN** `Comp.Desc.Name` is empty and `Comp.Race` is `Beast of the Burrow`
- **THEN** the companion SHALL be named `Beast of the Burrow`

#### Scenario: Companion without numeric HP
- **WHEN** a companion's max HP is not a number
- **THEN** it SHALL be created with `max_hp: 1`, and the preview SHALL include a warning

#### Scenario: Several companions
- **WHEN** `Template.extras.AScomp` lists two pages
- **THEN** two companions SHALL be created

### Requirement: Inventory is replaced only when the PDF has one
Confirming SHALL replace the PC's items and currency only if the PDF holds at least one item or a non-zero coin. An item whose name exactly matches a stored one SHALL keep its catalog reference.

#### Scenario: PDF with no inventory
- **WHEN** a character with items confirms an import of a sheet with no items or coins
- **THEN** the items and currency SHALL be unchanged, and the differences SHALL say so

#### Scenario: PDF with inventory
- **WHEN** the PDF has items or coins
- **THEN** both the items and the currency SHALL be replaced by those in the PDF

#### Scenario: Catalog link kept
- **WHEN** a stored item with a `catalog_item_id` has the same name as an imported item
- **THEN** the imported item SHALL keep its `catalog_item_id` and `reference_url`

#### Scenario: Non-numeric amount
- **WHEN** an item's amount is not a number
- **THEN** its quantity SHALL be 1

### Requirement: Import data is kept privately and for a limited time
Each import SHALL store the raw fields as an array of `{name, tm, type, value, display}` with non-empty values only. A draft SHALL expire after 24 hours. Only the latest applied import per PC SHALL be kept. Only the user who uploaded an import SHALL be able to confirm or discard it, and the DM route SHALL NOT expose imports. The source file itself SHALL NOT be stored.

#### Scenario: Field names containing dots
- **WHEN** a field is named `Attack.1.To Hit`
- **THEN** the raw fields SHALL store that name unchanged

#### Scenario: Draft discarded
- **WHEN** the uploader sends `DELETE /api/sheet-imports/{id}`
- **THEN** the draft SHALL be deleted and nothing else SHALL change

#### Scenario: Older imports removed
- **WHEN** a character with an applied import confirms a new one
- **THEN** only the new import SHALL remain for that character

#### Scenario: Someone else's import
- **WHEN** a different user confirms or discards an import
- **THEN** the server SHALL return HTTP 404 and change nothing

#### Scenario: Expired draft
- **WHEN** a user confirms a draft that no longer exists
- **THEN** the server SHALL return HTTP 404

### Requirement: Players import from the sheet page and the Dashboard
The sheet page SHALL offer an import action to the owner of a top-level PC. The Dashboard SHALL offer "New character from PDF". Both SHALL upload the file, then show the warnings and the differences, with actions to confirm or cancel. The DM's read-only view SHALL NOT offer an import.

#### Scenario: Import from the sheet page
- **WHEN** the owner selects a PDF on the sheet page
- **THEN** the client SHALL show the differences, and the sheet SHALL change only after confirming

#### Scenario: New character from the Dashboard
- **WHEN** a user confirms an import started from the Dashboard
- **THEN** the client SHALL open the new character's sheet, and the character SHALL appear in the Dashboard list

#### Scenario: Cancel
- **WHEN** the user cancels after seeing the differences
- **THEN** the client SHALL discard the draft and nothing SHALL change

#### Scenario: Warnings shown
- **WHEN** an upload returns warnings
- **THEN** the client SHALL list them before the confirm action

#### Scenario: Rejected file
- **WHEN** an upload is rejected
- **THEN** the client SHALL show the server's message

#### Scenario: Conflict on confirm
- **WHEN** a confirm returns HTTP 409
- **THEN** the client SHALL say the sheet changed and offer to upload again

#### Scenario: DM view
- **WHEN** the DM opens a party member's sheet
- **THEN** the page SHALL NOT offer an import
