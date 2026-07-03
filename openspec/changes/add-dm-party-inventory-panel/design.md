## Context

Party is deliberately independent of Room: parties persist across sessions and aren't scoped to any single room. Inventory (`PC.Items`, `PC.Currency`) is REST-only and intentionally decoupled from the WebSocket/`RoomState` broadcast layer — opening or editing it never emits a WS message.

DMView already renders a 🎒 icon on every `pc`-type `EntityRow`, wired to open `InventoryPanel` via `GET /api/pcs/:id`. That endpoint enforces strict `OwnerUserID` matching (`api/pcs.go:96`, `:53`), so the icon 404s for any PC the DM doesn't personally own — which is effectively always. This change needs to bridge Party and Room without reversing the "parties are room-independent" decision, and without widening the existing strict ownership check on `/api/pcs/:id`.

## Goals / Non-Goals

**Goals:**
- Let a DM see and edit a linked party's aggregated inventory from within a room, without any player-facing indication that an edit happened.
- Establish the Room↔Party link automatically, with no new UI for picking/searching parties.
- Fix the dead 🎒 icon as part of the same permission model this change introduces.
- Keep the blast radius narrow: no changes to `/api/pcs/:id`'s existing owner-only semantics.

**Non-Goals:**
- Supporting multiple parties claimed by the same room simultaneously. A second party's members (or unaffiliated guests) simply aren't covered by the aggregated panel.
- A DM-facing UI to manually pick, change, or clear a room's linked party. (Explicitly deferred — the DM will get other room controls later that this can fold into.)
- Any notification, log, or audit trail visible to players when the DM edits their inventory. Silence is the point.
- Enforcing that a PC belongs to only one party. The data model doesn't guarantee this today; this change assumes it's true and takes the first match if it isn't.

## Decisions

**1. Room claims a party automatically, on first party-affiliated join, rather than via DM-picked link.**
A DM-facing "search all parties by name" picker was considered and rejected: parties have no owner/privacy concept today (any party is just a Mongo document, reachable by anyone who knows its ID), so a search-all endpoint would be new, unscoped surface area with no existing precedent to lean on. Auto-claim needs no new endpoint and no new UI — it hooks into the existing `setup_character` flow (`ws/handler.go:186-199`). When `rm.SetupCharacter(c.SessionID)` succeeds, if `Room.PartyID` is unset, look up the joining PC's party via the existing `store.PartyStore.ListPartiesByMemberPCIDs([]string{c.PCID})` (`store/party.go:101`, already used for a different purpose in `api/auth.go:113` — no new query needed). If found, set `Room.PartyID` and `rm.MarkDirty()`. A PC with no party, or belonging to a party different from one that already claimed the room, is a no-op: it doesn't claim, and it doesn't block a later claim.

**2. `Room.PartyID` lives outside the broadcast `RoomState`, not inside it.**
Players have no reason to know which party (if any) is linked — it's DM-panel-only metadata. Keeping it off the `RoomState` struct means claiming never triggers or requires a `BroadcastState()` call, consistent with inventory's existing "no WS chatter" property. It still needs to survive a restart, so it's added to `store.RoomSnapshot` and the Mongo (de)serialization path (a `room-persistence` delta, not `room-state`).

**3. New room-scoped REST endpoints, not a bypass on `/api/pcs/:id`.**
Considered widening `GetPC`/`UpdatePC`'s ownership check to also allow "requester owns a room whose linked party contains this PC." Rejected: that check would need to reverse-search across all of a user's rooms with no room in the URL to anchor it, and it would widen the risk surface of an endpoint every player relies on for strict isolation. Instead:
- `GET /api/rooms/{room_id}/party-inventory` — aggregated view (all members' items, pooled party currency, summed personal currency).
- `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` — edit one member's items/currency.
Both authorize via: requester is `room.OwnerUserID`, `room.PartyID` is set, and (for the PUT) `pc_id ∈ party.MemberPCIDs`. `/api/pcs/:id` is untouched.

**4. Aggregation and filtering happen server-side for the fetch, client-side for the filter.**
The `GET` returns the full aggregated payload (all members, all items, both currency figures) in one call; the player-filter dropdown in the UI slices that response client-side rather than re-fetching per filter change. The result set is bounded by party size, so this trades a slightly larger single payload for zero extra round-trips while filtering.

**5. Items are flat, one row per instance — no cross-owner merging.**
`{ owner_pc_id, owner_name, name, quantity }` rows, matching `PC.Items`' existing shape plus an owner reference. Two players both holding a "Healing Potion" are two rows, not one merged stack. This keeps the edit model trivial (every row has exactly one owning PC to write back to) and sidesteps the ambiguity of "whose stack gets decremented" a merged view would introduce.

**6. Currency: two figures, not one, and the second collapses under a filter.**
Unfiltered: `party_currency` (existing pooled `Party.Currency`, unchanged) shown alongside `members_currency_total` (sum of every member's personal `PC.Currency`, denomination-wise). Filtered to one player: `party_currency` stays as-is (it's not personal to any one member), `members_currency_total` becomes that member's own `PC.Currency`. This matches the pooled purse's existing looser edit permission (`api/parties.go:80-90`, any member-owning user) staying on its own path — this change only ever writes to personal `PC.Currency` via the new PUT, never to `Party.Currency`.

**7. The 🎒 icon fix rides on the same permission model, reusing `InventoryPanel`.**
`EntityRow` gates the icon's enabled state on "this PC is in the room's linked party" instead of always-on. When clicked, it opens `InventoryPanel` sourced from the new room-scoped `GET` (single-member slice) instead of `GET /api/pcs/:id`. The component itself doesn't need new UI — just a swapped fetch source depending on whether it's opened by the PC's own owner (existing `/api/pcs/:id` path, unchanged) or by the room's DM (new path).

## Risks / Trade-offs

- **[Silent edits are the point but also the danger]** A DM fat-fingering a quantity or currency value has no undo and the player has no signal anything changed until they look. → Accepted as inherent to the feature (explicitly requested); no mitigation planned beyond normal client-side confirm-before-save UX on destructive edits (e.g. removing a row).
- **[First-join-wins claiming can pick the "wrong" party in a mixed table]** If a guest or a different party's PC happens to connect first, that's who gets claimed, silently, with no DM override in this change. → Deferred: acceptable for now per explicit scope decision; a manual unlink/relink control is left for a future room-controls change.
- **[`PC` in two parties has undefined claim behavior]** `Party.MemberPCIDs` doesn't prevent a PC from belonging to multiple parties; `ListPartiesByMemberPCIDs` could return more than one hit. → Take the first result deterministically (e.g. by a stable sort such as party ID) and don't treat it as an error; document the assumption in the spec rather than adding new validation to `party-management`.
- **[Aggregated payload grows with party size]** Not a concern at expected table sizes (single-digit party members); no pagination planned.

## Migration Plan

No data migration needed — `PartyID` is a new optional field, absent on existing in-flight rooms (defaults to unset/claimable on next join) and absent on existing Mongo snapshot documents (decodes as unset). No backfill required. Ship backend (claim logic + new endpoints) and frontend (panel + icon fix) together; no feature flag needed since the surface is entirely new/additive.

## Open Questions

- Should there be any DM-visible indicator in the room (outside this change's scope) that a party is or isn't currently linked, given there's no manual control yet? Left for the future room-controls work mentioned in Non-Goals.
