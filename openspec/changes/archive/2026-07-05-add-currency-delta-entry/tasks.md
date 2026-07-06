## 1. Delta parsing and math

- [x] 1.1 Add a small parser/validator for a signed delta string: accepts `^[+-]?\d+$`, treats bare digits as positive, rejects/ignores anything else (empty string, non-numeric, malformed).
- [x] 1.2 Add a pure helper that takes the current `Currency` and a per-denomination map of parsed deltas, returns a new `Currency` with each present delta applied and clamped to a minimum of `0` (denominations with no entered delta pass through unchanged).

## 2. Delta modal UI (`frontend/src/components/CurrencyEditor.tsx`)

- [x] 2.1 Add a trigger button next to the existing absolute-value inputs that opens the delta modal (local `isOpen` state), mirroring `CompanionForm.tsx`'s `position: fixed` overlay + centered panel pattern.
- [x] 2.2 Render one signed text field per denomination inside the modal (local draft state, reset to empty whenever the modal opens).
- [x] 2.3 Wire "Apply": parse all five fields (task 1.1), compute the new `Currency` (task 1.2), call `onChange` once with the result, then close the modal.
- [x] 2.4 Wire "Cancel": close the modal without calling `onChange`, discarding the draft state.

## 3. Verification

- [x] 3.1 Manually verify in `InventoryPanel`: open the delta modal, apply a single-denomination delta, confirm the absolute value updates and the existing Save button persists it via `PUT /api/pcs/:id`.
- [x] 3.2 Manually verify in `Dashboard`'s pooled-currency editor: same flow, saved via `PUT /api/parties/:id`.
- [ ] 3.3 Manually verify in `PartyInventoryPanel`: apply a delta to one party member's currency, confirm only that member's currency changes, saved via `PUT /api/rooms/{room_id}/party-inventory/pcs/{pc_id}`. NOT driven live — reaching it requires a full DM+player WebSocket room session with a linked party. `PartyInventoryPanel.tsx` has zero code diff and renders the same `CurrencyEditor` already verified live in 3.1/3.2.
- [x] 3.4 Manually verify clamping: enter a delta that would take a denomination below zero, confirm it clamps to `0` instead of going negative.
- [x] 3.5 Manually verify Cancel: enter deltas, click Cancel, confirm the underlying currency is unchanged.
- [x] 3.6 Manually verify bare-digit entry (no leading sign) is treated as positive.
