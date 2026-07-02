## Context

`DMView.tsx` grew to ~788 lines by containing four largely-independent units in one file: `EntityRow` (a single combatant's row, expand/edit panel), `AddCreatureForm` (a `forwardRef` component exposing `selectCustomMonster` to its parent), `EncounterTemplatesControl` (a self-contained dropdown), and the `DMView` layout shell that composes all three plus `DMNavColumn`, `StatblockDrawer`/`StatblockColumn`, and `InventoryPanel`.

## Goals / Non-Goals

**Goals:**
- Each of the three extractable units lives in its own file, importable independently.
- `DMView.tsx` contains only the responsive layout shell.
- Zero behavioral change — every prop, callback, and internal state stays exactly as it was.

**Non-Goals:**
- No change to `EncounterTemplatesControl`'s fetch behavior (its network-error-vs-HTTP-error retry asymmetry is tracked separately by GitHub issue #7).
- No further splitting of the layout shell itself (e.g. separating the phone-tier and tablet/desktop-tier JSX) — it stays as one component.
- No prop renames or API changes to any of the three extracted components.

## Decisions

**Each unit takes only what it uses.** `EntityRow.tsx` takes `parseHP` and `actionBtn` (only used by `EntityRow`). `AddCreatureForm.tsx` takes `SEARCH_MIN_CHARS`/`SEARCH_DEBOUNCE_MS` and `fieldStyle` (only used by `AddCreatureForm`) and the exported `AddCreatureFormHandle` interface (needed by `DMView.tsx` for its `useRef`). `EncounterTemplatesControl.tsx` is fully self-contained. Nothing shared between the three was factored into a fourth "shared" module — code-cleanup already extracted the pieces that were genuinely duplicated (`entityVitals.ts`, `ConditionToggles`, `fetchJSON`, `CustomMonsterPillList`, `formFieldStyles.ts`); what remains in each unit is unique to it.

**`EncounterTemplatesControl` moved verbatim, fetch logic untouched.** It still uses a raw `fetch(...).then(...)` rather than the shared `fetchJSON` helper, preserving its existing retry-on-network-error-but-not-on-HTTP-error behavior exactly. Swapping it to `fetchJSON` is out of scope here — it's the subject of GitHub issue #7, a separate decision.

## Risks / Trade-offs

- [A mechanical move accidentally changes an import path or drops a prop] → Verified via `tsc -b` (catches missing/renamed imports and prop mismatches) and an end-to-end Playwright pass exercising all three extracted units (search-add, My-Creatures-select-then-submit, expand/condition-toggle/kill on `EntityRow`, and the phone-tier `EncounterTemplatesControl` dropdown).

## Migration Plan

Frontend-only, no data migration. Single commit: create the three new files, trim `DMView.tsx` to the shell, verify.

## Open Questions

None.
