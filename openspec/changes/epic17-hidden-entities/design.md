## Context

This is the fourth extension of the same pattern: `room-state`'s "Frontend is responsible for role-based data presentation" requirement already covers HP masking, pre-combat creature masking (Epic 15), and name masking (Epic 16). `is_hidden` is a fourth, orthogonal masking axis on the same requirement.

It also composes directly with Epic 15's existing `visibleEntities` filter in `PlayerView.tsx` (`is_started ? entities : entities.filter(e => e.type !== 'creature')`), which already established the exact shape needed here.

The backend field/message shape mirrors two existing precedents: `InitiativeModifier`/`DisplayName` for the optional-field pattern, and `removeEntityMsg`/`RemoveEntity` for the "DM action addressed by entity_id alone" message pattern.

## Goals / Non-Goals

**Goals:**
- DM can toggle any creature's `is_hidden` flag instantly, with the change broadcast to all clients.
- DM Panel always shows every entity, with a distinct visual treatment (50% opacity) for hidden ones — the DM's own view is never filtered.
- Player View completely omits hidden creatures from the rendered initiative ladder — same omission guarantee as the existing pre-combat creature masking.
- The two masking layers (pre-combat blanket creature hiding, per-entity `is_hidden`) compose without either one accidentally exposing what the other hides.

**Non-Goals:**
- No support for hiding PC or companion entities — the DM Panel toggle only renders on creature rows, so `is_hidden` is never set for other types in this change.
- No animated reveal transition — instant re-render, matching the Epic 15 decision and the rest of the app.
- No bulk hide/reveal-all action — this change only adds the single-entity toggle; batch operations are out of scope.
- No interaction with "Start Combat" gating — hidden creatures don't need initiative before combat starts (the existing gate only checks PCs/companions), so no change needed there.

## Decisions

**Field shape mirrors `DisplayName`, not `InitiativeModifier`.** `IsHidden bool` (not a pointer) — a zero-value `bool` already means "not hidden," so there's no unset/false ambiguity to resolve with a pointer. `json:"is_hidden"` — no `omitempty`, matching how `Dead bool` is already serialized unconditionally (a boolean masking flag should always be explicit on the wire, not silently omitted when false).

**New dedicated message, not folded into `dm_update_entity`.** `toggle_entity_visibility` mirrors `remove_entity`'s `{entity_id}`-only shape rather than being added as a field on the already-large `dm_update_entity` message. Visibility is a discrete DM action (like Kill/Revive or Remove), not part of the HP/conditions/initiative bundle that `dm_update_entity` represents — bundling it there would mean every HP tick also has to carry the entity's current hidden state just to avoid clobbering it, adding a footgun for no benefit.

**Persistence: dirty-mark only, no immediate write.** Per the existing `room-persistence` spec, only five events trigger an immediate Mongo write (join, leave, combat start, combat end, turn advance). Toggling visibility is not combat-critical to survive a crash instantly, so it follows the same "mark dirty, periodic sweep picks it up" path as HP/condition updates.

**Player View filter composition.** The existing Epic 15 filter becomes:
```ts
const visibleEntities = entities.filter(e =>
  (is_started || e.type !== 'creature') && !e.is_hidden
)
```
Both conditions are independent AND-ed gates — pre-combat blanket-hides all creatures regardless of `is_hidden`, and `is_hidden` can additionally hide any individual creature at any combat state. Neither can un-hide what the other is hiding. No change needed to the Epic 15 staging-placeholder logic (`visibleEntities.length === 0 && entities.length > 0`) — PCs always render and are never hidden in this scope, so that placeholder's meaning (used pre-combat, when all creatures are always hidden) is unaffected by `is_hidden`.

**DM Panel row styling.** A hidden row keeps its existing `vitalState`-driven background (dead/unconscious/active/normal) but is rendered at reduced opacity (e.g. `opacity: 0.5` on the row wrapper) when `entity.is_hidden` is true — additive to the existing background-color logic, not a replacement for it, so a hidden-and-dead creature still reads as both.

**Icon choice.** 👁 (visible) / 🙈 (hidden) inline button, same visual weight and placement pattern as the existing 📋 statblock button on creature rows.

## Risks / Trade-offs

- **[Risk]** A DM Panel row can now be simultaneously "dead," "hidden," and "active" — three independent visual states stacking (greyed text, 50% opacity, orange active background). → **Mitigation**: accepted; each is a distinct, additive visual cue (color for vital state, opacity for visibility, background tint for turn) rather than mutually exclusive states, so they layer without ambiguity.
- **[Risk]** Creature-only scope means a DM cannot represent "an invisible player" this epic. → **Mitigation**: explicitly out of scope per the proposal; flagged for a future change if it becomes a real need, since it would require exposing the toggle on PC/companion rows too and deciding how a player would perceive their own hidden status.

## Open Questions

None outstanding — entity-type scope, reveal animation, and icon style were resolved during exploration prior to this proposal.
