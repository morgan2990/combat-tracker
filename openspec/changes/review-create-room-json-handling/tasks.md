## 1. Backend: tighten CreateRoom's JSON handling

- [ ] 1.1 In `api/rooms.go`, replace `json.NewDecoder(r.Body).Decode(&body) //nolint — empty body is fine` with `if !decodeJSON(w, r, &body) { return }`, matching every other handler's pattern.
- [ ] 1.2 Remove the now-unused `encoding/json` import from `api/rooms.go` if `decodeJSON`'s removal of the direct `json.NewDecoder` call leaves it unused (check remaining uses first — `writeJSON`'s call site doesn't need it directly, but confirm).
- [ ] 1.3 Run `go build ./...` and `go vet ./...` to confirm it compiles.

## 2. Verify

- [ ] 2.1 Manually test: `POST /api/rooms` with a well-formed body (`{"edition":"5.5e"}`) still creates a room with that edition.
- [ ] 2.2 Manually test: `POST /api/rooms` with no body (empty) still creates a room defaulting to `"5e"`.
- [ ] 2.3 Manually test: `POST /api/rooms` with an unrecognized edition value in a valid JSON body (e.g. `{"edition":"3e"}`) still creates a room defaulting to `"5e"`.
- [ ] 2.4 Manually test: `POST /api/rooms` with a malformed (non-JSON) body now returns HTTP 400 and does not create a room.
- [ ] 2.5 Close GitHub issue #8 referencing the merged change.
