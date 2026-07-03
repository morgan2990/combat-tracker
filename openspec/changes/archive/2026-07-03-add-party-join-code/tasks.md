## 1. Data model and code generation

- [x] 1.1 Add `JoinCode` field to `store.Party`
- [x] 1.2 Add a `newPartyJoinCode()` helper reusing the room-code alphabet/algorithm (`crypto/rand` over `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, 5 characters)
- [x] 1.3 Update `store.PartyStore.CreateParty` to generate a `JoinCode`, checking uniqueness against the `parties` collection, retrying on collision up to a small attempt cap
- [x] 1.4 Add lazy backfill: when `GetPartyByID` or `UpdateParty` loads a party with an empty `JoinCode`, generate and persist one before returning it

## 2. Self-join endpoint

- [x] 2.1 Add `POST /api/parties/join` handler accepting `{ join_code, pc_id }`
- [x] 2.2 Resolve the party by `join_code`; respond HTTP 404 if no match
- [x] 2.3 Verify the requester owns `pc_id`; respond HTTP 403 if not
- [x] 2.4 Add `pc_id` to `member_pc_ids` and persist; if already present, return success without modifying the document
- [x] 2.5 Register the route in `main.go`

## 3. Dashboard UI

- [x] 3.1 Display each party's `join_code` on `PartyCard`
- [x] 3.2 Add a "join a party" form to Dashboard's Parties section (code input + PC picker from the user's own characters), following the existing "Join a new room" layout
- [x] 3.3 Wire the form to `POST /api/parties/join` and refresh the party list on success

## 4. Verification

- [x] 4.1 Manually verify: a second user, starting with no relationship to a party, joins it using a shared code and one of their own PCs
- [x] 4.2 Manually verify: submitting a valid code with a PC not owned by the requester is rejected
- [x] 4.3 Manually verify: submitting an unknown code is rejected with 404
- [x] 4.4 Manually verify: re-submitting an already-joined PC/code pair succeeds without creating a duplicate entry
- [x] 4.5 Manually verify: an existing party created before this change receives a join code on first fetch
- [x] 4.6 Run existing `store` and `api` package tests to confirm no regressions to `party-management`
