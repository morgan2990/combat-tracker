## Context

`DMView.tsx` renders the "▶ Start Combat" button (around line 469) inside the "Combat controls" row. Clicking it always sends `{ type: 'start_combat' }`. The server (`room.go` `StartCombat`, lines 129-132) silently no-ops the message if any `player` or `companion` entity has `initiative === null`, so the DM currently gets no feedback when blocked.

`RoomState.entities` (already available as a prop in `DMView.tsx`) carries `type` and `initiative` for every entity, which is all the data needed to mirror the server's check client-side — no new WS messages or fields required.

## Goals / Non-Goals

**Goals:**
- Grey out the Start Combat button and prevent its click handler from firing when the server would reject `start_combat`.
- Show the DM which named entities are still missing initiative, so they don't have to scan the full entity list.
- Keep the disabled predicate identical to the server's, so the button's enabled state never lies about what the server will accept.

**Non-Goals:**
- No backend/API changes. The server remains the source of truth and continues to validate independently (defense in depth — the frontend check is advisory only).
- No change to how `shares_initiative` companions get their initiative (already auto-copied server-side on `set_initiative`).
- No handling of `creature`-type entities — they're excluded from the gate today (server auto-rolls their initiative at combat start) and stay excluded.

## Decisions

**Predicate**: `entities.filter(e => (e.type === 'player' || e.type === 'companion') && e.initiative === null)`. Computed inline (e.g. via `useMemo` or a plain derived const) in `DMView.tsx` from the existing `roomState.entities` prop — no new state.

**Button disabling**: Add `disabled={pending.length > 0}` plus inline style following the existing app convention (`JoinScreen.tsx` `primaryBtn`, `PlayerView.tsx` Set button): `opacity: 0.45` and `cursor: 'not-allowed'` when disabled, normal otherwise. The `onClick` stays as-is since a `disabled` button won't fire it.

**Summary line**: A `<span>` rendered in the same flex row as the button (matching the existing inline-warning pattern used for the End Combat confirmation text at line ~492), shown only when `pending.length > 0`:
```
Waiting on initiative: {pending.map(e => e.name).join(', ')}
```
No icon/emoji decided yet — left as a small open question below.

**Why not a tooltip instead of an inline line**: the End Combat flow already uses an inline `<span>` for transient guidance text in this exact row, so reusing that pattern keeps the panel visually consistent rather than introducing a new hover-tooltip interaction.

## Risks / Trade-offs

- [Drift between client and server predicate] → Both read `entity.type` and `entity.initiative` directly off `RoomState`; as long as `StartCombat` in `room.go` isn't changed without updating this check, they stay in sync. Low risk, single source of truth (the entity list) for both.
- [Long pending-name lists wrapping awkwardly on small screens] → Row already has `flexWrap: 'wrap'`; acceptable for typical party sizes (3-6 players).

## Open Questions

- Exact wording/icon for the summary line (e.g. plain text vs. `⏳ Waiting on initiative: ...`) — left to implementation, not a behavior-affecting decision.
