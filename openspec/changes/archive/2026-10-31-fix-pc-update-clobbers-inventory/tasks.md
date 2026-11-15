## 1. Store layer

- [x] 1.1 Change `PCUpdateFields` in `store/mongo.go`: `Currency` and `SharesInitiative` become pointers, and `Items` treats `nil` as omitted.
- [x] 1.2 Make `UpdatePC` always set `name` and `max_hp`, and set the other fields only when present.
- [x] 1.3 Update `TestPCStore_UpdatePC` and add integration tests in `store/pc_integration_test.go`: omitted fields keep their values; an empty `items` slice clears the list.

## 2. API layer

- [x] 2.1 Make `Currency` and `SharesInitiative` pointers in `pcPayload` (`api/pcs.go`), keep the negative-currency check for a present value, and pass the fields through in `UpdatePC`. `CreatePC` and `CreateCompanion` treat an absent value as zero.
- [x] 2.2 Update `fakePCStore.UpdatePC` in `api/pcstore_fake_test.go` to mirror the store's semantics.
- [x] 2.3 Add handler tests in `api/pcs_handlers_test.go`: a `{name, max_hp}` update preserves `items`, `currency`, and `shares_initiative`; `"items": []` clears; a companion update persists `shares_initiative`.

## 3. Verification

- [x] 3.1 Run `go test ./...` and `npm test` in `frontend/`.
- [x] 3.2 In the running app: add items and coins to a character, change its name on the Edit Character screen, and confirm the inventory is intact.
