## Why

DMs running lair encounters have no way to represent a lair action (an environmental hazard that always acts at initiative count 20) in the tracker — they either skip tracking it entirely or awkwardly repurpose a fake "creature" with placeholder HP that doesn't make sense. Epic 18 adds a one-click "+ Add Lair Action" injection that appears in the initiative order with none of the HP/condition machinery that doesn't apply to it.

## What Changes

- Add a new entity type value, `lair_action`, to the existing `pc | creature | companion` enum — **not** the nilable-HP approach literally described in Epic 18's AC1. Exploration surfaced that promoting `max_hp`/`current_hp` to pointers would touch 100+ call sites across PC/monster/companion code that has nothing to do with lair actions, with real nil-deref and NaN risk (e.g. `PlayerView.tsx`'s `current / max` ratio math). A type discriminator achieves the same UI outcome — no HP bar, no delta inputs, no status badges — at roughly a fifth of the footprint, reusing the `entity.type === 'creature'` gating pattern already used for statblocks and source lookups.
- Add a DM-only `add_lair_action` WS message (no payload) and a corresponding `AddLairAction(sessionID string) error` room method that appends an entity with `name: "Lair Action"`, `type: "lair_action"`, `initiative: 20`, `max_hp: 0`, `current_hp: 0`, and — **deviating from the epic's literal AC2**, which says `is_hidden: false` (inherits default) — `is_hidden: true` by default. This was a deliberate decision made during exploration: players shouldn't learn a lair action exists until the DM chooses to reveal it, which composes for free with the existing Epic 17 `is_hidden` filter once the entity type isn't `"creature"` (it bypasses Epic 15's pre-combat blanket-hide and is governed solely by `is_hidden`).
- Add an explicit tie-break rule to `sortEntities()`: when two entities share the same initiative value and one is `type: "lair_action"`, it always sorts after the other, regardless of insertion order. Today's stable sort alone doesn't guarantee this (a creature added *after* the lair action that also rolls a 20 would otherwise stay behind it only by insertion-order accident).
- Add a `+ Add Lair Action` button to the DM Panel's combat-controls bar, dispatching `add_lair_action` on click.
- Extend the DM Panel's 👁/🙈 visibility toggle (previously creature-only per Epic 17) to also render for `lair_action` rows, since this is now the DM's only way to reveal one.
- In both DM Panel and Player View row rendering, hide HP display, the HP delta/smart-input editor, dead/unconscious vital-state badges, condition toggles, and the Kill/Revive button for `type: "lair_action"` rows. Keep the Remove button (per AC5), the initiative editor (per exploration decision — DMs may want to override the fixed 20 for homebrew rules), and the Name/Alias editors (nothing in the epic asks to hide these, and they let a DM distinguish multiple lair actions, e.g. "Collapsing Ceiling" vs. "Toxic Gas", using the alias mechanism that already exists).
- No persistence schema change: `type`, `name`, `initiative`, `max_hp`, `current_hp`, and `is_hidden` are all already in the persisted-entity field list — a lair action round-trips through the existing snapshot/restore path with zero new fields.

## Capabilities

### New Capabilities
- `lair-actions`: the `add_lair_action` message/method, the fixed-initiative-20 injection shape, the default-hidden behavior, and the DM Panel button and row-rendering rules for this entity type.

### Modified Capabilities
- `room-state`: the entity `type` enum gains `lair_action`; the sorting requirement gains an explicit tie-break scenario; the role-based-presentation requirement gains a scenario documenting how default-`is_hidden` composes with the existing masking filter for this type.

## Impact

- `room/room.go`: `AddLairAction` method, `sortEntities` tie-break logic.
- `ws/handler.go`: `add_lair_action` dispatch case (no message struct needed — no payload).
- `frontend/src/types.ts`: `Entity.type` union gains `'lair_action'`.
- `frontend/src/components/DMView.tsx`: new button, extended visibility toggle condition, row-rendering hides for HP/status UI.
- `frontend/src/components/PlayerView.tsx`: row-rendering hides for HP/status UI (visibility filter itself needs no change).
- No changes to `store/room.go`, no changes to any PC/companion/monster code, no changes to `room-persistence` spec.
