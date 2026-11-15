## Context

The character sheet (`store.CharacterSheet`) already models everything the import needs, including a reserved `source` block. This change fills it from a PDF.

Findings from two real MPMB v14.0.8-beta sheets, of about 30 MB and 5,000-7,000 form fields each:

- pdfcpu's validation rejects them, because MPMB's hidden template pages use `/Type /Template`. Reading with `api.ReadContext`, which does not validate, and walking the AcroForm matched an independent pypdf reader on every field name, value, `/TM`, and type.
- On the production Pi 5, one read takes about 0.5 s and 110 MB, including the app's other containers running alongside it.
- pypdf's `get_fields()` keys fields by `/TM` when present, which hides real fields. Fields must be named by qualified `/T`.

## Goals / Non-Goals

**Goals:**
- Import an MPMB v14 sheet with no manual steps beyond reviewing the diff.
- Never change anything before the user confirms.
- Keep the raw fields, so mapping fixes can be applied without another upload.

**Non-Goals:**
- Synchronizing with the PDF, merging, or partial acceptance.
- Importing spent state: current HP, spent dice, uses, or slots.
- Exporting back to PDF.
- Running or evaluating anything embedded in the PDF.
- Re-mapping stored imports. The data is kept to allow it; the feature comes later.
- Linking the child and adult sheets of one character.

## Decisions

### Read without validation, with our own limits

`mpmb.Read` uses `api.ReadContext` and walks the AcroForm itself. `form export` is not an option, since it validates. It calls `api.DisableConfigDir()`, because pdfcpu otherwise writes a configuration directory into the user's home.

Skipping validation means the file gets less checking, and any signed-in user can upload one, so the reader enforces its own limits:

- `DecodeWithLimit` on stream values, 1 MiB per value
- 20,000 fields and depth 32 when walking
- `recover` around the library, which turns a panic into a rejected file
- `User_Imported_Files.Stringified` is never decoded: it is 8 MB of scripts, including published book content

A field becomes `{name, tm, type, value, display}`. `display` resolves a choice field's label through `/I` and `/Opt`, since `Size Category` stores `1` for both Medium and Small. Only non-empty fields are kept, as an array rather than a map because names contain dots.

### A pure `mpmb` package

`mpmb.Read` produces fields and sheet info. `mpmb.Map` produces `{sheet, pc, companions, inventory, warnings}`. Neither touches the database or HTTP, so the mapping can be tested from field lists alone.

### Mapping rules

Page prefixes come from `Template.extras.*`, never fixed numbers, and each value is a comma-separated list, so several companions or spell pages work. The unprefixed template master fields are ignored.

| Target | Source |
|---|---|
| PC name, max HP | `PC Name`, `HP Max` |
| Classes | `Class and Levels` as `classes_text`; entries from `Class Features` headers (`**Druid (Circle of Dreams), level 2:**`). A header without a base class takes it from its first feature's source (`Ranger 1`) |
| Size | `Size Category` display label |
| Abilities, saves | `Str`, `Str Mod`, `Str ST Mod`, `Str ST Prof`, and so on. The optional seventh score, `HoS`, stays in the raw fields only |
| Skills | The 18 skill prefixes: bonus, `Prof`, `Exp`, with a fixed name and ability table |
| Combat | `AC`, `Initiative bonus`, `Proficiency Bonus`, `Speed` as `speed_text`, `HD1`-`HD3`. Numeric speeds are parsed only when `Unit System` is imperial |
| Attacks | `Attack.N.*`, using `Weapon Selection` as the name; `Cha Mod` becomes `Cha`; empty rows are skipped |
| Actions | `Action N`, `Bonus Action N`, `Reaction N`; `/TM` `name#!#source` gives the source, and `Default action` sets `is_default` |
| Limited uses | `Limited Feature N` and its `Max Usages` and `Recovery`. `source` is the magic item whose full printed name starts the use's name |
| Features | `Class Features`, `Extra.Notes`, and the notes pages: rows `#◆ Name# (Source N, Book Page) #[note]#` and the text after them |
| Feats | `Feat Name`, `Description`, and `Note`. A notes-page row with the same name merges into the feat, keeping the longer text |
| Species features | `Racial Traits`, split on its bullets |
| Weapon masteries | Rows `#◆ Sling# (Weapon Mastery)` and the `'Slow' Mastery` in their text; these do not become features |
| Proficiencies | Armor and weapon checkboxes, `Tool N`, `Language N`; `pending` when the text starts with `from ` or ends in `(n/m)` |
| Magic items | `Extra.Magic Item N`; `(Level 1)` becomes `stage`; `Note` becomes `source`; `Attuned` becomes `attuned`. `attunement_required` cannot be derived and stays false |
| Spell lists | `spellshead.*` for name, ability, DC, attack, and prepare count; `refType` in `CurrentSpells.Stringified` gives the kind |
| Spells | Rows in order: `setheader##x##n` changes the list, `setdivider##level##n` changes the level, `setcaptions` and `hidethisline` are skipped. `remember` holds the key. `check` maps `markedbox` to `always`, `checkedbox` to `prepared`, `checkbox` to `unprepared`, blank to `known`, `atwill` to `at_will`, `oncelr` to `limited`, and anything else to `other` with the original in `mark` |
| Slots | `SpellSlots.CheckboxesSet.lvlN` minus Pact Magic |
| Pact Magic | A `CurrentSpells` entry with `factor:[n,"warlock"]`, its level, and a Warlock slot table. The one rules table in the system, because the sheet only stores the combined total. If the result would be negative, the total is kept and a warning is added |
| Wild shapes | `Wildshapes.Info.*`; forms from `Wildshape.Race.N` and `Wildshape.N.*` |
| Companions | `Comp.Desc.Name`, falling back to `Comp.Race`; `Comp.Use.*` and `Comp.eqp.*` |
| Notes | `Personality Trait`, `Ideal`, `Bond`, `Flaw`, `Background_History`, `Background_Appearance`; free text from the notes fields that is not a feature row |
| Inventory | `Adventuring Gear Row/Amount N`, `Extra.Gear Row/Amount N`, and the five `Pieces` fields |

