import { useState } from 'react'
import type { ReactNode } from 'react'
import { sectionHeading } from './CustomMonsterList'
import { color } from '../theme'

interface CollapsibleSectionProps {
  // Stable slug used as the localStorage persistence key, e.g. "dm.rooms".
  id: string
  title: string
  count: number
  // Rendered next to the toggle, visible whether the section is expanded or
  // collapsed — reserved for a section's single-interaction "create new"
  // action (a bare link/button); anything requiring its own input belongs in
  // the collapsible body instead.
  headerAction?: ReactNode
  children: ReactNode
}

function storageKey(id: string): string {
  return `collapsible-section.${id}`
}

function readStoredExpanded(id: string): boolean {
  try {
    return localStorage.getItem(storageKey(id)) !== 'false'
  } catch {
    return true
  }
}

function writeStoredExpanded(id: string, expanded: boolean) {
  try {
    localStorage.setItem(storageKey(id), String(expanded))
  } catch {
    // Collapse state just won't persist this session.
  }
}

export function CollapsibleSection({ id, title, count, headerAction, children }: CollapsibleSectionProps) {
  const [expanded, setExpanded] = useState(() => readStoredExpanded(id))

  function toggle() {
    const next = !expanded
    setExpanded(next)
    writeStoredExpanded(id, next)
  }

  return (
    <div style={{ marginBottom: 20 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <button type="button" onClick={toggle} aria-expanded={expanded} style={toggleBtnStyle}>
          <span aria-hidden="true" style={chevronStyle}>{expanded ? '▾' : '▸'}</span>
          <span style={{ ...sectionHeading, marginBottom: 0 }}>{title}</span>
          <span style={countStyle}>{count}</span>
        </button>
        {headerAction}
      </div>
      <div style={dividerStyle} />
      {expanded && children}
    </div>
  )
}

const toggleBtnStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: 6, flex: 1,
  background: 'none', border: 'none', cursor: 'pointer', padding: 0, textAlign: 'left',
}

const chevronStyle: React.CSSProperties = { fontSize: 14, color: color.textSecondary, width: 10 }

const countStyle: React.CSSProperties = { fontSize: 11, color: color.textMuted, fontWeight: 400 }

const dividerStyle: React.CSSProperties = { borderTop: `1px solid ${color.border}`, margin: '8px 0 12px' }
