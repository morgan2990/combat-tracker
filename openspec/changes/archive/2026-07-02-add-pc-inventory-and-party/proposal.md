## Why

Players have no way to track what their character is carrying or how much money they have, so this bookkeeping happens entirely outside the app (on paper or in a separate tool). Tables also commonly share a communal pool of coin (adventuring funds, loot split) with no good way to represent that today.

## What Changes

- Add `items: {name, quantity}[]` and `currency: {pp, gp, ep, sp, cp}` fields directly to the PC record, managed through the existing PC create/update endpoints.
- Add a new **Party** entity: a standalone, user-agnostic named container that references member PCs (potentially owned by different users) and holds a single pooled `currency` value. Any user who owns a member PC can edit party membership and the pooled currency.
- Parties do **not** hold shared items — only currency is pooled. Personal items always stay attached to a specific PC (see design.md for rationale).
- Add a new shared `InventoryPanel` UI component, launchable both from the Dashboard's PC list and mid-combat from DMView/PlayerView, that reads/writes a PC's items and currency via REST (independent of the live WebSocket room-sync state).
- Add a new "Parties" section in Dashboard for creating parties, adding/removing member PCs, and viewing/editing the pooled currency.

## Capabilities

### New Capabilities
- `pc-inventory`: Personal item list and currency purse attached to a PC, managed via the existing PC endpoints, and viewable/editable through a new InventoryPanel UI reachable from Dashboard and from an active room.
- `party-management`: Standalone Party entity grouping PCs across owners, with a shared pooled currency editable by any member's owner, managed through new CRUD endpoints and a new Dashboard section.

### Modified Capabilities
- `player-profile-management`: `POST /api/pcs` and `PUT /api/pcs/:id` payloads/responses now include `items` and `currency` fields on the PC document.

## Impact

- **Backend (Go)**: `store/mongo.go` (PC struct gains `items`/`currency`; new `parties` collection + CRUD methods), `api/handler.go` (PC create/update handlers accept new fields; new Party handlers/routes).
- **Frontend (TS)**: `types.ts` (PC interface gains `items`/`currency`; new `Party` interface), new `InventoryPanel.tsx` component, `Dashboard.tsx` (new Parties section, launch point for InventoryPanel), `DMView.tsx`/`PlayerView.tsx` (launch point for InventoryPanel from an entity row).
- No changes to live combat sync (`Entity` type, WebSocket `RoomState` broadcast) — inventory and party data are fetched/updated out-of-band via REST.