The markers `#`, `##`, and `**` are removed from text, and `\r` becomes a newline.

A value that does not fit never fails the import. It stays in the raw fields, and a warning explains what happened:

- Invalid `HP Max`: the PC keeps its current value. A new character cannot be created without one, so the confirm is rejected.
- A companion without numeric HP: created with 1, because the model requires a positive value.

### A small literal parser, never `eval`

`CurrentSpells.Stringified` is a JavaScript literal: unquoted keys, either quote style, arrays, numbers, booleans. A recursive-descent parser reads only that subset and returns an error for anything else, such as a function. A parse failure is a warning: lists default to `class`, and Pact Magic stays combined. Fields that can hold functions, such as `CurrentEvals`, are never parsed.

### Draft, then confirm

`POST` parses and stores a draft in `character_sheet_imports`; nothing else changes. The response carries the warnings and the diff. Confirming applies the draft. A draft expires after 24 hours through a TTL index on `expires_at`, and confirming clears it. When a confirm succeeds, older imports for that PC are deleted.

Holding the parsed result avoids uploading and parsing twice, and means the confirm applies exactly what the user reviewed.

### Whole-sheet replace guarded by revision

`ReplaceSheet` is a compare-and-set on the draft's `base_revision`. If the sheet changed after the diff was produced, the confirm returns 409 and changes nothing. The sheet can only be replaced this way; section saves still leave `source` alone. Sections missing from the PDF are cleared, because an import overwrites.

There are no multi-document transactions, so the confirm applies in the order that fails most safely:

1. Replace the sheet. This is the guard: a conflict stops everything.
2. Update the PC's name, max HP, and, under the inventory rule, items and currency.
3. Create the new companions and their sheets.
4. Delete the old companions and their sheets.
5. Mark the import applied.

A new character creates the PC first and then runs the same steps.

### Diff on the clean model

The diff compares the current and imported sheets as JSON, per section, including name and max HP. Raw fields are not compared, since values such as `CurrentEvals` change on every save. List rows pair by their title field (`name`, `weapon`, `key`, `class`) and fall back to position, so a reordered list does not show as a full rewrite. Companions are listed as removed and added, and inventory is listed as replaced or unchanged.

### Inventory

Items and currency are replaced only if the PDF has at least one item or non-zero coin. A replaced item keeps its catalog link when its name matches exactly. A non-numeric amount becomes 1.

### Upload handling

The upload streams to a temp file while hashing it, limited to 64 MiB, and is deleted when the handler returns. The original PDF is never kept. One import runs at a time per server, using a non-blocking semaphore; a second one gets 429. With parsing under a second, a small group will rarely hit it.

### API

| Route | Purpose |
|---|---|
| `POST /api/pcs/{id}/sheet/import` | Upload for an existing PC. Owner only; `type: "pc"` only |
| `POST /api/sheet-imports` | Upload for a new character |
| `POST /api/sheet-imports/{id}/confirm` | Apply; returns `{pc_id, revision}` |
| `DELETE /api/sheet-imports/{id}` | Discard |

Someone else's import, or an expired one, returns 404. Confirming an applied import returns 409.

### Testing without personal data

The repository is public, so committed fixtures are synthetic: a small hand-made PDF for the reader, covering `/Type /Template`, `/TM`, stream values, and `/I` with `/Opt`, and invented field lists for the mapper. An end-to-end test against real sheets runs only when `MPMB_SAMPLES_DIR` is set.

## Risks / Trade-offs

- [A future MPMB version renames fields] → The sheet version is checked; anything other than 14.x imports with a warning, and the raw fields stay available.
- [A failure partway through a confirm leaves a partial import] → The steps are ordered so the old companions are removed last. The user can upload again.
- [pdfcpu is a large dependency used for a small part of it] → Reading is isolated in `mpmb.Read`, so it can be replaced without touching the mapper.
- [Heuristics on free text, such as a limited use's source or `pending`, can misread homebrew] → The result is visible in the diff and editable afterward.
- [Only two real sheets have been tested] → Unknown markers become warnings rather than failures.
- [Several companions and metric sheets have not been seen in a real file] → Covered with synthetic tests only.

## Migration Plan

No data migration. The store creates the collection's indexes on startup. Rollback leaves an unused collection, and imported sheets stay valid because they use the existing model.

Check after deploying: an upload near the limit passes through the Cloudflare tunnel, whose body limit has not been verified.
