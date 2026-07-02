## Why

No shared theme or design-token module exists anywhere in `frontend/src/`. ~297 inline hex-color literals are scattered across 24 component/utility files. Ten colors — the app's core text/border/background/status palette — account for the large majority of that repetition; the remaining ~30 distinct colors are single-purpose (e.g. `entityVitals.ts`'s dead/unconscious/active row-color functions) or coincidental value matches with no shared design meaning.

## What Changes

- `frontend/src/theme.ts` exports `color` (10 named tokens: `textPrimary`, `textSecondary`, `textMuted`, `border`, `bgPanel`, `bgCard`, `accent`, `danger`, `info`, `success`) and `radius` (3 named tokens: `sm` (4px), `md` (6px), `pill` (12px)).
- Every exact-match occurrence of those 10 hex values and 3 border-radius values across `frontend/src/` is replaced with a reference to the corresponding token, including composite CSS shorthand strings (e.g. `'1px solid #2e2e48'` becomes a template literal using `color.border`).
- Colors and radii outside that set of 10+3 are deliberately left as inline literals — see Impact for why.
- Pure internal refactor — every replaced value is byte-identical to what it replaced, so there is no visual change.

## Capabilities

### New Capabilities
(none — this is internal frontend structure, not an application capability)

### Modified Capabilities
(none — no requirement-level behavior or visual output changes)

## Impact

- 24 files across `frontend/src/` and `frontend/src/components/` gain an import from the new `theme.ts` and have their matching literals replaced.
- **Deliberately not touched**: colors that are single-purpose and already centralized (e.g. `entityVitals.ts`'s `vitalRowBg`/`vitalTextColor` dead/unconscious/active-turn colors — pulling those into shared tokens would create false coupling between an intentionally-scoped vital-state palette and the general UI palette), and colors that coincidentally share a hex value across two unrelated components (e.g. `#1a1608` is both `entityVitals.ts`'s "unconscious" row background and, separately, `PlayerView.tsx`'s "set your initiative" prompt background — forcing these into one shared token would make changing one accidentally change the other).
- No API, data model, or backend changes.
