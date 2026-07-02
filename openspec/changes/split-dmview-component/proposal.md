## Why

`frontend/src/components/DMView.tsx` is ~832 lines, containing four largely-independent units — `EntityRow`, `AddCreatureForm`, `EncounterTemplatesControl`, and the `DMView` layout shell itself. This is scoped to happen after `code-cleanup`'s `EntityRow`/`PlayerView.tsx` duplication extraction, so there's less code to move once the shared classifier/row-color logic is already pulled out.

## What Changes

- Move `EntityRow`, `AddCreatureForm`, and `EncounterTemplatesControl` out of `DMView.tsx` into their own files, mirroring the extraction pattern already used for `CustomMonsterList`/`useLayoutTier`.
- Leave the `DMView` layout shell (the three-column responsive structure) in `DMView.tsx`.
- Pure internal restructuring — no behavioral change intended; exact file boundaries and any further splitting to be defined when this change is picked up (ideally after `code-cleanup`'s `EntityRow`/`PlayerView` extraction is done).

This is a placeholder proposal capturing the need; scope and specifics are to be defined when this change is picked up.

## Capabilities

### New Capabilities
(none — this is internal frontend structure, not an application capability)

### Modified Capabilities
(none identified yet — to be determined when scoped)

## Impact

- `frontend/src/components/DMView.tsx` and new sibling component files.
- No API, data model, or backend changes.
