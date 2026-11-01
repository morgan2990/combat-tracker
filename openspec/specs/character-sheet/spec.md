# Spec: Character Sheet

## Purpose

Defines the character sheet stored for each PC and companion apart from its PC document: its sections and stored values, per-section owner editing with revision checks, read access for the owner and the party DM, cleanup on deletion, import provenance, and the client page for viewing and editing it.

## Requirements

### Requirement: A character has a sheet stored apart from its PC document
Each PC or companion SHALL have at most one sheet, stored in the `character_sheets` collection and keyed by `pc_id`. A PC with no stored sheet SHALL read as an empty sheet with `revision: 0`. `name` and `max_hp` SHALL be stored only on the PC document. Loading a PC through `GET /api/me` or a WebSocket connection SHALL NOT load its sheet.

#### Scenario: A PC with no sheet reads as empty
- **WHEN** the owner sends `GET /api/pcs/{id}/sheet` for a PC with no stored sheet
- **THEN** the server SHALL return HTTP 200 with an empty sheet and `revision: 0`, and SHALL NOT create a document

#### Scenario: The sheet is returned with the PC and its companions
- **WHEN** the owner sends `GET /api/pcs/{id}/sheet` for a PC with companions
- **THEN** the server SHALL return `{ "pc": ..., "sheet": ..., "companions": [{ "pc": ..., "sheet": ... }] }`

#### Scenario: The sheet is absent from the user's PC list
- **WHEN** a user with a stored sheet sends `GET /api/me`
- **THEN** the returned PCs SHALL NOT include sheet data

### Requirement: A sheet stores displayed values without calculating rules
The sheet SHALL store values as displayed, and the system SHALL NOT derive one stored value from another. Fields that can hold free text SHALL be strings, including `to_hit`, `damage`, a limited use's `max`, and a creature's `ac`. A blank number SHALL be stored as `null`, not `0`.

#### Scenario: Non-numeric values round-trip
- **WHEN** the owner saves a limited use with `max: "2d6"` and an attack with `to_hit: "DC 12"`
- **THEN** a later read SHALL return both values unchanged

#### Scenario: Changing a score does not change its modifier
- **WHEN** the owner saves `abilities` with a new `score` and the previous `mod` for an ability
- **THEN** the server SHALL store both values as sent

#### Scenario: A blank value stays blank
- **WHEN** the owner saves `combat` with `ac: null`
- **THEN** a later read SHALL return `ac: null`

### Requirement: A sheet covers every section of the character sheet
A PC's sheet SHALL support the sections `identity`, `abilities`, `skills`, `combat`, `defenses`, `attacks`, `actions`, `limited_uses`, `features`, `weapon_masteries`, `proficiencies`, `magic_items`, `spellcasting`, `wild_shapes`, and `notes`. A companion's sheet SHALL support only `creature`. The sheet SHALL NOT store current or temporary HP, spent hit dice, death saves, or spent uses or slots.

#### Scenario: Multiclass identity
- **WHEN** the owner saves `identity` with two entries in `classes`, each with a subclass and level
- **THEN** a later read SHALL return both entries in order

#### Scenario: Class text without a parsed level
- **WHEN** the owner saves `identity` with `classes_text: "Beast Master"` and no entries in `classes`
- **THEN** the server SHALL accept and store it

#### Scenario: Actions keep their source
- **WHEN** the owner saves `actions` with a bonus action whose `source` is `Kid Tricks (feat)`
- **THEN** a later read SHALL return that `source`

#### Scenario: Limited uses name a recovery and source
- **WHEN** the owner saves a limited use with `recovery: "dawn"` and `source: "Ring of Jaedear"`
- **THEN** a later read SHALL return both values

#### Scenario: Progressing magic item
- **WHEN** the owner saves a magic item with `name: "Ring of Jaedear"` and `stage: "Level 1"`
- **THEN** a later read SHALL return both values separately

#### Scenario: Proficiency with an unresolved choice
- **WHEN** the owner saves a language with `name: "from Human"` and `pending: true`
- **THEN** a later read SHALL return that entry with `pending: true`

#### Scenario: Notes hold personality fields
- **WHEN** the owner saves `notes` with `personality`, `ideals`, `bonds`, and `flaws`
- **THEN** a later read SHALL return all four values

