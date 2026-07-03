## Why

A Party is meant to be a cross-user grouping of characters, but there's currently no way for a second player to actually get their character into a party someone else created. The Dashboard's "add member" control only ever offers the viewing user's own characters, and a party is invisible in a user's Dashboard until they already own a member PC in it — so building a real multi-player party through the UI today is effectively impossible outside a single user adding their own several characters.

## What Changes

- A Party gains a short, shareable join code (5-character, mirroring the existing room-code format), generated when the party is created.
- Any authenticated user can self-join an existing party by submitting the join code together with one of their own PCs — no existing membership required, and no need for another member to already know their PC ID.
- Dashboard's Parties section displays each party's join code to its current members (for sharing) and adds a "Join a party" control (code input + PC picker) alongside the existing "Join a new room" flow, following the same interaction pattern.

## Capabilities

### Modified Capabilities
- `party-management`: party creation now also generates a join code; a new self-join path lets any user add one of their own PCs to a party by presenting that code, independent of the existing "owns a current member" membership-edit rule.

## Impact

- **Backend**: `store.Party` gains a `JoinCode` field; `store.PartyStore.CreateParty` generates and uniqueness-checks it against MongoDB; a new `POST /api/parties/join` handler resolves a party by code and adds the requester's PC to `member_pc_ids`.
- **Frontend**: `PartyCard` (`Dashboard.tsx`) displays the join code; Dashboard gains a join-by-code form.
- **Relationship to `add-dm-party-inventory-panel`**: that change's auto-claim logic depends on parties actually containing multiple players' PCs before a room session starts. This change is a prerequisite for `add-dm-party-inventory-panel` to be exercised with a real multi-player table, though the two are independently shippable.
