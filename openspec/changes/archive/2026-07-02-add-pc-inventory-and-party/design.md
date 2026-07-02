## Context

PCs are persistent documents in the `pcs` Mongo collection (`store/mongo.go`), owned via `owner_user_id`, and managed through `POST /api/pcs` / `PUT /api/pcs/:id` / `GET /api/pcs/:id` (`api/handler.go`). They are independent of any Room: a Room's live `RoomState.entities` is an ephemeral, WebSocket-synced combat view that references PCs but doesn't store their persistent data.

There is currently no `items`, `currency`, or `party` concept anywhere in the codebase (backend or frontend), and no character-sheet-style UI or reusable modal/overlay component — PCs are only touched via the creation/edit form and a flat list in Dashboard.

## Goals / Non-Goals

**Goals:**
- Let a PC's owner track a personal list of items (name + quantity) and a 5-tier currency purse (pp/gp/ep/sp/cp).
- Let any user group PCs (potentially owned by different users) into a named Party with a single pooled currency value that any member's owner can adjust.
- Provide a UI surface for both, reachable from Dashboard and from an active combat room, without touching the live combat sync path.

**Non-Goals:**
- No item metadata beyond name + quantity (no weight, value, equipped state, rarity, etc.).
- No shared/communal item list at the party level — items always belong to exactly one PC.
- No general-purpose character-sheet view — only the inventory panel itself.
- No changes to `Entity`, `RoomState`, or WebSocket message types.

## Decisions

### 1. Inventory is fields on the PC document, not a separate entity
`items: []Item{name, quantity}` and `currency: Currency{pp, gp, ep, sp, cp}` are added directly to the `PC` struct (`store/mongo.go`) and `PC` interface (`types.ts`), flowing through the existing `CreatePC`/`UpdatePC` methods and `POST /api/pcs` / `PUT /api/pcs/:id` handlers.
- **Alternative considered**: a standalone `Inventory` entity referenced by PC. Rejected — inventory has no independent lifecycle or reuse case; it's strictly 1:1 with a PC, so a separate entity would only add an id, a collection, and an extra fetch for no benefit.

### 2. Party is a new standalone, user-agnostic entity
A new `parties` collection holds `{id, name, member_pc_ids: []string, currency: Currency}`. Party is not owned by a single user and is not scoped to a Room — any user can create one, and any user can add a PC (their own or another's) as a member. Any user who owns at least one member PC can edit the party (membership and pooled currency).
- **Alternative considered**: Party scoped 1:1 to a Room, or DM-owned with single-writer semantics. Rejected — the table's actual pattern is that any player might update the shared purse after a shopping trip, not just the DM; and a party persisting independently of any single room better matches an ongoing campaign.

### 3. Party pools currency only, never items
Items stay attached to the owning PC even when that PC is a party member: potions, scrolls, and similar items are consumed in the moment and often need to be in a specific character's hands for a rule interaction (e.g., using a potion as an action), so shared ownership would create real ambiguity about who currently has it. Currency is abstract and fungible, so pooling it doesn't create that ambiguity — matching the common "party fund" pattern at real tables.

### 4. InventoryPanel is a new shared UI component, decoupled from combat sync
A single `InventoryPanel` component takes a `pcId`, fetches/updates that PC via the existing REST endpoints, and is mounted as an expandable panel/modal from two call sites: Dashboard's PC list, and an entity row inside `DMView`/`PlayerView` during an active combat session. Because inventory lives on the persistent PC record (not the ephemeral `Entity`), opening/editing it mid-combat never touches `RoomState` or the WebSocket connection — it's a plain, independent REST round trip.
- **Alternative considered**: route inventory edits through the room's WebSocket so all connected clients see live updates. Rejected for this change — no other client is watching another player's private inventory in real time today, and it would couple an orthogonal feature to combat-sync infrastructure for no clear benefit. Can be revisited later if live multi-viewer sync becomes a real need.

### 5. Editable item list follows the EncounterForm pattern
The items list in the PC form/panel is edited as a local array in `useState` (add row / update-by-index / remove-by-index), submitted as a single `PUT /api/pcs/:id` on save — mirroring `EncounterForm.tsx`'s `monsters` array handling rather than per-row API calls.

## Risks / Trade-offs

- **[Risk] Any user can add any PC to any Party, and any party member's owner can spend pooled currency** → **Mitigation**: acceptable for this change since it matches the app's existing trust model (single shared table, no granular permission system exists elsewhere); revisit if the app grows multi-table/public-room support.
- **[Risk] Concurrent currency edits to a Party (two players adjusting the pool at once) could race** → **Mitigation**: treat `PUT` on party currency as a full overwrite (last-write-wins), consistent with how `UpdatePC` already works; acceptable given low real-world concurrency (one table, turn-based updates).
- **[Risk] New modal/overlay pattern has no precedent in this codebase** → **Mitigation**: keep `InventoryPanel` a self-contained component with inline styles matching existing conventions (`React.CSSProperties`, dark theme palette), so it doesn't require pulling in a UI library.

## Migration Plan

- Additive schema change only: new fields on `PC` documents default to empty (`items: []`, `currency: all zeros`) for existing records; no backfill required.
- New `parties` collection created on first use (following the existing `ensure*Index` pattern in `store/mongo.go`).
- No breaking changes to existing endpoints or message formats; rollback is simply reverting the code, since no destructive migration occurs.

## Open Questions

- Should there be any UI affordance for quickly moving an item from a PC's list into someone else's (a "give item" action), or is that left to players narrating it and manually editing both lists? (Left out of scope for this change; can be a fast-follow.)
