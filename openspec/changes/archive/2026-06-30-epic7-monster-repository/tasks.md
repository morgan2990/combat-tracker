## 1. Data Model and Backend Foundation

- [x] 1.1 Define `Monster` struct in `store/monster.go` with fields: `Name`, `MaxHP`, `SourceType`, `ReferenceURL`, `PDFObjectKey`, `FivEToolsID`, `SourceBook`
- [x] 1.2 Implement `UpsertMonster(m Monster)` in `store/monster.go` using ReplaceOne with upsert on `name`
- [x] 1.3 Implement `GetMonsterByName(name string)` in `store/monster.go` returning nil if not found
- [x] 1.4 Initialize the `monsters` collection handle in `store/mongo.go` `Init()` alongside the existing `entities` collection
- [x] 1.5 Extend `room.Entity` struct in `room/room.go` with `SourceType`, `ReferenceURL`, `PDFObjectKey` string fields

## 2. MinIO Integration

- [x] 2.1 Add `github.com/minio/minio-go/v7` to `go.mod` and run `go mod tidy`
- [x] 2.2 Create `store/minio.go` with a `MinioClient` singleton initialized from env vars `MINIO_ENDPOINT` (default `192.168.0.193:9001`), `MINIO_ACCESS_KEY` (default `usuario`), `MINIO_SECRET_KEY` (default `password`), `MINIO_BUCKET` (default `pdfs`)
- [x] 2.3 Implement `UploadPDF(name string, r io.Reader, size int64)` in `store/minio.go` uploading to key `monsters/{name}.pdf`
- [x] 2.4 Implement `StreamPDF(name string) (io.ReadCloser, error)` in `store/minio.go` fetching the object for streaming
- [x] 2.5 Call MinIO `Init()` in `main.go` on startup (non-fatal if MinIO is unavailable — log warning)

## 3. REST Endpoints

- [x] 3.1 Add `POST /api/monsters` handler in `api/handler.go` supporting both JSON (URL-type) and multipart (PDF-type) based on `Content-Type`
- [x] 3.2 Enforce 20 MB max size on multipart PDF upload via `http.MaxBytesReader`
- [x] 3.3 Add `GET /api/monsters/{name}` handler returning the monster document as JSON or 404
- [x] 3.4 Add `GET /api/monsters/{name}/pdf` handler that streams the MinIO object with `Content-Type: application/pdf`
- [x] 3.5 Register all three routes in `main.go` HTTP mux

## 4. WebSocket: add_creature Extension

- [x] 4.1 Extend the `add_creature` message struct in `ws/handler.go` to include `Quantity int`, `SourceType`, `ReferenceURL`, `PDFObjectKey` fields
- [x] 4.2 Update `room.AddCreature()` in `room/room.go` to loop `quantity` times (default 1), naming entities with suffix when quantity > 1
- [x] 4.3 Pass statblock reference fields through `AddCreature` to each created `Entity`
- [x] 4.4 Ensure a single `BroadcastState()` call is made after all entities from one `add_creature` are inserted

## 5. Frontend: Monster Registration Form

- [x] 5.1 Create `frontend/src/components/MonsterForm.tsx` with fields: Name, Max HP, Source Type toggle (None / URL / PDF), Reference URL input (shown when URL), PDF file input (shown when PDF)
- [x] 5.2 On submit: POST to `/api/monsters` as JSON (URL/none type) or multipart (PDF type)
- [x] 5.3 Add route `/monsters/new` in `App.tsx` pointing to `MonsterForm`
- [x] 5.4 Add a nav link or button in `DMView` header pointing to `/monsters/new`

## 6. Frontend: DM Add Creature Form Updates

- [x] 6.1 Add `Quantity` number input (default 1, min 1) to the Add Creature form in `DMView.tsx`
- [x] 6.2 On name field blur: call `GET /api/monsters/{name}`, autofill Max HP if found, store `source_type`, `reference_url`, `pdf_object_key` in local form state
- [x] 6.3 Include `quantity`, `source_type`, `reference_url`, `pdf_object_key` in the `add_creature` WS message sent on form submit
- [x] 6.4 Clear statblock state when the name field is cleared or changed

## 7. Frontend: Statblock Drawer

- [x] 7.1 Update `Entity` interface in `types.ts` to include `source_type`, `reference_url`, `pdf_object_key` string fields
- [x] 7.2 Create `StatblockDrawer.tsx` component accepting `entity: Entity` and `onClose: () => void` props; renders `<img>` for URL type, `<embed>` for PDF type
- [x] 7.3 Add statblock icon button to the `EntityRow` in `DMView.tsx`, shown only when `entity.source_type` is non-empty and `entity.type === "creature"`
- [x] 7.4 Wire icon click to toggle drawer open/close; track `openDrawerEntityId` in `DMView` state so only one drawer is open at a time
- [x] 7.5 Lazy-render drawer content (mount the inner `<img>` or `<embed>` only when drawer is open)
