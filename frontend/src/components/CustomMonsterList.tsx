import type { CustomMonster } from '../types'
import { color, radius } from '../theme'

interface CustomMonsterListProps {
  monsters: CustomMonster[]
  onSelect: (monster: CustomMonster) => void
}

export const sectionHeading: React.CSSProperties = {
  fontSize: 12, fontWeight: 600, color: color.textSecondary,
  textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 8,
}
export const emptyText: React.CSSProperties = { fontSize: 12, color: '#454568', padding: '4px 0' }
export const navItem: React.CSSProperties = {
  padding: '8px 10px', cursor: 'pointer', fontSize: 13, color: color.textPrimary,
  borderRadius: radius.sm, marginBottom: 4, background: color.bgPanel, border: '1px solid transparent',
}

export function CustomMonsterList({ monsters, onSelect }: CustomMonsterListProps) {
  return (
    <div>
      <div style={sectionHeading}>My Creatures</div>
      {monsters.length === 0 && (
        <div style={emptyText}>No custom monsters for this edition.</div>
      )}
      {monsters.map(m => (
        <div key={m.id} onClick={() => onSelect(m)} style={navItem}>
          {m.name} <span style={{ color: color.textSecondary }}>{m.max_hp} HP</span>
        </div>
      ))}
    </div>
  )
}
