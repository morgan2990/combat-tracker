## 1. Backend: tighten CreateRoom's JSON handling

- [x] 1.1 In `api/rooms.go`, replaced `json.NewDecoder(r.Body).Decode(&body) //nolint — empty body is fine` with a decode step that rejects any decode error except `io.EOF`. The shared `decodeJSON` helper was **not** reused as originally planned — it treats an empty body (`io.EOF`) as a decode failure too, which would have broken the spec'd "no body → defaults to 5e" scenario, so `CreateRoom` keeps a decode step that special-cases `io.EOF` as valid.
- [x] 1.2 N/A — `encoding/json` is still needed directly (for `json.NewDecoder`), plus `errors` and `io` were added for the `errors.Is(err, io.EOF)` check.
- [x] 1.3 `go build ./...` and `go vet ./...` both clean.

## 2. Verify

- [x] 2.1 `POST /api/rooms` with `{"edition":"5.5e"}` → HTTP 201, `edition: "5.5e"`. Confirmed via curl.
- [x] 2.2 `POST /api/rooms` with no body → HTTP 201, `edition: "5e"`. Confirmed via curl.
- [x] 2.3 `POST /api/rooms` with `{"edition":"3e"}` (unrecognized value) → HTTP 201, `edition: "5e"`. Confirmed via curl.
- [x] 2.4 `POST /api/rooms` with `{bad json` (malformed) → HTTP 400 `invalid json`, no room created. Confirmed via curl.
- [ ] 2.5 Close GitHub issue #8 referencing the merged change. (Left open until this change is actually merged.)
