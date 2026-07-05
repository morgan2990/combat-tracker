## ADDED Requirements

### Requirement: CurrencyEditor offers a delta-entry modal alongside absolute-value editing

`CurrencyEditor` SHALL offer a delta-entry modal, opened via a trigger next to its existing absolute-value inputs (which remain unchanged and always available). The modal SHALL present one signed text field per denomination (`pp`, `gp`, `ep`, `sp`, `cp`), accepting an optional leading `+` or `-` followed by digits (bare digits with no sign SHALL be treated as positive). The modal SHALL have a single "Apply" and a single "Cancel" action for the whole modal, not a per-field commit.

Applying SHALL compute each denomination's new absolute value as `current + delta`, clamped to a minimum of `0` (matching the clamp `CurrencyEditor`'s absolute-value inputs already apply), for every denomination with a non-empty, validly-formatted delta field. A denomination whose delta field is left empty SHALL be unchanged. The modal SHALL call `onChange` exactly once with the resulting `Currency` object, then close. Canceling SHALL close the modal without calling `onChange`.

#### Scenario: Applying a single-denomination delta
- **WHEN** a user opens the delta modal, enters `+10` in the `gp` field, leaves all other fields empty, and clicks Apply
- **THEN** `CurrencyEditor` SHALL call `onChange` once with `gp` equal to its previous value plus `10` and every other denomination unchanged

#### Scenario: Applying deltas across multiple denominations at once
- **WHEN** a user enters deltas in more than one denomination field and clicks Apply
- **THEN** `CurrencyEditor` SHALL call `onChange` once with all entered deltas applied simultaneously to their respective denominations

#### Scenario: A delta that would go negative clamps to zero
- **WHEN** a user enters a delta for a denomination whose current value plus that delta would be less than `0` (e.g. `-8` against a current value of `5`)
- **THEN** `CurrencyEditor` SHALL set that denomination to `0` in the resulting `Currency` passed to `onChange`, without an error

#### Scenario: Canceling discards all entered deltas
- **WHEN** a user enters one or more deltas and clicks Cancel instead of Apply
- **THEN** `CurrencyEditor` SHALL close the modal without calling `onChange`, leaving `currency` unchanged

#### Scenario: Bare digits are treated as a positive delta
- **WHEN** a user enters a denomination's delta field with digits only and no leading sign (e.g. `10`)
- **THEN** `CurrencyEditor` SHALL treat it identically to `+10`

#### Scenario: Absolute-value inputs remain available and unaffected
- **WHEN** a user has the delta modal available but instead edits a denomination directly in `CurrencyEditor`'s existing absolute-value input
- **THEN** the existing absolute-value editing behavior SHALL be unaffected by the presence of the delta modal
