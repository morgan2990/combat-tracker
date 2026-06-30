import { useState } from 'react'
import type { FormEvent } from 'react'

interface CompanionFormProps {
  onSubmit: (msg: object) => void
  onClose: () => void
}

export function CompanionForm({ onSubmit, onClose }: CompanionFormProps) {
  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [initiative, setInitiative] = useState('')
  const [sharesInitiative, setSharesInitiative] = useState(false)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    if (!name.trim() || !hp || hp <= 0) return
    const init = initiative.trim() !== '' ? parseInt(initiative, 10) : null
    onSubmit({
      type: 'add_companion',
      name: name.trim(),
      max_hp: hp,
      shares_initiative: sharesInitiative,
      initiative: isNaN(init as number) ? null : init,
    })
    onClose()
  }

  return (
    <div style={overlayStyle}>
      <div style={panelStyle}>
        <h3 style={{ marginTop: 0, marginBottom: 16, color: '#d4d4e8' }}>Add Summon / Pet</h3>
        <form onSubmit={handleSubmit}>
          <label style={labelStyle}>
            <span>Name</span>
            <input
              value={name}
              onChange={e => setName(e.target.value)}
              required
              placeholder="e.g. Wolf"
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            <span>Max HP</span>
            <input
              type="number"
              value={maxHP}
              onChange={e => setMaxHP(e.target.value)}
              min={1}
              required
              placeholder="e.g. 18"
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            <span>Initiative <span style={{ fontWeight: 400, color: '#5a5a78' }}>(optional)</span></span>
            <input
              type="number"
              value={initiative}
              onChange={e => setInitiative(e.target.value)}
              placeholder="leave blank if sharing"
              disabled={sharesInitiative}
              style={{ ...inputStyle, opacity: sharesInitiative ? 0.4 : 1 }}
            />
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16, cursor: 'pointer', fontSize: 13, color: '#8888aa' }}>
            <input
              type="checkbox"
              checked={sharesInitiative}
              onChange={e => { setSharesInitiative(e.target.checked); if (e.target.checked) setInitiative('') }}
            />
            Shares initiative with me
          </label>
          <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
            <button
              type="button"
              onClick={onClose}
              style={{ flex: 1, padding: '10px 0', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4, cursor: 'pointer', fontSize: 14 }}
            >
              Cancel
            </button>
            <button
              type="submit"
              style={{ flex: 2, padding: '10px 0', fontWeight: 'bold', background: '#9b59b6', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer', fontSize: 14 }}
            >
              Add to Tracker
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0,
  background: 'rgba(0,0,0,0.65)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  zIndex: 100,
}

const panelStyle: React.CSSProperties = {
  background: '#1e1e30',
  border: '1px solid #2e2e48',
  borderRadius: 8,
  padding: 24,
  width: '90%',
  maxWidth: 340,
}

const labelStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 4,
  marginBottom: 12,
  fontWeight: 600,
  fontSize: 13,
  color: '#7878a0',
}

const inputStyle: React.CSSProperties = {
  padding: '8px 12px',
  fontSize: 16,
  width: '100%',
  boxSizing: 'border-box',
  fontWeight: 'normal',
}
