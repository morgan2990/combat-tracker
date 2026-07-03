## 1. Room data model and persistence

- [x] 1.1 Add `PartyID` (nullable) field to the `room.Room` struct
- [x] 1.2 Add `party_id` (nullable) field to `store.RoomSnapshot` and its Mongo (de)serialization
- [x] 1.3 Update the snapshot-building code path to include `PartyID` when persisting
- [x] 1.4 Update `GetOrRestoreRoom` to restore `PartyID` from a decoded snapshot, defaulting to unset when absent
- [x] 1.5 Confirm `PartyID` is never included in the `RoomState` struct or `BroadcastState()` payload

## 2. Party-room claiming logic

- [x] 2.1 In `ws/handler.go`'s `setup_character` case, after `rm.SetupCharacter` succeeds, look up the joining PC's party via `store.PartyStore.ListPartiesByMemberPCIDs([]string{c.PCID})`
- [x] 2.2 If the room's `PartyID` is unset and a party is found, set `Room.PartyID` and call `rm.MarkDirty()`
- [x] 2.3 If the room's `PartyID` is already set, or the joining PC has no party, leave the room's claim unchanged (no-op)
- [x] 2.4 If the joining PC belongs to more than one party, deterministically pick one (e.g. lowest party ID) rather than erroring
- [x] 2.5 Verify claiming does not call `BroadcastState()` and does not appear in any WS message sent to clients

## 3. DM party-inventory REST endpoints

- [x] 3.1 Add `GET /api/rooms/{room_id}/party-inventory` handler: verify requester is `room.OwnerUserID`, resolve `Room.PartyID → Party.MemberPCIDs`, fetch each member PC, and return the aggregated `items` list (with owner references), `party_currency`, and `members_currency_total`
- [x] 3.2 Handle the "no linked party" case with an empty/no-party response rather than an error
- [x] 3.3 Add `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` handler: verify requester is `room.OwnerUserID`, `Room.PartyID` is set, and `pc_id ∈ Party.MemberPCIDs`, then overwrite that PC's `items`/`currency` the same way `UpdatePC` does
- [x] 3.4 Reject negative currency values on the new PUT endpoint with HTTP 400, mirroring the existing check in `api/pcs.go`
- [x] 3.5 Register both routes in `main.go`
- [x] 3.6 Verify neither endpoint touches `/api/pcs/:id`'s existing ownership check or emits any WebSocket message

## 4. DM aggregated inventory panel (frontend)

- [x] 4.1 Build a new panel component that fetches `GET /api/rooms/{room_id}/party-inventory` and renders the flat item list, each row tagged with owner name
- [x] 4.2 Render `party_currency` and `members_currency_total` alongside the item list
- [x] 4.3 Add a per-player filter control; filtering slices the already-fetched response client-side (no re-fetch) and narrows `members_currency_total` to the selected member
- [x] 4.4 Add inline add/edit/remove controls per item row, saving via `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}`
- [x] 4.5 Add an editable field for each member's personal currency, saving via the same PUT endpoint
- [x] 4.6 Wire the panel into DMView (e.g. a new `DMNavColumn` section or overlay, following the `InventoryPanel` overlay pattern) with an appropriate empty state when the room has no linked party
- [x] 4.7 Confirm the panel never triggers a WebSocket message and that a player's own `InventoryPanel` shows the DM's edits only on its next independent fetch

## 5. Entity row inventory icon fix

- [x] 5.1 In `EntityRow`, gate the 🎒 icon's enabled state on whether the row's PC is a member of the room's linked party (requires the room's `party_id` and the linked party's `MemberPCIDs` to be available to the component)
- [x] 5.2 When enabled and clicked by the DM, open `InventoryPanel` sourced from the room-scoped `GET` (single-member slice) instead of `GET /api/pcs/:id`
- [x] 5.3 Confirm a player opening their own inventory still resolves via `/api/pcs/:id`
- [x] 5.4 Disable/hide the icon when the room has no linked party or the row's PC isn't a member of it

## 6. Verification

- [x] 6.1 Manually verify: two players from the same party join a room, the second player's join doesn't change the claim, a third unaffiliated guest's join is a no-op
- [x] 6.2 Manually verify: DM edits an item and currency via the aggregated panel; confirm the affected player's client shows no change until they reopen their own inventory
- [x] 6.3 Manually verify: a non-owner of the room gets 403/404 from both new endpoints
- [x] 6.4 Manually verify: restarting the server preserves a room's party claim (persistence round-trip)
- [x] 6.5 Run existing test suites for `room`, `store`, and `api` packages to confirm no regressions to `room-persistence`, `pc-inventory`, or `dm-entity-override` behavior
