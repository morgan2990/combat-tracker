## Context

`player-profile-management` covers persistent PC/companion documents in the `pcs` MongoDB collection (`store.PC`, distinguished only by a `type: "pc" | "companion"` field and, for companions, a `parent_pc_id`). It currently has create, update, and fetch, but no delete — and the existing update path silently drops `shares_initiative` even though the request payload already carries it. Separately, `room.RefreshFromProfile` pulls profile changes into an already-live room entity, but only for `max_hp`.

## Goals / Non-Goals

**Goals:**
- Let a player edit a companion's name, max HP, and shares-initiative flag, and delete a companion, from the Edit Character screen.
- Let a player delete a whole character (PC), cascading to its companions, from the Dashboard.
- Fix `shares_initiative` being silently dropped on update.
- Keep the profile→live-room sync (`RefreshFromProfile`) consistent across all three editable companion fields, not just `max_hp`.

**Non-Goals:**
- No push-based sync (edits do not reach into a live room's in-memory state immediately) — sync stays pull-based, triggered by the client's existing `refresh_from_profile` message.
- No blocking or special-casing of delete when the PC/companion is currently instantiated in an active room — delete is unconditional, matching existing precedent (room/encounter/custom-monster delete).
- No new nested route (`/api/pcs/{id}/companions/{companion_id}`) — reuses the existing flat `/api/pcs/{id}` shape.

## Decisions

**1. Widen `RefreshFromProfile` to sync `Name` and `SharesInitiative`, not just `MaxHP` — still pull-based.**
Alternative considered: push-based sync, where `PUT /api/pcs/{id}` reaches into any live room(s) using that companion and broadcasts immediately. Rejected — it would require threading in-memory room-registry access into the HTTP API layer, a boundary this codebase has deliberately avoided (the existing `RefreshFromProfile` pull-on-reconnect pattern exists specifically to keep `*store.Store` out of routine room-mutation methods). Widening the existing pull loop to cover two more fields is a small, in-place change with no new architectural surface.

**2. Deleting a companion (or a PC) does not touch already-live room state — snapshot semantics.**
Alternative considered: reconnect-time cleanup that removes an orphaned live entity when its `PCID` no longer resolves, or blocking delete outright while the document is live in a room. Both were rejected: cleanup opens mid-combat removal semantics (turn order, conditions) for a rare sequence (disconnect, then delete, then someone notices); blocking requires the same API-layer-reaches-into-room-registry boundary crossing declined in (1). This matches how room/encounter/custom-monster delete already behave — unconditional, no live-state awareness.

**3. Flat route: `DELETE /api/pcs/{id}` and the existing `PUT /api/pcs/{id}` operate on a document by its own id, not nested under a parent.**
A companion is a `PC` row with `type: "companion"` in the same collection as top-level PCs. `GetPCByID` already resolves either type by its own id. Nesting (`/api/pcs/{id}/companions/{companion_id}`) would require a parent-ownership lookup plus a child lookup; the flat shape needs one lookup and one ownership check (`doc.OwnerUserID == callerUserID`), mirroring the existing `UpdatePC` handler almost exactly.

**4. `DELETE /api/pcs/{id}` is generic across both document types, with cascade delete of companions when deleting a `"pc"`-type document — not gated to companion-only.**
Alternative considered: gate the new endpoint to reject deletion of `"pc"`-type rows, deferring "delete a whole character" to a separate future stub (consistent with this codebase's history of small, single-slice changes). Rejected in favor of building it now: once the flat, type-agnostic route from (3) exists, restricting it back down to one type is the extra work, not the reverse — and it closes a real, user-visible gap (Dashboard's "My Characters" has Edit but no Delete) in the same pass rather than leaving a half-finished capability.

**5. Top-level PC delete is unconditional (consistent with Decision 2), with a client-side warning instead of a server-side block.**
Deleting a PC that's the credential checked on every room (re)join (`ws/handler.go` re-fetches `GetPCByID` on join/reconnect and closes with 4003 "forbidden" if it's gone) is a materially worse failure mode than a stale companion — a player who disconnects, deletes their character, then tries to rejoin is permanently locked out of that room. Blocking this server-side would reverse the boundary decision made in (1) and (2) for a narrow, deliberate-action edge case. Instead, the Dashboard delete-confirm dialog's copy calls out the risk explicitly, at zero architectural cost.

**6. `UpdatePC` gains a `sharesInitiative bool` parameter; the handler passes `body.SharesInitiative` through.**
`pcPayload` already decodes `shares_initiative` from the request body — the store method just never received or wrote it. This is a bug fix riding along with the edit-companion work, not new scope: without it, companion edits made through this same endpoint would silently fail to persist the shares-initiative flag.

## Risks / Trade-offs

- **[Risk]** A player disconnects mid-session, deletes their character, and cannot ever rejoin that room. → **Mitigation**: confirm-dialog warning copy (Decision 5); accepted as a rare, self-inflicted edge case rather than solved architecturally.
- **[Risk]** Cascade-deleting a PC's companions could surprise a player who forgot they had companions attached. → **Mitigation**: none planned beyond the existing confirm dialog pattern used elsewhere (room/encounter/custom-monster delete) — consistent with how destructive actions are already handled in this codebase.
- **[Risk]** `RefreshFromProfile` now syncs three fields instead of one; a room with many companions pays a slightly larger per-reconnect update. → **Mitigation**: negligible — same O(n) loop over already-fetched companion docs, just two more field writes per match.

## Open Questions

None outstanding — all four design questions were resolved during exploration before this proposal was written.