### Requirement: Spellcasting supports lists from any source
`spellcasting` SHALL hold a list of `spell_lists`, each with a `kind` of `class`, `feat`, `item`, or `other`, and its own ability, save DC, attack, and spells. It SHALL hold slot totals per level for levels 1 to 9, and Pact Magic separately as `pact_magic` with a slot level and total. Each spell SHALL have a `preparation` of `always`, `prepared`, `unprepared`, `known`, `at_will`, `limited`, or `other`; `other` SHALL keep the original value in `mark`.

#### Scenario: Lists from a class, a feat, and an item
- **WHEN** the owner saves `spell_lists` with entries of kind `class`, `feat`, and `item`
- **THEN** a later read SHALL return each list with its own ability, save DC, attack, and spells

#### Scenario: Pact Magic stays separate from other slots
- **WHEN** the owner saves `slots` with 3 first-level slots and `pact_magic` with `slot_level: 1, total: 1`
- **THEN** a later read SHALL return them separately and SHALL NOT combine them

#### Scenario: Always prepared is distinct from prepared
- **WHEN** the owner saves one spell with `preparation: "always"` and another with `preparation: "prepared"`
- **THEN** a later read SHALL return each spell's own value

#### Scenario: Invalid preparation rejected
- **WHEN** the owner saves a spell with an unlisted `preparation`
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Spell level out of range rejected
- **WHEN** the owner saves a spell with `level: 10`
- **THEN** the server SHALL return HTTP 400 and make no change

### Requirement: Companions and wild shapes use a shared creature sheet
A companion's `creature` section and each form in `wild_shapes.forms` SHALL use the same type: species, type, size, alignment, AC, speed, senses, abilities, saves, skills, attacks, traits, gear, and notes. `wild_shapes` SHALL also hold `uses`, `duration`, and `limitations`. A companion's `name`, `max_hp`, and `shares_initiative` SHALL stay on its PC document.

#### Scenario: Owner saves a companion's creature section
- **WHEN** the owner sends `PUT /api/pcs/{id}/sheet/creature` for a companion, with `ac: "13+Prof"` and `speed_text: "40 ft, climb 30 ft, burrow 15 ft"`
- **THEN** the server SHALL store both values as sent

#### Scenario: Owner saves wild shape forms
- **WHEN** the owner saves `wild_shapes` with `uses`, `limitations`, and two forms
- **THEN** a later read SHALL return them, with each form using the creature fields

#### Scenario: Section that does not apply to a companion
- **WHEN** the owner sends `PUT /api/pcs/{id}/sheet/spellcasting` for a companion
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Creature section does not apply to a PC
- **WHEN** the owner sends `PUT /api/pcs/{id}/sheet/creature` for a `type: "pc"` document
- **THEN** the server SHALL return HTTP 400 and make no change

### Requirement: The owner edits a sheet one section at a time
The owner SHALL save a section with `PUT /api/pcs/{id}/sheet/{section}`, sending `{ "revision": N, "data": ... }`. A save SHALL change only that section and SHALL increment `revision`. If `revision` does not match the stored value, the server SHALL return HTTP 409 and make no change.

#### Scenario: Saving one section leaves the others intact
- **WHEN** the owner saves `attacks` on a sheet with stored `notes`
- **THEN** a later read SHALL return the new `attacks` and the unchanged `notes`

#### Scenario: First save creates the sheet
- **WHEN** the owner saves a section with `revision: 0` for a PC with no stored sheet
- **THEN** the server SHALL create the sheet and return `revision: 1`

#### Scenario: Stale revision rejected
- **WHEN** the owner saves a section with a `revision` lower than the stored one
- **THEN** the server SHALL return HTTP 409 and make no change

#### Scenario: Unknown section rejected
- **WHEN** the owner sends `PUT /api/pcs/{id}/sheet/unknown`
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Saving identity updates the PC's name
- **WHEN** the owner saves `identity` with a new `name`
- **THEN** the server SHALL update `name` on the PC document, leaving `max_hp`, `items`, and `currency` unchanged

#### Scenario: Saving combat updates the PC's max HP
- **WHEN** the owner saves `combat` with a new `max_hp`
- **THEN** the server SHALL update `max_hp` on the PC document, leaving `name`, `items`, and `currency` unchanged

#### Scenario: Invalid name or max HP rejected
- **WHEN** the owner saves `identity` with an empty `name`, or `combat` with `max_hp` ≤ 0
- **THEN** the server SHALL return HTTP 400 and make no change

#### Scenario: Oversized section rejected
- **WHEN** a request body exceeds 1 MiB
- **THEN** the server SHALL reject it with HTTP 413 and make no change

