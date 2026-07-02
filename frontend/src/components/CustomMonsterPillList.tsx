import type { CustomMonster } from '../types'
import { color, radius } from '../theme'

interface CustomMonsterPillListProps {
  monsters: CustomMonster[]
  onSelect: (monster: CustomMonster) => void
}

export function CustomMonsterPillList({ monsters, onSelect }: CustomMonsterPillListProps) {
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 4 }}>
      {monsters.map(m => (
        <button
          key={m.id}
          type="button"
          onClick={() => onSelect(m)}
          style={{
            padding: '5px 10px', fontSize: 12, cursor: 'pointer', borderRadius: radius.pill,
            border: `1px solid ${color.border}`, background: color.bgPanel, color: color.textPrimary,
          }}
        >
          {m.name} <span style={{ color: color.textSecondary }}>{m.max_hp} HP</span>
        </button>
      ))}
    </div>
  )
}
