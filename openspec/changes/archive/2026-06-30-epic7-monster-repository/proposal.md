## Why

The DM currently has to retype monster names and HP values every session, and has no quick access to statblocks during combat without leaving the app. A persistent monster repository eliminates that friction and brings statblock references directly into the combat tracker.

## What Changes

- New `monsters` MongoDB collection storing DM-registered creature templates with statblock references
- New REST endpoints: `POST /api/monsters`, `GET /api/monsters/:name`, `GET /api/monsters/:name/pdf`
- MinIO integration in the Go backend for PDF file storage and streaming
- `add_creature` WS message extended to accept statblock reference fields and a quantity
- Room `Entity` struct extended with `source_type`, `reference_url`, `pdf_object_key`
- New `/monsters/new` management route in the React SPA for registering monsters
- DM Add Creature form: monster lookup on blur (autofills Max HP), quantity field (auto-numbers duplicates)
- Statblock icon per creature entity in `DMView`; clicking opens a slide-out drawer
- Drawer renders `<img>` for URL-type statblocks or a proxied PDF embed for PDF-type

## Capabilities

### New Capabilities
- `monster-repository`: Persistent DM-owned monster templates stored in MongoDB with name, max_hp, source_type, statblock reference, and optional 5etools metadata fields for future auto-fetch
- `statblock-drawer`: Slide-out drawer in the DM panel that renders a creature's statblock as an image (URL) or embedded PDF (MinIO-proxied), lazy-loaded on first open

### Modified Capabilities
- `creature-management`: `add_creature` now accepts optional `source_type`, `reference_url`, `pdf_object_key` fields and a `quantity` field; when quantity > 1 creatures are named with an auto-number suffix; the room Entity carries statblock reference fields

## Impact

- **Backend**: New `store/monster.go` (MongoDB CRUD), new MinIO client package, two new REST routes, extended `room.Entity` struct, extended `add_creature` WS handler
- **Frontend**: New `MonsterForm.tsx` component and `/monsters/new` route, updates to `DMView.tsx` (lookup on blur, quantity field, statblock icon, drawer component), updated `types.ts` Entity interface
- **Dependencies**: MinIO Go client (`github.com/minio/minio-go/v7`) added to `go.mod`
- **Infrastructure**: MinIO instance assumed running and reachable; connection config via env vars (`MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`)
- **No breaking changes** to existing player or companion flows
