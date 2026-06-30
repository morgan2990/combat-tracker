## Context

Three quality-of-life gaps remain after Epic 4: the DM onboarding flow requires cURL knowledge; the Kill button doesn't zero HP causing a tracker/reality mismatch; and PlayerView has no visual distinction between unconscious and dead entities. All fixes are frontend-only — no new backend endpoints, WS messages, or data model fields are needed.

## Goals / Non-Goals

**Goals:**
- DM can create and join a room entirely from the browser
- Kill button atomically sets dead + HP=0 in one WS message
- Both views distinguish three entity states: Alive, Unconscious (HP=0, not dead), Dead (dead flag)

**Non-Goals:**
- Adding a server-side "unconscious" field — derived state is sufficient
- Automatic death on HP reaching 0 — DM confirmation via Kill remains required
- Resetting HP on Revive — HP stays at 0 (Unconscious) until explicit DM healing

## Decisions

### US5.1 — API call lives in JoinScreen, not App.tsx

**Decision:** `JoinScreen` calls `fetch('POST /api/rooms')` internally and passes the result to `onJoin`. App.tsx's `onJoin` signature is unchanged.

**Rationale:** The API call is a UI concern (loading state, error display, button disable). Lifting it to App.tsx would add a new async flow to a component that currently only handles WS connection. Keeping it in JoinScreen isolates the change.

**DM tab layout:** Two visually separated sections:
1. **Create New Room** — name field + "Create New Room" button. On click: disable button, fetch API, on success call onJoin; on failure show error inline.
2. **Rejoin Existing Room** — name + room code + DM token + "Rejoin Room" button. Calls onJoin directly (existing flow).

The name field is shared between both sections (one input, used for either action).

### US5.2 — Kill sends current_hp: 0 alongside dead: true

**Decision:** The Kill onClick in `EntityRow` (DMView) sends `sendUpdate({ dead: true, current_hp: 0 })`. Revive sends `sendUpdate({ dead: false })` only — HP is not touched.

**Rationale:** The server already accepts `current_hp` and `dead` in the same `dm_update_entity` message. No protocol change needed. Keeping Revive HP-neutral preserves the explicit-healing design from Epic 3.

### US5.3 — Vital state helper function

**Decision:** Introduce a shared helper `entityVitalState(entity): 'alive' | 'unconscious' | 'dead'`:
```
dead === true          → 'dead'
current_hp === 0       → 'unconscious'
otherwise              → 'alive'
```
Used in both `PlayerView` (tracker rows + editor panel header) and `DMView` (EntityRow). Prevents the condition from being duplicated across multiple render locations.

**Placement:** Defined inline in each component (not a shared module) — it's two lines and the components are already separate files. A shared utils module is premature for this project scale.

### Visual treatment table

| State | Background | Text | Badge |
|---|---|---|---|
| Alive | white (or role highlight) | normal | — |
| Unconscious | `#fff8e1` amber | `#666` dim | `😵 Unconscious` |
| Dead | `#f0f0f0` grey | `#aaa` | `💀 Dead` |

DMView already implements Dead. Only the Unconscious row needs adding to DMView; PlayerView needs both.

## Risks / Trade-offs

- [DM clicks "Create New Room" while API is slow] → Button disabled during fetch; error shown inline if request fails. No timeout handling — the `fetch` will use browser defaults (~2 min). Acceptable for a LAN game.
- [Entity HP reaches 0 via Revive and DM forgets to heal] → Entity stays Unconscious indefinitely. This is intentional — the DM is responsible for tracking recovery. The visual makes it obvious something needs attention.

## Open Questions

None.
