## Why

`frontend/src/components/DMView.tsx` was ~788 lines, containing four largely-independent units — `EntityRow`, `AddCreatureForm`, `EncounterTemplatesControl`, and the `DMView` layout shell itself. This is scoped to happen after `code-cleanup`'s `EntityRow`/`PlayerView.tsx` duplication extraction, so there's less code to move once the shared classifier/row-color logic is already pulled out.

## What Changes

- `EntityRow` moves to `frontend/src/components/EntityRow.tsx`, taking its local `parseHP` helper and `actionBtn` style constant with it.
- `AddCreatureForm` (a `forwardRef` component) moves to `frontend/src/components/AddCreatureForm.tsx`, taking its `SEARCH_MIN_CHARS`/`SEARCH_DEBOUNCE_MS` constants, `fieldStyle`, and the exported `AddCreatureFormHandle` interface with it.
- `EncounterTemplatesControl` moves to `frontend/src/components/EncounterTemplatesControl.tsx` verbatim, including its existing retry-on-network-error-only fetch behavior (tracked separately by GitHub issue #7 — not touched here).
- `DMView.tsx` keeps only the three-column responsive layout shell, importing the three extracted components.
- Pure file-structure move — no behavioral change, no prop/API changes to any of the four units.

## Capabilities

### New Capabilities
(none — this is internal frontend structure, not an application capability)

### Modified Capabilities
(none — no requirement-level behavior changes)

## Impact

- `frontend/src/components/DMView.tsx`: reduced from ~788 lines to ~230 (the layout shell only).
- New files: `frontend/src/components/EntityRow.tsx`, `AddCreatureForm.tsx`, `EncounterTemplatesControl.tsx`.
- No API, data model, or backend changes.
