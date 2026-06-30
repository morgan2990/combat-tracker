# Design: Epic 10 — Automated Monster Initiative Rolling

## Decision 1: Monster.InitiativeModifier becomes *int

`Monster.InitiativeModifier` is promoted from `int` to `*int`.

The scrubber always sets it (even for +0, which is a valid modifier for Dex 10). Custom monsters registered via `MonsterForm.tsx` may leave the field blank, which the handler stores as nil. This distinction propagates to the WS message: a nil modifier means no auto-roll; a non-nil modifier (even 0) means roll d20 + 0.

## Decision 2: InitiativeModifier and InitiativeRoll land on Entity

Two new fields on `room.Entity`:

- `InitiativeModifier *int` — captured from the `add_creature` WS message at creation time; never recalculated from MongoDB.
- `InitiativeRoll *int` — the raw d20 face (1–20), set when the roll fires (StartCombat or mid-combat add).

Storing the modifier on Entity (rather than re-fetching from MongoDB at roll time) eliminates a DB round-trip inside a locked room and avoids a failure mode where the monster is deleted between staging and StartCombat.

Storing the roll separately from the final initiative lets the frontend show the breakdown ("d20: 13 + mod: +3 = 16") without any arithmetic.

## Decision 3: add_creature WS message loses initiative, gains initiative_modifier

Old: `{ type, name, max_hp, initiative, quantity, source_type, reference_url, pdf_object_key }`
New: `{ type, name, max_hp, initiative_modifier, quantity, source_type, reference_url, pdf_object_key }`

The frontend no longer provides a final initiative value. It provides a modifier (which it read from the search result), and the backend decides when to roll. If the field is absent/null, the backend treats it as nil.

## Decision 4: Roll timing

- **Pre-combat**: modifier stored on Entity, Initiative left nil, no roll. Roll fires at StartCombat.
- **StartCombat**: loop over creature entities where `InitiativeModifier != nil && Initiative == nil`; roll each; sort; then set `IsStarted = true`.
- **Mid-combat add**: if `r.State.IsStarted && initiativeModifier != nil`, roll immediately when creating each entity.

Creatures where `Initiative` was manually set by the DM pre-combat (via `dm_update_entity`) are skipped in the StartCombat loop (`Initiative != nil`).

## Decision 5: Independent rolls per creature in a batch

When quantity > 1, each creature in the batch gets its own independent d20 roll. "3 Goblins" could end up at 14, 9, and 17. The batch loop calls `rollD20()` per iteration.

## Decision 6: Nil modifier → nil initiative, DM sets manually

A creature added with no modifier (either custom monster with blank form, or a name typed directly without a search hit) gets `InitiativeModifier = nil` and `Initiative = nil`. It appears in the staging area with `--`. The DM sets its initiative manually via the override controls. This is the existing behaviour for all creatures; auto-roll is additive.

## Decision 7: rollD20 uses crypto/rand

`crypto/rand` is already imported in `room/room.go` for ID generation. A `rollD20()` helper uses `rand.Int(rand.Reader, big.NewInt(20))` and adds 1, reusing the existing import.

## Decision 8: Missing US10.3 absorbed

The original epic file had no US10.3. The work that would have belonged there (Entity struct changes on the backend, frontend type changes) is split: backend struct changes belong to US10.2, frontend type changes belong to US10.4.
