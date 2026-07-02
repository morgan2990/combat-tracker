## 1. Backend: PC inventory fields

- [x] 1.1 Add `Items []Item` and `Currency Currency` fields (with `Item{Name string, Quantity int}` and `Currency{PP, GP, EP, SP, CP int}`) to the `PC` struct in `store/mongo.go`
- [x] 1.2 Update `CreatePC` to initialize `Items` as an empty slice and `Currency` fully zeroed
- [x] 1.3 Update `UpdatePC` (and its call site in `api/handler.go`) to accept and persist `Items` and `Currency`
- [x] 1.4 Add request validation: reject `PUT /api/pcs/:id` with any negative `Currency` field (HTTP 400)
- [x] 1.5 Confirm `GET /api/pcs/:id` and `GET /api/me` responses include the new fields (no code change expected if PC is serialized wholesale — verify)

## 2. Backend: Party entity

- [x] 2.1 Define `Party` struct in `store/mongo.go`: `ID`, `Name`, `MemberPCIDs []string`, `Currency Currency`
- [x] 2.2 Add `parties` collection wiring in `Init()`, following the existing `ensure*Index` pattern (unique index on `id`)
- [x] 2.3 Implement `CreateParty(name string) (*Party, error)`
- [x] 2.4 Implement `UpdateParty(id string, memberPCIDs []string, currency Currency) error`
- [x] 2.5 Implement `GetPartyByID(id string) (*Party, error)`
- [x] 2.6 Implement `ListPartiesByMemberOwner(ownerUserID string) ([]Party, error)` (parties containing at least one PC owned by this user) for use in `GET /api/me`

## 3. Backend: Party API handlers & routes

- [x] 3.1 Add `POST /api/parties` handler: create party, require auth, validate non-empty `name`
- [x] 3.2 Add `PUT /api/parties/:id` handler: validate requester owns a PC in current `member_pc_ids` (or party has no members yet), reject negative `Currency` values, save membership + currency
- [x] 3.3 Add `GET /api/parties/:id` handler
- [x] 3.4 Extend `GET /api/me` response to include the requesting user's parties (via `ListPartiesByMemberOwner`)
- [x] 3.5 Register new routes in the router setup alongside existing `/api/pcs` routes

## 4. Frontend: types & API client

- [x] 4.1 Add `Item { name: string; quantity: number }` and `Currency { pp: number; gp: number; ep: number; sp: number; cp: number }` types to `types.ts`
- [x] 4.2 Add `items: Item[]` and `currency: Currency` to the `PC` interface
- [x] 4.3 Add a `Party` interface (`id`, `name`, `member_pc_ids: string[]`, `currency: Currency`) to `types.ts`
- [x] 4.4 Add `parties: Party[]` (or equivalent) to `MeResponse`
- [x] 4.5 Add fetch helpers for `POST/PUT/GET /api/parties*` alongside existing PC fetch helpers (inline `fetch()` calls in `Dashboard.tsx`/`InventoryPanel.tsx`, matching this codebase's existing convention — no separate API-client module exists anywhere to extend)

## 5. Frontend: InventoryPanel component

- [x] 5.1 Create `InventoryPanel.tsx`: takes a `pcId`, fetches the PC via existing REST endpoint, renders editable `items` list (add/update-by-index/remove-by-index in local `useState`, following `EncounterForm.tsx`'s pattern) and `currency` fields
- [x] 5.2 Wire panel save to submit the full updated PC (`items` + `currency`) via a single `PUT /api/pcs/:id`
- [x] 5.3 Style as a dismissable overlay/modal (new pattern — no existing modal component to reuse), consistent with the app's existing dark theme inline styles
- [x] 5.4 Add an "Inventory" launch action to each PC row in `Dashboard.tsx`, opening `InventoryPanel` for that PC
- [x] 5.5 Add an "Inventory" launch action to a PC's entity row in `DMView.tsx` and `PlayerView.tsx`, opening `InventoryPanel` for that PC's `pc_id` (confirm the panel makes no WebSocket calls and does not read/write `RoomState`)

## 6. Frontend: Parties section in Dashboard

- [x] 6.1 Add a "Parties" section to `Dashboard.tsx` listing the user's parties (from `me.parties`)
- [x] 6.2 Add a create-party form (name only) posting to `POST /api/parties`
- [x] 6.3 Add UI to add/remove member PCs on a party (submitting the full `member_pc_ids` array via `PUT /api/parties/:id`)
- [x] 6.4 Add editable currency fields for the party's pooled `currency`, submitting via `PUT /api/parties/:id`

## 7. Verification

- [x] 7.1 Manually verify: create a PC, add items and currency via InventoryPanel from Dashboard, reload, confirm persistence
- [x] 7.2 Manually verify: open InventoryPanel mid-combat from DMView/PlayerView, confirm no WebSocket traffic is triggered and live combat state (HP, initiative, conditions) is unaffected
- [x] 7.3 Manually verify: create a party, add two PCs owned by different test users, confirm both owners can edit the pooled currency and a non-member owner gets HTTP 403
- [x] 7.4 Manually verify: negative currency values are rejected on both PC and Party update endpoints
