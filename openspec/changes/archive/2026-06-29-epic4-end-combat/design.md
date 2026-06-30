## Context

The room has a `Room.EndCombat()` method gap: Epic 3 can start combat and advance turns but has no termination path. The DM panel renders "Start Combat" / "Next Turn" / "End Combat" conditionally on `is_started`. All clients already handle `is_started = false` correctly via existing PlayerView and DMView rendering logic — no client-side state machine changes are needed beyond adding the button.

## Goals / Non-Goals

**Goals:**
- One backend method that filters entities and resets combat fields atomically under the room mutex
- One new WS dispatch case (`end_combat`)
- Inline confirmation UI in DMView with a single boolean state flag

**Non-Goals:**
- Resetting player HP, conditions, or dead flags (explicit healing remains DM responsibility)
- Archiving or logging the completed encounter
- Multi-step cleanup wizards

## Decisions

### Entity filter: two-pass approach

**Decision:** `EndCombat` builds the survivor list in two passes:
1. Collect all player entities into a set; record their IDs.
2. Walk all entities: keep players; keep companions whose `owner_id` is in the player ID set; discard everything else.

**Rationale:** A single-pass filter cannot determine companion eligibility without first knowing which player IDs survive. Two passes are O(n) and trivially correct for the small entity counts this app handles.

**Alternative considered:** Filter companions by checking whether any active WebSocket session holds the owner's session_id. Rejected — companions persist through player disconnects (Epic 2 spec), so session presence is the wrong signal. Owner entity existence is the correct predicate.

### Inline confirmation: single `confirmingEnd` boolean state

**Decision:** DMView holds a `confirmingEnd bool` state. "End Combat" click sets it to `true`, rendering the confirmation row. "Cancel" resets it to `false`. "Yes, End Combat" sends the message and resets it to `false`.

**Rationale:** No modal component, no portal, no z-index wrestling. The confirmation row appears inline below the combat controls, replacing the "End Combat" button. This matches the existing pattern of conditional rendering in DMView and requires no new component.

### Combat state reset values

| Field | Value after EndCombat |
|---|---|
| `is_started` | `false` |
| `round` | `0` |
| `active_index` | `0` |

`active_index = 0` is safe because with `is_started = false` the frontend never renders an active indicator regardless of its value.

## Risks / Trade-offs

- [DM double-clicks "Yes, End Combat" before broadcast returns] → The second click sends a second `end_combat` message; the server ignores it because `is_started` is already false. No state corruption.
- [All players have disconnected when DM ends combat] → Entity list has no players; all companions are orphaned; result is an empty entity list. Valid and expected — the DM is cleaning up an abandoned room.

## Open Questions

None.
