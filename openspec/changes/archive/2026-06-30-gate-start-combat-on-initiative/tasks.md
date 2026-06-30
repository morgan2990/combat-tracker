## 1. Derive pending-initiative entities

- [x] 1.1 In `DMView.tsx`, compute the list of blocking entities from `roomState.entities`: `type === 'player' || type === 'companion'`, `initiative === null`.

## 2. Disable the Start Combat button

- [x] 2.1 Add `disabled={pending.length > 0}` to the Start Combat button.
- [x] 2.2 Apply disabled styling (opacity ~0.45, `cursor: not-allowed`) following the existing pattern in `JoinScreen.tsx` (`primaryBtn`) / `PlayerView.tsx` Set button.

## 3. Render the pending-initiative summary

- [x] 3.1 Add a summary `<span>` in the Combat controls row, rendered only when `pending.length > 0`, listing blocking entity names comma-separated (e.g. `Waiting on initiative: Bob, Fido`).

## 4. Verify

- [x] 4.1 Manually test: add a player and companion (non-sharing) without initiative, confirm button is disabled and summary lists both by name.
- [x] 4.2 Manually test: set initiative on all blocking entities, confirm button enables and summary disappears.
- [x] 4.3 Manually test: companion with `shares_initiative: true` — confirm it drops off the summary automatically once its owning player's initiative is set.
- [x] 4.4 Manually test: a `creature` entity with no initiative does not block the button.
