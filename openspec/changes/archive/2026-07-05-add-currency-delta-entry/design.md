## Context

`CurrencyEditor` is a small, fully-controlled component (`currency: Currency` in, `onChange: (currency: Currency) => void` out) rendering five absolute-value number inputs (`pp`/`gp`/`ep`/`sp`/`cp`), each clamped to `Math.max(0, ...)` on change. It's used identically by three consumers — `InventoryPanel.tsx` (a PC's personal currency), `Dashboard.tsx` (a party's pooled currency), and `PartyInventoryPanel.tsx` (a DM's per-member view) — each of which holds its own local `currency` state and persists it via its own explicit "Save" action (`PUT /api/pcs/:id`, `PUT /api/parties/:id`, or `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}` respectively). None of the three auto-saves on keystroke.

## Goals / Non-Goals

**Goals:**
- Let a user apply a currency change as a delta ("+10 gp", "-3 sp") instead of only overwriting the absolute total, in every context `CurrencyEditor` appears.
- Keep the change contained to `CurrencyEditor` itself — no changes to any of its three consumers.

**Non-Goals:**
- No backend changes. Currency is already stored and validated as an absolute total (`Currency.IsNegative()` server-side); the delta modal only changes how the client arrives at that absolute value before the existing save action runs.
- No auto-save or auto-apply — delta entry still ends with the same explicit consumer-owned "Save" step that exists today.
- No combined/multi-denomination free-text parsing (e.g. one field accepting "+10gp -3sp"). Each denomination gets its own field.

## Decisions

**1. The delta editor is a modal owned entirely by `CurrencyEditor`, not a companion component the three consumers render separately.**
Delta math (`current[denom] + delta = new absolute`) only needs data `CurrencyEditor` already has (its `currency` prop), and produces exactly what it already emits (`onChange(currency)`). Alternative considered: a separate companion component the three consumers render next to `CurrencyEditor`. Rejected — each consumer would need a new render line for no functional benefit; the trigger button and modal both fit naturally inside `CurrencyEditor`'s own render output. The existing `CompanionForm.tsx` `position: fixed` overlay + centered panel is the modal pattern to mirror.

**2. Five per-denomination signed delta fields, not one combined free-text field.**
Alternative considered: a single field parsing something like "+10 gp -3 sp" in one entry, closer to how a real loot drop might be described verbally. Rejected for this pass — it requires a parser and ambiguous-input handling that per-denomination fields avoid entirely, and the per-denomination shape mirrors the layout `CurrencyEditor`'s existing absolute inputs already use.

**3. A single "Apply"/"Cancel" for the whole modal, not a per-field commit.**
Alternative considered: each denomination field commits independently as soon as it's entered, letting the modal stay open across multiple unrelated adjustments. Rejected — batching every entered delta behind one Apply/Cancel means exactly one `onChange` call regardless of how many denominations were touched, and gives a clean, reversible "did I mean to do this" moment before anything is written to the consumer's local state.

**4. Signed inline text fields (`type="text"`, manually validated), not unsigned amount + separate `+`/`−` buttons.**
A native `<input type="number">` won't accept a typed leading `+` — `CurrencyEditor`'s existing absolute inputs rely on `type="number"`, but the delta fields need `type="text"` with manual parsing (`/^[+-]?\d+$/` or similar) to accept "+10"/"-3"/"10" (bare positive, no sign, treated as `+10`). This is one field per denomination instead of an amount field plus two buttons.

**5. A delta that would take a denomination below zero clamps to `0`, silently.**
Matches the clamp already applied by `CurrencyEditor`'s existing absolute-value `onChange` handler (`Math.max(0, ...)`). Alternatives considered: rejecting the Apply with an inline error (more honest about the transaction not being valid as entered, but a new error-surfacing path this component doesn't have today), or letting a negative value through to rely on the backend's existing `IsNegative()` 400 at save time (would be the first way a negative value could reach that check from the UI at all, since the absolute inputs already prevent it from ever getting there today). Clamping keeps the same silent, immediate-feedback behavior the absolute inputs already have.

## Risks / Trade-offs

- **[Risk]** Clamping to zero silently could surprise a user who mistyped a delta larger than the current balance (e.g. "-30" against a balance of 5). → **Mitigation**: none beyond what the absolute-value inputs already do in the same situation today — accepted as consistent with existing behavior, not a new gap.
- **[Risk]** Manually validating a signed text field is a small new parsing surface that native `type="number"` doesn't need. → **Mitigation**: the accepted grammar is narrow (`^[+-]?\d+$`), and an invalid entry simply doesn't parse — Apply can skip/ignore any field that doesn't match rather than needing a dedicated error state.

## Open Questions

None.
