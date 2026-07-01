## Context

Epic 18 asks for a way to inject a fixed-initiative-20 hazard marker into the tracker. Its AC1 literally proposes making `max_hp`/`current_hp` nilable across the entity model; exploration (an `Explore` agent scanning the full backend/frontend) found this would touch 54+ backend and 49+ frontend call sites — including PC creation, companion instantiation, monster form validation, and HP ratio/clamping math entirely unrelated to lair actions — with concrete nil-deref and NaN risks (`room/room.go:390` `if currentHP > e.MaxHP`, `PlayerView.tsx:16` `current / max`). A type discriminator (`type: "lair_action"`, HP fields stay plain `int` defaulting to `0`) achieves the identical UI outcome at roughly a fifth of the footprint and reuses the `entity.type === 'creature'` gating pattern already used repeatedly in `DMView.tsx` (statblock button, name/alias editors, visibility toggle).

This also lands on top of three prior epics' masking machinery: Epic 15 (pre-combat creature fog-of-war, gated on `type === 'creature'`), Epic 16 (display name/alias, currently gated on `type === 'creature'`), and Epic 17 (`is_hidden` toggle, currently gated on `type === 'creature'` in the DM Panel UI). Each of these gates needs a decision about whether `lair_action` participates.

## Goals / Non-Goals

**Goals:**
- A DM can inject a `"Lair Action"` entity at initiative 20 with one click, with no HP/condition UI cluttering its row.
- It reliably loses initiative ties against any creature also at 20, regardless of add order.
- It starts hidden from players and stays hidden until the DM explicitly reveals it, reusing the Epic 17 mechanism rather than inventing a new one.
- DMs can rename/alias a lair action (for distinguishing multiple hazards) and can still override its initiative if a homebrew rule calls for it.

**Non-Goals:**
- No nilable HP fields — explicitly rejected per the exploration above.
- No change to `room-persistence` — `type`, `name`, `initiative`, `max_hp`, `current_hp`, `is_hidden` are all already persisted fields; a `lair_action` entity round-trips with zero schema changes.
- No limit on how many lair actions a DM can add — the epic doesn't ask for one, and D&D lair-action rules (one lair action per round, chosen from a list) are a DM judgment call the tracker doesn't need to enforce.
- No auto-roll/initiative-modifier concept for this type — initiative is fixed at creation, not rolled.

## Decisions

**Entity shape, set once at creation, no new fields:**
```go
Entity{
    ID:         newToken(8),
    Name:       "Lair Action",
    Type:       "lair_action",
    Initiative: intPtr(20),
    MaxHP:      0,
    CurrentHP:  0,
    Conditions: []string{},
    IsHidden:   true,  // deviates from AC2's literal "false" — see proposal
}
```
`MaxHP`/`CurrentHP` are `0`, not omitted — matching how every other entity's HP fields are always-present plain ints. Nothing reads them for a `lair_action` row once the render gates are in place, so the zero value is inert.

**Tie-break lives in `sortEntities`'s comparator, not a separate pass.** Current comparator: `nil`-initiative entities always sort last, otherwise descending by value. Add one more rule ahead of the numeric comparison: if both have non-nil equal initiative and exactly one is `type == "lair_action"`, that one sorts second (returns `false` for "should sort before"). This is a single `if` branch inserted before the `*a > *b` comparison — no restructuring of the existing sort.

**`AddLairAction` needs no message struct.** Unlike `add_creature` or `dm_update_entity`, there's no per-call data — every lair action is created identically (name, type, initiative are all fixed). The WS dispatch case can call `rm.AddLairAction(c.SessionID)` directly from the raw message type switch, with no `json.Unmarshal` step, mirroring how `remove_dead_creatures` needs no payload today.

**Widen three existing `entity.type === 'creature'` gates to include `'lair_action'`, leave two unchanged:**
| Gate | Widen to include `lair_action`? | Why |
|---|---|---|
| 👁/🙈 visibility toggle | Yes | It's now the DM's only way to reveal a hidden lair action |
| Name editor | Yes | Lets a DM relabel "Lair Action" to something specific |
| Alias editor | Yes | Same reasoning, matches Epic 16's mechanism |
| Statblock button (📋) | No | A lair action has no `source_type`/statblock to view — the existing `entity.source_type &&` condition already excludes it naturally, no code change needed |
| Kill/Revive button | No | No HP/dead concept applies; hidden entirely for this type |

**HP/status UI hidden via a single `isLairAction` check, not scattered `!==` conditions.** Both `DMView.tsx`'s `EntityRow` and `PlayerView.tsx`'s row renderer already compute a `vitalState` before rendering badges — `lair_action` short-circuits to a fixed non-dead/non-unconscious state the same way `PlayerView.tsx` already special-cases `isCreature` today (`const vitalState = isCreature ? 'alive' : entityVitalState(...)`). The HP display block, the HP smart-input editor, and the condition-toggle row are each wrapped in `entity.type !== 'lair_action' &&`.

**Player View's `visibleEntities` filter needs zero changes.** Since `lair_action !== 'creature'`, the existing `(is_started || e.type !== 'creature') && !e.is_hidden` filter already resolves the first clause to `true` unconditionally for this type — the only gate that matters is `!e.is_hidden`, which is exactly the desired behavior given `IsHidden` defaults to `true` at creation.

**Button placement: outside the `is_started` conditional.** The combat-controls bar currently branches entirely on `is_started` (pre-combat: Start Combat button; mid-combat: Next Turn/End Combat). `+ Add Lair Action` is useful in both states (a DM might stage one before combat starts, or inject one mid-fight when a lair triggers), so it renders unconditionally in that row, not inside either branch.

## Risks / Trade-offs

- **[Risk]** A `lair_action` row's initiative can still be edited via the kept initiative input, which could accidentally "fix" it away from 20 in a way that looks like a bug rather than an intentional homebrew override. → **Mitigation**: accepted per exploration decision; the input's placeholder/label doesn't need special-casing since the DM already sees the current value and chose to open the edit panel deliberately.
- **[Risk]** Two independent "hidden by default" mechanisms now exist in the codebase for different reasons (Epic 15's blanket pre-combat creature hide vs. this epic's per-entity default-`true` flag) — a future reader might not immediately see why `lair_action` doesn't need the Epic 15 treatment. → **Mitigation**: documented explicitly in the `room-state` delta spec's new scenario, not left implicit.

## Open Questions

None outstanding — HP-field approach, tie-break behavior, and which DM row controls to keep were resolved during exploration prior to this proposal.
