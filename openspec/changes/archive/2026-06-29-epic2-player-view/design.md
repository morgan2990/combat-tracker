## Context

Epic 1 established the WebSocket infrastructure: rooms are created, clients connect with roles, and the full `RoomState` is broadcast on any change. However, the server's read loop discards all incoming messages, no player entities exist in the state, and the `PlayerView` component is a placeholder. This epic activates the player side of the system.

## Goals / Non-Goals

**Goals:**
- WS message dispatcher routing player actions to room mutations
- Player character setup flow (Max HP + Initiative) with reconnection re-linking
- Server-side initiative sorting (`State.Entities` always sorted descending)
- Ownership-enforced `update_entity` and `add_companion` actions
- Full phone-optimized `PlayerView`: initiative list, hybrid HP editor, condition toggles, companion management

**Non-Goals:**
- DM controls (turn advancement, creature management) — Epic 3
- Initiative editing after combat starts — Epic 3
- Removing companions (DM action) — Epic 3
- Free-form condition text — predefined list only

## Decisions

### 1. Message dispatcher replaces the discard loop

**Decision:** Parse every incoming WS message as `{ "type": string, ...payload }` and route to the appropriate room method. Unknown message types are silently ignored. The read loop remains the single goroutine per connection; no additional goroutines are introduced.

**Rationale:** Keeping dispatch in the read goroutine avoids concurrency complexity. Each action method acquires the room mutex, mutates state, and calls `BroadcastState` — the same pattern already used for connect/disconnect.

---

### 2. Server-side sorting — sort slice in-place after every mutation

**Decision:** After `SetupCharacter` or `AddCompanion`, sort `State.Entities` descending by `initiative` using a stable sort (preserves insertion order for equal initiatives). `active_index` always refers to the position in this sorted slice.

**Rationale:** If sorting were done on the frontend, `active_index` in the broadcast state would be meaningless (it would point into an unsorted slice). A single sorted slice keeps state unambiguous for all clients.

**When sorting is suppressed:** Once `is_started = true` (set by DM in Epic 3), the order is frozen. Sorting is skipped on any mutation while combat is active, preserving the locked turn order.

---

### 3. Reconnection re-linking in ValidateAndRegister

**Decision:** When a player connects with a name that matches an existing entity in `State.Entities` (by `name` and `type == "player"`), the server updates that entity's `session_id` to the new session and registers the client normally. No name conflict is raised.

**Rationale:** The existing `isNameTaken` check only looks at active `Clients`. Since disconnected players are removed from `Clients`, their name is already free. The re-link just adds the additional step of updating the entity. This makes reconnection seamless without any client-side token.

**Client-side detection:** After connecting, the React client checks whether any entity in the broadcast state matches `name === myName && type === "player"`. If found, the player is already in the tracker and the setup form is skipped.

---

### 4. Authorization: session ownership + companion owner_id

**Decision:** `UpdateEntity` checks two conditions:
1. `entity.SessionID == client.SessionID` — the entity belongs to this session (own player entity)
2. `entity.OwnerID == myEntityID && entity.Type == "companion"` — the entity is a companion owned by this player

Where `myEntityID` is resolved by finding the entity in `State.Entities` with `session_id == client.SessionID`. Any other combination is rejected with no state change and no broadcast.

**Rationale:** Session IDs are server-generated and never exposed to other clients, so they cannot be spoofed over the WS protocol.

---

### 5. HP editor: hybrid delta + direct set (client-only)

**Decision:** The HP editor shows `[-10][-5][-1] [HP display] [+1][+5][+10]`. Tapping the HP display opens an inline numeric input for direct set. Both paths call the same `update_entity` WS message with the resolved `current_hp`. No new server protocol needed.

**Rationale:** Delta buttons cover the common case (took 5 damage) without typing; direct set covers large changes without repeated taps. Both map to the same server message, keeping the backend simple.

---

### 6. Conditions: predefined list, full array replacement

**Decision:** The client maintains a fixed list of 8 D&D 5e conditions: Prone, Stunned, Poisoned, Blinded, Frightened, Incapacitated, Restrained, Paralyzed. Toggling a condition rebuilds the full `conditions` array and sends an `update_entity` message. The server stores and broadcasts whatever array it receives (no validation of condition names).

**Rationale:** Full array replacement is simpler than add/remove delta operations and avoids ordering edge cases. The server stays agnostic about condition names, making it easy to add more conditions in the future without backend changes.

## Risks / Trade-offs

- **Reconnection TOCTOU** → Between `ValidateAndRegister`'s name-free check and the entity re-link, another client could claim the name. For a friends game this is not a real risk; the write lock on the room serializes both checks anyway.
- **Entity not found on `update_entity`** → If the entity ID is stale (e.g., DM removed a companion), the server silently ignores the action. The next broadcast will resync the client.
- **Companion orphaned on disconnect** → Companions persist by design (US2.3 AC5). If the owning player never returns, their companion stays in the tracker until the DM removes it (Epic 3). This is the intended behavior.
- **Sort stability for equal initiatives** → Go's `slices.SortStableFunc` preserves insertion order for ties, which is the fairest tiebreak.
