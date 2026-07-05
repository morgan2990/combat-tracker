## Why

`CurrencyEditor` — used for a PC's personal currency, a party's pooled currency, and a DM's per-member view of party currency — only supports typing the new absolute total per denomination. Recording a transaction ("I found 10 gp", "I paid 3 sp for rope") requires checking the current total and doing the arithmetic by hand before typing the new value.

## What Changes

- `CurrencyEditor` gains a delta-entry modal, opened via a small trigger next to its existing absolute-value inputs (which are unchanged). The modal has one signed text field per denomination (`pp`/`gp`/`ep`/`sp`/`cp`, e.g. `+10` or `-3`), and a single "Apply"/"Cancel" for the whole modal — Apply computes all five new absolute values in one go and calls the same `onChange(currency)` `CurrencyEditor` already exposes; Cancel discards everything.
- A denomination whose delta would take it below zero is clamped to `0`, matching the clamp `CurrencyEditor`'s existing absolute-value inputs already apply.
- Because delta math only needs data `CurrencyEditor` already receives (its `currency` prop) and produces exactly what it already emits (`onChange`), this is entirely contained inside `CurrencyEditor` — its three consumers (`InventoryPanel.tsx`, `PartyInventoryPanel.tsx`, `Dashboard.tsx`'s pooled-currency editor) require no changes.
- No backend changes: currency is already stored and saved as an absolute total; the delta modal only changes how that absolute value is arrived at client-side, still going through each consumer's existing save action.

## Capabilities

### Modified Capabilities
- `pc-inventory`: `CurrencyEditor`'s currency-purse editing requirement gains the delta-entry modal (this is the capability that already documents `CurrencyEditor`'s currency behavior via `InventoryPanel`).
- `dm-party-inventory`: `PartyInventoryPanel`'s per-member currency editor gets the same delta-entry modal as `pc-inventory`'s `CurrencyEditor`, mirroring how catalog search is already cross-referenced between these two specs.
- `party-management`: Dashboard's pooled-currency editor gets the same delta-entry modal.

## Impact

- `frontend/src/components/CurrencyEditor.tsx`: adds the delta-entry modal (trigger button, 5 signed fields, Apply/Cancel, clamp-to-zero logic). Existing absolute-value inputs and the `onChange`/`currency` prop contract are unchanged.
- `frontend/src/components/InventoryPanel.tsx`, `PartyInventoryPanel.tsx`, `Dashboard.tsx`: no code changes — all three already just pass `currency`/`onChange` through to `CurrencyEditor`.
