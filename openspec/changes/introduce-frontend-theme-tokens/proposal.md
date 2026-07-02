## Why

No shared theme or design-token module exists anywhere in `frontend/src/`. The same handful of hex colors (`#7878a0`, `#1a1a2c`, `#2e2e48`, `#d4d4e8`, `#454568`, `#e67e22`, `#e74c3c`, `#27ae60`, `#3498db`) and spacing/border-radius values are redefined inline across all 17 component files (~250 occurrences). This is scoped separately from `code-cleanup` because it's a large, purely mechanical pass better done as an independent change, rather than bundled with smaller extractions.

## What Changes

- Introduce a shared theme/design-token module (e.g. `frontend/src/theme.ts`) with named constants for the repeated colors, spacing, and border-radius values.
- Replace the inline hex-literal/spacing usages across the 17 component files with references to the shared constants.
- Pure internal refactor — no visual or behavioral change intended; exact scope (which values become tokens, naming convention) to be defined when this change is picked up.

This is a placeholder proposal capturing the need; scope and specifics are to be defined when this change is picked up.

## Capabilities

### New Capabilities
(none — this is internal frontend structure, not an application capability)

### Modified Capabilities
(none identified yet — to be determined when scoped)

## Impact

- All 17 files under `frontend/src/components/` that currently hand-roll color/spacing literals.
- No API, data model, or backend changes.
