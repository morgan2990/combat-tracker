## Context

`RoomState.entities` is broadcast in full to every connected client, regardless of role — this is an established pattern (`room-state` spec, "Frontend is responsible for role-based data presentation"), already used to hide exact creature HP from players in favor of qualitative labels (Healthy/Injured/Dying). Epic 15 asks for the same kind of client-side masking, extended to hide entire creature rows during pre-combat staging.

`DMView.tsx` and `PlayerView.tsx` are separate components with independent render logic (no shared entity-list component), so this is purely a `PlayerView.tsx` change.

The backend piece of Epic 15 (a combat-active flag present on every broadcast, defaulting false, flipping on start/end combat) is already fully implemented as `RoomState.IsStarted` / `is_started` — confirmed by reading `room/room.go` (set at lines 155 and 335, broadcast via the existing full-state-on-every-mutation pattern) and the `room-state` spec's existing data model section. No backend change is in scope.

## Goals / Non-Goals

**Goals:**
- Hide DM-staged creatures from the player-facing initiative ladder while `is_started` is `false`.
- Reveal them instantly, with no filter, the moment `is_started` becomes `true`.
- Give players a clear signal that staging is happening even when the filtered list is empty.
- Leave the DM Panel, the WS payload, and the backend entirely unchanged.

**Non-Goals:**
- No animation/transition on reveal — matches the rest of the app's instant re-render behavior on every other state change.
- No ally/hostile distinction among creatures — the schema has one DM-controlled type (`creature`); all creatures are hidden pre-combat, none are exempted.
- No rename of `is_started` to `is_combat_active` — treated as a naming decision, not a functional gap (see proposal).
- No change to how companions or PCs are filtered — they already always render; this only touches `type === 'creature'` rows.

## Decisions

**Filter placement: derive a `visibleEntities` list at the top of `PlayerView`, before the existing `.map`.**
```ts
const visibleEntities = is_started
  ? entities
  : entities.filter(e => e.type !== 'creature')
```
Alternative considered: filtering inside the `.map` callback with an early `return null`. Rejected — that still allocates a row slot conceptually and makes the staging empty-state check (`visibleEntities.length === 0`) awkward to express; a plain array filter keeps the existing render loop untouched below the first line.

**Empty-state copy: distinguish "nothing added yet" from "creatures hidden."**
The existing check is `entities.length === 0` → "No combatants yet." Two new cases both use `visibleEntities`:
- `visibleEntities.length === 0 && entities.length === 0` → keep existing "No combatants yet." copy (nothing staged at all).
- `visibleEntities.length === 0 && entities.length > 0` → new copy, e.g. "The DM is preparing the encounter…" (creatures exist but are hidden pre-combat).

This only requires comparing the two lengths already available in scope — no new state or WS field needed to distinguish the cases.

**No new prop threading.** `is_started` is already destructured from `roomState` in `PlayerView`; the filter is entirely local to the existing render, no prop changes to `PlayerViewProps`.

**Spec location: extend `room-state`'s existing role-based-presentation requirement rather than a new capability.** The requirement already establishes "server sends full state, client decides what to show per role" as the governing rule; creature-visibility-during-staging is one more scenario under it, not a new concern.

## Risks / Trade-offs

- **[Risk]** A player who is mid-render when `is_started` flips could see a single jarring layout jump (rows appearing) rather than a graceful transition. → **Mitigation**: explicitly a non-goal per the "instant reveal" decision confirmed during exploration; the app has no precedent for animated transitions and adding one here would be new scope, not a bug fix.
- **[Risk]** Future work might introduce a non-hostile/ally creature type, at which point a blanket `type === 'creature'` hide would over-hide. → **Mitigation**: none needed now; flagged here so a future change touching entity hostility revisits this filter.

## Open Questions

None outstanding — naming, empty-state copy behavior, and animation scope were resolved during exploration prior to this proposal.