### Requirement: Only the owner and the party's DM can read a sheet
Only the PC's owner SHALL be able to edit its sheet. `GET /api/rooms/{room_id}/pcs/{pc_id}/sheet` SHALL give the owner of a room read-only access to the sheet of a PC in that room's linked party, including its companions. No other user SHALL be able to read a sheet, including other members of the party.

#### Scenario: Owner reads their own sheet
- **WHEN** the owner sends `GET /api/pcs/{id}/sheet`
- **THEN** the server SHALL return HTTP 200 with the sheet

#### Scenario: Another user's read reveals nothing
- **WHEN** a user who does not own the PC sends `GET /api/pcs/{id}/sheet`, including a member of the same party
- **THEN** the server SHALL return HTTP 404

#### Scenario: Another user cannot edit
- **WHEN** a user who does not own the PC sends `PUT /api/pcs/{id}/sheet/{section}`
- **THEN** the server SHALL return HTTP 403 and make no change

#### Scenario: DM reads a party member's sheet
- **WHEN** a room's owner sends `GET /api/rooms/{room_id}/pcs/{pc_id}/sheet` for a PC in the room's linked party
- **THEN** the server SHALL return HTTP 200 with the sheet and its companions' sheets

#### Scenario: DM cannot read a PC outside the party
- **WHEN** a room's owner sends that request for a PC not in the linked party, or the room has no linked party
- **THEN** the server SHALL return HTTP 403

#### Scenario: Someone other than the DM uses the room route
- **WHEN** a user who does not own the room sends that request
- **THEN** the server SHALL return HTTP 403

#### Scenario: Unauthenticated requests rejected
- **WHEN** a client without a valid session sends any sheet request
- **THEN** the server SHALL return HTTP 401 and make no change

### Requirement: A sheet is removed with its character
Deleting a PC SHALL also delete its sheet and the sheets of its companions. Deleting a companion SHALL delete only that companion's sheet.

#### Scenario: Deleting a PC removes all related sheets
- **WHEN** the owner sends `DELETE /api/pcs/{id}` for a PC with a sheet and a companion with a sheet
- **THEN** both sheets SHALL be deleted

#### Scenario: Deleting a companion keeps the PC's sheet
- **WHEN** the owner deletes a companion
- **THEN** the companion's sheet SHALL be deleted and the parent PC's sheet SHALL remain

### Requirement: A sheet can record where its data came from
A sheet SHALL support `schema_version` and an optional `source` block holding the source kind, sheet version, sheet build, import date, file name, file hash, mapper version, and import ID. A section save SHALL NOT change `source`.

#### Scenario: A manual save keeps the source
- **WHEN** the owner saves a section on a sheet with a stored `source`
- **THEN** a later read SHALL return the unchanged `source`

#### Scenario: A manually created sheet has no source
- **WHEN** a sheet is created by a section save
- **THEN** a later read SHALL return no `source`, and `schema_version` SHALL be set

### Requirement: Players and the DM can open the sheet page
The client SHALL provide a page at `/characters/:id/sheet` showing every section of the PC's sheet and its companions' sheets. The owner SHALL be able to edit each section there. With `?room=<room_id>`, the page SHALL load through the DM route and SHALL be read-only. The page SHALL be reachable from the Dashboard's character list, from `PlayerView`, and from a party member's row in `DMView`. Opening it from a room SHALL NOT send a WebSocket message or change `RoomState`.

#### Scenario: Owner opens their sheet from the Dashboard
- **WHEN** a user follows the sheet link on one of their characters
- **THEN** the client SHALL show the sheet with edit actions on each section

#### Scenario: Empty sections are editable
- **WHEN** the owner opens a sheet with no stored data
- **THEN** every section SHALL render empty, and the owner SHALL be able to fill it in

#### Scenario: DM opens a party member's sheet
- **WHEN** the DM follows the sheet link on a party member's row
- **THEN** the client SHALL open the sheet in a new tab with no edit actions

#### Scenario: No sheet link for a PC outside the party
- **WHEN** the DM views the row of a PC that is not in the linked party
- **THEN** the client SHALL NOT show a sheet link

#### Scenario: Always-prepared spells are distinguishable
- **WHEN** the page shows a spell with `preparation: "always"` and one with `preparation: "prepared"`
- **THEN** each SHALL have a distinct visible marker

#### Scenario: Magic item shows its uses
- **WHEN** a limited use's `source` matches a magic item's `name`
- **THEN** the page SHALL show that limited use with the magic item

#### Scenario: Save conflict
- **WHEN** a section save returns HTTP 409
- **THEN** the client SHALL show a message, keep the user's input on screen, and offer to reload the sheet
