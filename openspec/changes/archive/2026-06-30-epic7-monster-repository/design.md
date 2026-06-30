## Context

CombatApp is a Go backend + React SPA real-time combat tracker. State is synchronized over WebSocket with full `RoomState` broadcasts after every mutation. The REST layer is thin (room creation, entity profile CRUD). MongoDB holds player and companion profiles in a single `entities` collection. No file upload infrastructure exists today.

The app runs on a Raspberry Pi 4 (arm64). MongoDB 4.4.18 and MinIO are already deployed and running on the Pi via Portainer.

## Goals / Non-Goals

**Goals:**
- Persistent DM-owned monster templates in MongoDB with statblock references
- MinIO integration for PDF storage and proxied streaming
- Monster lookup on blur in the Add Creature form (autofills Max HP, carries statblock ref)
- Quantity field for bulk creature addition with auto-number suffix
- Statblock drawer in `DMView` for creatures with references

**Non-Goals:**
- Search-as-you-type autocomplete (deferred to a future story)
- Auto-import from 5etools data (deferred; `five_etools_id` + `source_book` stored now for later)
- Player-visible statblocks
- Monster editing or deletion UI
- Authentication on the PDF streaming endpoint (trusted local network app)

## Decisions

### Separate `monsters` collection, not extending `entities`
`entities` stores runtime player/companion profiles keyed by character name and owned by a player. Monsters are DM-owned templates that exist independently of any session. Mixing them would complicate all existing queries. A dedicated `monsters` collection with its own `store/monster.go` is the right boundary.

### MinIO for PDF storage over local disk
Local disk inside the container is lost on recreation. A named Docker volume solves persistence but adds operational burden (backup, size management). MinIO is already running on the Pi, S3-compatible, and the Go MinIO client is mature. PDFs are stored as objects under key `monsters/{name}.pdf`.

### Go proxies PDF stream — no presigned URLs
Presigned URLs would expose MinIO's port and endpoint to the browser, requiring CORS configuration and firewall changes. Proxying through the Go backend (`GET /api/monsters/:name/pdf`) keeps MinIO internal and requires no browser-side changes.

### `<img>` tag for URL statblocks, not `<iframe>`
Most reference sites (D&D Beyond, etc.) set `X-Frame-Options: DENY`. The DM will paste direct image URLs from 5etools book stats view. An `<img>` tag renders reliably with no embedding restrictions.

### Exact-match lookup on blur, not live search
A `GET /api/monsters/:name` exact lookup on the name field blur is sufficient for the current scale (dozens of saved monsters). The DM types the name correctly. Search-as-you-type is deferred and will add a `?q=` query endpoint and frontend debounce when needed.

### Quantity field handled server-side
When `quantity > 1`, the server loops and creates N entities with suffix-numbered names ("Goblin 1"… "Goblin N"). A single `RoomState` broadcast is sent after all N are inserted. This keeps the client simple — it sends one `add_creature` message regardless of quantity.

### `five_etools_id` + `source_book` in schema now, not in UI
Adding optional `omitempty` fields to the MongoDB document costs nothing and avoids a schema migration when the auto-fetch story lands. The form does not expose these fields; they will be populated by a future lookup endpoint.

## Risks / Trade-offs

- **MinIO connectivity**: if MinIO is unreachable, PDF upload fails at registration time and PDF streaming returns 404. Mitigation: surface a clear error message in the form; the URL path remains unaffected.
- **Name collision on quantity suffix**: if "Goblin 1" already exists in the room, the server creates a duplicate name. Mitigation: acceptable at this scale; DM can remove the duplicate. Future story can add conflict detection.
- **Large PDF streaming on Pi**: a 50MB PDF proxied through Go on the Pi could spike RAM. Mitigation: use `io.Copy` for streaming (no full buffer in memory); enforce a reasonable max upload size (e.g. 20MB) at the multipart handler.
- **No monster deletion UI**: the DM cannot remove saved monsters from the app. Mitigation: acceptable for now; direct MongoDB access can clean up. Deletion UI is a future story.

## Migration Plan

1. Deploy updated Go binary — new routes are additive, no existing routes change
2. The `monsters` collection is created lazily on first write (MongoDB creates collections on insert)
3. MinIO is already deployed — set env vars before starting the Go binary:
   - `MINIO_ENDPOINT=192.168.0.193:9000`
   - `MINIO_ACCESS_KEY=usuario`
   - `MINIO_SECRET_KEY=password`
   - `MINIO_BUCKET=pdfs`
4. No rollback complexity — new collection and routes are isolated; removing them restores prior state

## Open Questions

- What MinIO bucket name should be used? Suggest `combatapp-monsters` as default via env var.
- Should the PDF upload enforce a max file size? Suggest 20MB enforced at the Go handler.
