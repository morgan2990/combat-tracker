# Tasks: Epic 10 — Automated Monster Initiative Rolling

## Backend

- [x] T1: Promote `Monster.InitiativeModifier` from `int` to `*int` in `store/monster.go` (field, bson tag, json tag with omitempty); update the scrubber (`cmd/scrub/`) to assign `&modifier` instead of `modifier`; update `UpsertMonster` multipart handler branch in `api/handler.go` to parse `initiative_modifier` form value as optional `*int`.

- [x] T2: Add `InitiativeModifier *int` and `InitiativeRoll *int` to `room.Entity` struct in `room/room.go` with `json:"initiative_modifier,omitempty"` and `json:"initiative_roll,omitempty"` tags; add `rollD20() int` helper using `crypto/rand`.

- [x] T3: Update `room.AddCreature` in `room/room.go`: replace `initiative int` parameter with `initiativeModifier *int`; in the per-creature loop, if `r.State.IsStarted && initiativeModifier != nil` call `rollD20()`, compute total, set `Initiative` and `InitiativeRoll` on the entity; otherwise store `InitiativeModifier` and leave `Initiative` nil.

- [x] T4: Update `room.StartCombat` in `room/room.go`: after the player/companion initiative guard, loop over creature entities where `InitiativeModifier != nil && Initiative == nil`, call `rollD20()` per entity, set `InitiativeRoll` and `Initiative`; call `r.sortEntities()` before setting `IsStarted = true`.

- [x] T5: Update `addCreatureMsg` in `ws/handler.go`: remove `Initiative int`, add `InitiativeModifier *int`; update the `add_creature` dispatch to pass `msg.InitiativeModifier` to `room.AddCreature`.

## Frontend

- [x] T6: Fix `MonsterForm.tsx`: add `edition` state (`'5e' | '5.5e'`, default `'5e'`) with a toggle selector; add `initiativeModifier` state (`string`, default `''`) with an optional integer input labelled "Initiative Modifier (optional)"; in the submit handler, include `edition` always and include `initiative_modifier` only when the input is non-empty (parse to int, omit the key otherwise).

- [x] T7: Update `Entity` interface in `frontend/src/types.ts`: add `initiative_modifier?: number | null` and `initiative_roll?: number | null`.

- [x] T8: Update `DMView.tsx` Add Creature form: remove the initiative input and its state; in `handleAddCreature`, construct the `add_creature` WS message without `initiative` and with `initiative_modifier` only when the searched monster has a non-null `initiative_modifier`; ensure the monster search result stores `initiative_modifier` alongside the existing fields used for `source_type`, `reference_url`, `pdf_object_key`.

- [x] T9: Update initiative display in `DMView.tsx`: show `--` when `entity.initiative === null` (pre-combat staging); when `entity.initiative_roll != null`, render a tooltip on the initiative value showing `{initiative} (d20: {roll} + mod: {modifier with sign})`.
