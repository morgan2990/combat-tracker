## Why

A DM running a session has no way to see or adjust a party's collective inventory from within a room — they'd have to ask each player individually, and there's no way to make a covert change (e.g. an item stolen off a sleeping character) without the player noticing immediately. The DM Panel already has a half-built entry point for this (a bag icon on every player row that opens `InventoryPanel`), but it 404s for any PC the DM doesn't personally own, because the underlying endpoint enforces strict owner-only access with no notion of "DM of this player's room."

## What Changes

- A `Party` can now be "claimed" by a `Room`: the first time a player whose PC belongs to a party finishes character setup in an unclaimed room, that room links to that party for the rest of the session. Unaffiliated PCs (no party, or a second party's members) joining the same room neither claim it nor block a later claim — they're simply not covered by the panel described below.
- New DM-only room-scoped endpoints expose an aggregated view of the linked party's inventory: every member's items in a flat list (one row per item instance, tagged with owner), the party's pooled currency, and the sum of members' personal currency (narrowed to a single member's when the DM filters the view by player).
- The DM can filter the aggregated list by party member, and can add, edit, or remove any member's items and personal currency directly from this view.
- These edits are deliberately silent: they go through plain REST against the persisted PC document, the same way player-initiated inventory edits already do, so nothing is pushed to the player's client and nothing appears until they next look at their own inventory.
- **Fix**: the 🎒 icon on a player's entity row in DMView is enabled only when that PC is a member of the room's currently linked party, and opens `InventoryPanel` via the room-scoped endpoints described above.

## Capabilities

### New Capabilities
- `party-room-claiming`: a room automatically links to a party the first time one of that party's members completes character setup in the room; the link is not broadcast to players and does not affect any other party's ability to be unaffiliated in the same room.
- `dm-party-inventory`: DM-only, room-scoped read/write access to a linked party's aggregated inventory (items across all members, pooled currency, personal currency), including per-member filtering and the entity-row shortcut into a single member's view.

### Modified Capabilities
- `room-persistence`: the persisted room snapshot gains an optional `party_id` field so a room's party claim survives a server restart.

## Impact

- **Backend**: `room.Room` struct gains a `PartyID` field; `ws/handler.go`'s `setup_character` case gains claim-attempt logic; new REST handlers (likely in a new or extended `api/rooms.go`) for the aggregated view and per-member edits; `store.RoomSnapshot` and its Mongo (de)serialization gain `party_id`.
- **Frontend**: new DM panel component for the aggregated/filterable party inventory view (reachable from `DMNavColumn` or similar); `EntityRow`'s bag-icon gating and click target change.
- **No changes** to `/api/pcs/:id`, `party-management`, or `dm-entity-override` — this stays additive to those capabilities rather than modifying their existing contracts.
- **Depends on `add-party-join-code`**: this change's auto-claim logic assumes a party can actually contain multiple different players' PCs before a room session starts. That requires `add-party-join-code` to ship first.
