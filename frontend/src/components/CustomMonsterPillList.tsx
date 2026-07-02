import type { CustomMonster } from '../types'

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
            padding: '5px 10px', fontSize: 12, cursor: 'pointer', borderRadius: 12,
            border: '1px solid #2e2e48', background: '#1a1a2c', color: '#d4d4e8',
          }}
        >
          {m.name} <span style={{ color: '#7878a0' }}>{m.max_hp} HP</span>
        </button>
      ))}
    </div>
  )
}
