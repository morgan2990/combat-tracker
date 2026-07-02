import { CONDITIONS } from '../entityVitals'
import { color, radius } from '../theme'

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
              padding: '4px 10px', fontSize: 12, borderRadius: radius.pill, border: '1px solid',
              borderColor: active ? color.danger : color.border,
              background: active ? '#2a0808' : color.bgPanel,
              color: active ? color.danger : '#8888aa',
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
