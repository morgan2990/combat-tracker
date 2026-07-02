import type { Currency } from '../types'
import { color, radius } from '../theme'

export const CURRENCY_DENOMS = ['pp', 'gp', 'ep', 'sp', 'cp'] as const

interface CurrencyEditorProps {
  currency: Currency
  onChange: (currency: Currency) => void
  inputWidth?: number
}

export function CurrencyEditor({ currency, onChange, inputWidth = 56 }: CurrencyEditorProps) {
  function updateDenom(denom: keyof Currency, value: string) {
    const n = Math.max(0, parseInt(value, 10) || 0)
    onChange({ ...currency, [denom]: n })
  }

  return (
    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
      {CURRENCY_DENOMS.map(denom => (
        <label key={denom} style={denomLabel}>
          {denom}
          <input
            type="number"
            min={0}
            value={currency[denom]}
            onChange={e => updateDenom(denom, e.target.value)}
            style={{ ...inputStyle, width: inputWidth }}
          />
        </label>
      ))}
    </div>
  )
}

const denomLabel: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: 2, fontSize: 11, color: color.textSecondary, textTransform: 'uppercase',
}

const inputStyle: React.CSSProperties = {
  padding: '6px 8px', fontSize: 13, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}
