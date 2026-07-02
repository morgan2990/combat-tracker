import type { CSSProperties } from 'react'

// Shared by DMView, EncounterForm, and MonsterForm's label/field wrappers.
// Values match EncounterForm/MonsterForm (the majority of the 3 sites);
// DMView's labelStyle gap and labelText fontSize were 1px smaller and are
// normalized here. fieldStyle is intentionally NOT shared: DMView's inline
// flex-row form relies on a fixed base width (120px) for its Name/search
// fields, while EncounterForm/MonsterForm's single-column forms use 100%
// width — these are genuinely different layout contexts, not drift.
export const labelStyle: CSSProperties = { display: 'flex', flexDirection: 'column', gap: 4 }
export const labelText: CSSProperties = { fontSize: 12, color: '#7878a0' }
