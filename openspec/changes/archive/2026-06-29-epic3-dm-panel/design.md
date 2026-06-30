## Context

The Go backend has a `Room` struct with in-memory `RoomState` and a WebSocket dispatcher in `ws/handler.go`. Epic 2 established the pattern: DM-only messages are distinguishable only by the session's `role` field (stored on `Client`). The `sortEntities()` method currently skips sorting when `is_started` is true — that invariant changes in this epic. The React `DMView` is a read-only placeholder.

## Goals / Non-Goals

**Goals:**
- Full combat lifecycle: start, turn advance, round wrap
- Creature add/kill/remove with mid-combat sort safety
- DM override of any entity's stats, smart HP input client-side
- Greyed-out dead entities visible on all clients

**Non-Goals:**
- Persistent storage (rooms remain in-memory, lost on restart)
- DM-initiated player name change
- Automated death saves or status effects
- Multiple simultaneous DM connections

## Decisions

### `sortEntities()` always sorts; preserves active index by entity ID

**Decision:** Remove the `is_started` guard. `sortEntities()` always runs the sort. When `is_started` is true it additionally preserves `active_index` by recording the active entity's ID before the sort and scanning for it after.

**Rationale:** The user wants mid-combat additions and DM initiative overrides to re-sort the list. Tracking by ID is the only safe way to keep `active_index` valid after a sort that may shift every position.

**Alternative considered:** Keep frozen order but insert mid-combat additions at the correct index without a full sort. Rejected because DM initiative overrides would then require a separate re-sort path, leading to two code paths.

### DM role check instead of re-verifying dm_token per action

**Decision:** DM-only message handlers check `c.Role == "dm"` on the Client struct. No per-message token re-verification.

**Rationale:** The dm_token is verified once at WebSocket upgrade time. The session's role is set then and never changes — it is as authoritative as the token itself. Re-verifying per message would add latency and complexity with no security benefit (an attacker with an active DM session already has full control).

### `dm_update_entity` sends all fields, server ignores `name` for non-creatures

**Decision:** The WS message includes `name`, `current_hp`, `temp_hp`, `initiative`, `conditions`, and `dead`. The server applies `name` only when `entity.Type == "creature"`.

**Rationale:** A single message type is simpler to maintain than entity-type-specific variants. The server is the authority on which fields may change per type — the client does not need to know this rule.

### Smart HP delta is computed client-side before sending

**Decision:** The DM HP input field on the client parses the string: `+N` / `-N` → delta applied to current HP; bare integer → absolute. The resolved absolute value (clamped to `[0, max_hp]`) is sent as `current_hp`. The server always receives an absolute value.

**Rationale:** Keeping the server contract uniform (always absolute `current_hp`) avoids adding a `hp_delta` branch to the server. The client already knows `current_hp` from the latest `RoomState` broadcast, so it can resolve the delta locally.

### `remove_dead_creatures` filter: dead AND type == "creature"

**Decision:** The batch remove action only purges entities satisfying both conditions. Player and companion entities — even if marked dead — are excluded and must be removed individually.

**Rationale:** Players may be at 0 HP but not dead (making death saves). Companions may be dead but their owner may want to revive them later. The DM retains explicit control over non-creature entities.

### `active_index` adjustment after `remove_entity`

**Decision:**
- Removed index < active_index → decrement active_index
- Removed index == active_index → keep active_index (now points to the successor; wrap to 0 if was last)
- Removed index > active_index → no change

**Rationale:** This keeps the active turn on the "next" entity when the active one is removed, which is the most natural DM expectation.

## Risks / Trade-offs

- [ID-based sort preservation adds O(n) scan after every sort] → Acceptable: rooms have at most ~20 entities in typical play. No optimization needed.
- [DM removes the only entity mid-combat] → `active_index` is set to 0 but `Entities` is empty; frontend must guard against `entities[active_index]` being undefined.
- [Two DM tabs open simultaneously] → Only one DM token exists per room but nothing prevents the same token holder from opening two tabs. Both would hold DM sessions and the last write wins. Accepted as out of scope; the game is a personal tool for a small group.

## Open Questions

None — all design decisions are resolved.
