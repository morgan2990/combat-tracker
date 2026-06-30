## Why

The monster repository (Epic 7) supports manual DM entry only — one creature at a time. To make the autocomplete search (Epic 12) useful from day one, the database needs a full compendium of creatures for both 5e and 5.5e editions. A CLI scrubber tool that bulk-imports from a locally cloned 5etools repository fills that gap without requiring UI work or a running server.

## What Changes

- The `monsters` MongoDB collection gains three new fields: `edition`, `initiative_modifier`, and `is_custom`
- The unique index on `name` is replaced with a compound index on `{ name, edition }`, allowing the same creature to exist once per edition
- `source_book` and `five_etools_id` (already present on the struct) are formally scoped: `source_book` is a first-class display/filter field; `five_etools_id` is the URL slug fragment used to build the statblock reference URL
- A new standalone CLI tool (`cmd/scrubber/main.go`) reads all `bestiary-*.json` files from a local 5etools repository and bulk-upserts monsters into MongoDB, auto-generating `reference_url` per creature
- DM-created monsters (via `POST /api/monsters`) are updated to set `is_custom: true` and require an `edition` value

## Capabilities

### New Capabilities
- `monster-scrubber`: CLI tool that reads a local 5etools bestiary directory and bulk-upserts normalized monster documents into MongoDB for a given edition

### Modified Capabilities
- `monster-repository`: Schema extended with `edition`, `initiative_modifier`, and `is_custom` fields; upsert key changes from `{ name }` to `{ name, edition }`

## Impact

- **`store/monster.go`** — `Monster` struct gains `Edition`, `InitiativeModifier`, `IsCustom` fields; `UpsertMonster` filter changes to `{ name, edition }`; `GetMonsterByName` becomes ambiguous until Epic 9 introduces edition context on the room
- **`api/handler.go`** — `UpsertMonster` handler must set `is_custom: true` and require `edition` on DM-created monsters
- **`cmd/scrubber/main.go`** — new entry point, no server dependency, reads from local filesystem and writes to MongoDB
- **MongoDB index** — migration required: drop `{ name: 1, unique: true }`, create `{ name: 1, edition: 1, unique: true }`
- **Downstream** — Epic 12 Typesense sync hooks into `US8.1`/`US9.2`; Epic 9 room edition field will resolve the `GetMonsterByName` ambiguity
