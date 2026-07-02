## Why

The Encounter Builder screen (`/encounters/new`, `/encounters/:id/edit`) shows the "My Creatures" quick-pick as rounded pill buttons at every viewport width, including desktop. DMView's in-room equivalent already replaced pills with a persistent row-list design (`DMNavColumn`) at tablet and desktop tiers, keeping pills only on phone. The Encounter Builder never got that treatment, so it now reads as visually stale next to the rest of the desktop experience.

## What Changes

- Extract DMView's viewport-width tier detection (`useLayoutTier`, phone/tablet/desktop) into a shared hook module so it isn't duplicated.
- Extract `DMNavColumn`'s "My Creatures" row-list rendering (heading, empty state, row items) into a shared list component so both `DMNavColumn` and the Encounter Builder render it identically.
- `DMNavColumn` adopts the shared list component with no behavior change.
- The Encounter Builder's "My Creatures" quick-pick becomes tier-aware:
  - Phone tier (< 768px): unchanged pill buttons.
  - Tablet and desktop tiers (>= 768px): the shared row-list component, matching DMView's existing tablet/desktop presentation.
- No change to DMView's in-room "Add Creature" panel — it already keeps pills on phone only and rows on tablet/desktop via `DMNavColumn`.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `encounter-builder`: the "My Creatures" quick-pick section's requirement gains viewport-tier-aware presentation — pills below 768px, row-list at 768px and above — instead of always rendering pills.

## Impact

- `frontend/src/components/EncounterForm.tsx`: quick-pick section becomes tier-aware; adds a viewport-tier subscription.
- `frontend/src/components/DMView.tsx`: `useLayoutTier` (and its supporting constants/media-query logic) moves out to a shared module; `DMView` imports it instead of defining it locally.
- `frontend/src/components/DMNavColumn.tsx`: "My Creatures" row rendering moves out to a shared component; `DMNavColumn` imports and uses it instead of inlining the rows.
- New shared modules: a layout-tier hook module and a custom-monster row-list component, both under `frontend/src/`.
- No API, data model, or backend changes.
