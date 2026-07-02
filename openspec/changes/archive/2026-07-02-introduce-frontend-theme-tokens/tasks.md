## 1. Introduce the theme module

- [x] 1.1 Create `frontend/src/theme.ts` exporting `color` (10 tokens: `textPrimary`, `textSecondary`, `textMuted`, `border`, `bgPanel`, `bgCard`, `accent`, `danger`, `info`, `success`) and `radius` (3 tokens: `sm`, `md`, `pill`), with each value matching its most-frequent existing usage exactly.

## 2. Apply tokens across the codebase

- [x] 2.1 Replace every exact-match occurrence of the 10 target hex colors and 3 border-radius values across `frontend/src/` (24 files: `App.tsx`, `entityVitals.ts`, `formFieldStyles.ts`, and 21 files under `components/`) with the corresponding token reference, including composite CSS shorthand strings (`'Npx solid #HEX'` → template literal).
- [x] 2.2 Insert the `import { color, radius } from '../theme'` (or `'./theme'` for root-level files) into each modified file.
- [x] 2.3 Confirm zero remaining occurrences of the 10 target hex values outside `theme.ts` itself, via `grep`.

## 3. Verify

- [x] 3.1 `tsc -b` and `oxlint` clean (same pre-existing warnings as before, no new ones).
- [x] 3.2 Spot-checked several file diffs by hand (`ConditionToggles.tsx`, `DMView.tsx`, `formFieldStyles.ts`) to confirm the substitutions read correctly and rare/single-purpose colors were correctly left untouched.
- [x] 3.3 Visual verification via Playwright screenshots (Dashboard, EncounterForm, MonsterForm, DMView desktop and phone tiers) — pixel-identical to prior screenshots from earlier in the session, confirming zero visual change.
