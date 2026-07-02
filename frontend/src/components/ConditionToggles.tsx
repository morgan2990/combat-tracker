import { CONDITIONS } from '../entityVitals'

interface ConditionTogglesProps {
  conditions: string[]
  onToggle: (condition: string) => void
}

export function ConditionToggles({ conditions, onToggle }: ConditionTogglesProps) {
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
      {CONDITIONS.map(cond => {
        const active = conditions.includes(cond)
        return (
          <button
            key={cond}
            onClick={() => onToggle(cond)}
            style={{
              padding: '4px 10px', fontSize: 12, borderRadius: 12, border: '1px solid',
              borderColor: active ? '#e74c3c' : '#2e2e48',
              background: active ? '#2a0808' : '#1a1a2c',
              color: active ? '#e74c3c' : '#8888aa',
              cursor: 'pointer',
            }}
          >
            {cond}
          </button>
        )
      })}
    </div>
  )
}
