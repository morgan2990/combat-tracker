## Context

A Party's membership is edited via `PUT /api/parties/:id` with a full replacement `member_pc_ids` array; the requester must own at least one PC already in that array (or the party must currently have zero members). Dashboard's `PartyCard` component builds its "add member" picker from the viewing user's own PC list only, and a party only appears in a user's Dashboard once they already own a member PC in it (`GET /api/me` scopes party visibility to PCs the requester owns). The net effect: there is no path, in the UI, for a second user to add their own character to a party they didn't create or aren't already in.

Room joining already solves an equivalent problem: `room_id` doubles as a short, human-typeable code (`ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, 5 characters, `room/room.go:15,1017-1024`), generated with `crypto/rand` and shared out-of-band by whoever's running the room. This design reuses that pattern for parties.

## Goals / Non-Goals

**Goals:**
- Let any authenticated user add one of their own PCs to an existing party by presenting a short code, without needing an existing member to already know their PC's ID.
- Reuse the existing room-code generation pattern (alphabet, length, `crypto/rand`) rather than inventing a new format.
- Make the code visible to a party's current members so they can share it.

**Non-Goals:**
- Join-code rotation or expiry. A party has exactly one join code for its lifetime in this change.
- Removing or changing the existing `PUT /api/parties/:id` membership-editing rule (owns-a-current-member). Self-join is additive, not a replacement.
- Restricting who can read a party's join code beyond the existing party-document visibility rules (`GET /api/parties/:id` is already readable by any authenticated user who has the party's ID).

## Decisions

**1. Join code format and generation mirror room codes exactly.**
5 characters drawn from the same `idChars` alphabet (`ABCDEFGHJKLMNPQRSTUVWXYZ23456789` — uppercase letters and digits with visually ambiguous characters removed), generated with `crypto/rand`. Reusing the constant/helper keeps the two code formats visually and behaviorally identical for users.

**2. Collision checking happens against MongoDB, not an in-memory registry.**
Rooms check uniqueness against the in-memory room registry only (`room/room.go:908-923`) because rooms live in memory first. Parties have no equivalent in-memory registry — they're pure MongoDB documents — so `CreateParty` checks the generated code against the `parties` collection directly (retry with a fresh code on collision, capped to a small number of attempts before failing the request).

**3. Self-join is a new endpoint with its own authorization rule, not a change to the existing PUT path.**
`POST /api/parties/join` takes `{ join_code, pc_id }`. Authorization is: the requester owns `pc_id`. That's it — no requirement to already own a current member, unlike `PUT /api/parties/:id`. The two paths coexist: `PUT` remains the way an existing member manages membership (including removal, which a join code alone shouldn't grant), and `POST /join` is the self-service entry point for a new participant.

**4. Joining with a PC already in the party is idempotent, not an error.**
If `pc_id` is already in `member_pc_ids`, the endpoint returns success without modifying the document. Avoids a race where two rapid clicks (or a retried request) produce an error instead of just confirming membership.

**5. The join code rides on the existing party document, with no new access restriction.**
`JoinCode` is returned as part of the same party document `GET /api/parties/:id` and `POST/PUT /api/parties` already return. That endpoint is already readable by any authenticated user who has the party's ID (per the existing `party-management` spec) — adding one more field to an already-fully-readable document doesn't change that document's access model.

## Risks / Trade-offs

- **[Join code never expires or rotates]** Once shared, anyone holding the code can add PCs to the party indefinitely. → Accepted for this change, consistent with room codes having no expiry/rotation either; a rotation control can be added later if it becomes a problem in practice.
- **[Collision check costs a DB round trip per attempt]** Unlike rooms' cheap in-memory map lookup, each generation attempt is a MongoDB query. → Negligible at expected party-creation rates; retry loop capped (e.g. 5 attempts) rather than unbounded.
- **[Existing parties predate this field]** Parties created before this change have no stored `JoinCode`. → Generate and persist one lazily the first time such a party is fetched or updated, rather than requiring a separate migration script.

## Migration Plan

No feature flag or coordinated rollout needed. New parties get a `JoinCode` at creation. Existing parties without one get it generated and persisted the next time they're returned by `GetPartyByID`, `UpdateParty`, or `ListPartiesByMemberPCIDs` (the last of which backs the Dashboard's party list), so no backfill script is required and no party is ever returned without a usable code.
