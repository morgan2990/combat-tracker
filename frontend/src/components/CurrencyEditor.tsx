import { useState } from 'react'
import type { Currency } from '../types'
import { color, radius } from '../theme'

export const CURRENCY_DENOMS = ['pp', 'gp', 'ep', 'sp', 'cp'] as const

interface CurrencyEditorProps {
  currency: Currency
  onChange: (currency: Currency) => void
  inputWidth?: number
}

export function clamp(n: number): number {
  return Math.max(0, n)
}

// Applies every present, validly-formatted signed delta ("+10", "-3", "10")
// to currency, clamping each result to a minimum of 0. Denominations with
// an empty or unparseable draft pass through unchanged.
export function applyDrafts(currency: Currency, drafts: Partial<Record<keyof Currency, string>>): Currency {
  const result = { ...currency }
  for (const denom of CURRENCY_DENOMS) {
    const draft = drafts[denom]?.trim()
    if (!draft || !/^[+-]?\d+$/.test(draft)) continue
    result[denom] = clamp(currency[denom] + Number.parseInt(draft, 10))
  }
  return result
}

interface DenomInputProps {
  denom: keyof Currency
  type: 'number' | 'text'
  value: string | number
  placeholder?: string
  onChange: (value: string) => void
  inputWidth: number
}

function DenomInput({ denom, type, value, placeholder, onChange, inputWidth }: DenomInputProps) {
  return (
    <label style={denomLabel}>
      {denom}
      <input
        type={type}
        inputMode={type === 'text' ? 'numeric' : undefined}
        min={type === 'number' ? 0 : undefined}
        placeholder={placeholder}
        value={value}
        onChange={e => onChange(e.target.value)}
        style={{ ...inputStyle, width: inputWidth }}
      />
    </label>
  )
}

export function CurrencyEditor({ currency, onChange, inputWidth = 56 }: CurrencyEditorProps) {
  const [modalOpen, setModalOpen] = useState(false)
  const [drafts, setDrafts] = useState<Partial<Record<keyof Currency, string>>>({})

  function updateDenom(denom: keyof Currency, value: string) {
    onChange({ ...currency, [denom]: clamp(Number.parseInt(value, 10) || 0) })
  }

  function openModal() {
    setDrafts({})
    setModalOpen(true)
  }

  function applyModal() {
    onChange(applyDrafts(currency, drafts))
    setModalOpen(false)
  }

  return (
    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
      {CURRENCY_DENOMS.map(denom => (
        <DenomInput
          key={denom}
          denom={denom}
          type="number"
          value={currency[denom]}
          onChange={value => updateDenom(denom, value)}
          inputWidth={inputWidth}
        />
      ))}
      <button type="button" onClick={openModal} style={adjustBtn}>± Adjust</button>

      {modalOpen && (
        <div style={overlayStyle}>
          <div style={panelStyle}>
            <h3 style={{ marginTop: 0, marginBottom: 16, color: color.textPrimary }}>Adjust Currency</h3>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
              {CURRENCY_DENOMS.map(denom => (
                <DenomInput
                  key={denom}
                  denom={denom}
                  type="text"
                  placeholder="±0"
                  value={drafts[denom] ?? ''}
                  onChange={value => setDrafts(prev => ({ ...prev, [denom]: value }))}
                  inputWidth={inputWidth}
                />
              ))}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <button
                type="button"
                onClick={() => setModalOpen(false)}
                style={{ flex: 1, padding: '10px 0', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm, cursor: 'pointer', fontSize: 14 }}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={applyModal}
                style={{ flex: 2, padding: '10px 0', fontWeight: 'bold', background: '#9b59b6', color: '#fff', border: 'none', borderRadius: radius.sm, cursor: 'pointer', fontSize: 14 }}
              >
                Apply
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const denomLabel: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: 2, fontSize: 11, color: color.textSecondary, textTransform: 'uppercase',
}

const inputStyle: React.CSSProperties = {
  padding: '6px 8px', fontSize: 13, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const adjustBtn: React.CSSProperties = {
  padding: '6px 10px', fontSize: 12, cursor: 'pointer', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm,
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0,
  background: 'rgba(0,0,0,0.65)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  zIndex: 100,
}

const panelStyle: React.CSSProperties = {
  background: '#1e1e30',
  border: `1px solid ${color.border}`,
  borderRadius: 8,
  padding: 24,
  width: '90%',
  maxWidth: 340,
}
