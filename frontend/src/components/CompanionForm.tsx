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

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    const init = parseInt(initiative, 10)
    if (!name.trim() || !hp || hp <= 0) return
    onSubmit({ type: 'add_companion', name: name.trim(), max_hp: hp, initiative: init || 0 })
    onClose()
  }

  return (
    <div style={overlayStyle}>
      <div style={panelStyle}>
        <h3 style={{ marginTop: 0, marginBottom: 16 }}>Add Summon / Pet</h3>
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
            <span>Initiative</span>
            <input
              type="number"
              value={initiative}
              onChange={e => setInitiative(e.target.value)}
              placeholder="e.g. 12"
              style={inputStyle}
            />
          </label>
          <div style={{ display: 'flex', gap: 8, marginTop: 20 }}>
            <button type="button" onClick={onClose} style={{ flex: 1, padding: '10px 0' }}>
              Cancel
            </button>
            <button type="submit" style={{ flex: 2, padding: '10px 0', fontWeight: 'bold' }}>
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
  background: 'rgba(0,0,0,0.5)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  zIndex: 100,
}

const panelStyle: React.CSSProperties = {
  background: 'white',
  borderRadius: 8,
  padding: 24,
  width: '90%',
  maxWidth: 340,
  fontFamily: 'sans-serif',
}

const labelStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 4,
  marginBottom: 12,
  fontWeight: 600,
}

const inputStyle: React.CSSProperties = {
  padding: '8px 12px',
  fontSize: 16,
  width: '100%',
  boxSizing: 'border-box',
  fontWeight: 'normal',
}
