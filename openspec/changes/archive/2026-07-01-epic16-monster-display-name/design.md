## Context

The entity model already has one optional per-instance field added the same way this one will be: `InitiativeModifier`/`InitiativeRoll` (see `entity-schema` spec) — an optional field threaded through the Go struct, the `add_creature` WS message, the Mongo snapshot, and the frontend type, with `omitempty`/nullable semantics throughout. `display_name` follows the identical shape.

The masking side extends a pattern already used twice: `room-state`'s "Frontend is responsible for role-based data presentation" requirement already covers HP masking (players see qualitative labels, not exact HP) and pre-combat creature masking (players don't see staged monsters at all). Name masking is a third instance of the same rule: server sends full state to everyone, client decides what to render per role.

Two decisions here diverge from Epic 16's literal text (both already discussed and decided during exploration, see proposal.md):
1. Batch-added aliases auto-number per instance, rather than sharing one identical string.
2. The alias becomes live-editable after creation, not just settable at spawn time.

## Goals / Non-Goals

**Goals:**
- Optional `display_name` on creature entities, round-tripped through the WS payload, Mongo snapshot, and restore path.
- DM Panel shows both names when an alias exists (`"{display_name} ({name})"`); players only ever see one name per entity (alias if set, else base name).
- Batch-added aliases are auto-numbered exactly like base names already are.
- DM can set the alias at creation (Add Creature form) and change it afterward (existing live-edit row control).

**Non-Goals:**
- PCs and companions do not get aliases — the field is only ever populated via `add_creature`, which only creates creature-type entities.
- No validation/uniqueness constraints on alias text — same freeform-string treatment as the base `name` field.
- No change to how `source_type`/`reference_url`/statblock lookups work — those stay keyed off the real monster identity, never the alias.

## Decisions

**Field shape mirrors `initiative_modifier`.** `DisplayName string` on `Entity` (not a pointer) — unlike `InitiativeModifier *int`, there's no need to distinguish "unset" from "zero value" for a string; empty string already unambiguously means "no alias." `json:"display_name,omitempty"` on the wire so old clients/rooms without an alias don't see a spurious empty field.

**Batch numbering reuses the existing `AddCreature` loop.** The function already does `entityName := name; if quantity > 1 { entityName = fmt.Sprintf("%s %d", name, i+1) }`. The same pattern applies to `displayName` when non-empty: `entityDisplayName := displayName; if displayName != "" && quantity > 1 { entityDisplayName = fmt.Sprintf("%s %d", displayName, i+1) }`. When no alias was provided, it stays empty across the whole batch (never numbered into e.g. `" 1"`).

**Live editing extends `dm_update_entity`, `DMUpdateEntity`, and the existing rename control — not a new message type.** The existing `name` edit already lives in `dm_update_entity` (guarded by `e.Type == "creature" && name != ""` — creatures only, and blank means "don't touch"). `display_name` gets its own field on the same message, but with different blank-handling: unlike `name`, an empty `display_name` is meaningful (it means "clear the alias, go back to showing the base name"), so the update is unconditional for creatures — no `!= ""` guard. The frontend's `EntityRow` gets a second input next to the existing "Name" field, and `sendUpdate`'s default-field spread gains `display_name: entity.display_name ?? ''` so every update preserves the current alias unless explicitly changed.

**Dual-label format is a plain string template, not a new component.** `` `${entity.display_name} (${entity.name})` `` inline where the DM Panel currently renders `entity.name`. No new dedicated UI component — this is the same weight as the existing inline `entity.type` / vital-state badges already rendered next to the name.

**Player View: single conditional fallback.** `entity.display_name || entity.name` at the one place `PlayerView` currently renders `entity.name`. No new empty-state or placeholder logic needed — this is a straight substitution, unlike the Epic 15 staging filter which needed new placeholder copy.

## Risks / Trade-offs

- **[Risk]** Auto-numbering aliases (deviating from AC4) means a DM who genuinely wants three identically-named "Guard" instances (deliberately indistinguishable to players) can no longer get that by typing one alias — they'd have to name each instance manually one at a time (adding creatures with quantity 1 three times). → **Mitigation**: accepted trade-off; matches how base names already work today (a DM can't get 3 identically-named base creatures either), so behavior is at least consistent within the app rather than surprising.
- **[Risk]** Extending `dm_update_entity`'s blank-handling asymmetrically (`name`: blank = no-op, `display_name`: blank = clear) could confuse a future reader of `DMUpdateEntity`. → **Mitigation**: the two fields have genuinely different semantics (a creature must always have *a* name; an alias is optional by definition), and this is called out explicitly in the `entity-schema` delta spec below so the asymmetry is documented, not implicit.

## Open Questions

None outstanding — format, batch-numbering, and live-edit scope were resolved during exploration prior to this proposal.
