import { useState } from 'react'
import type { FormEvent } from 'react'

interface SetupFormProps {
  myName: string
  onSubmit: (msg: object) => void
}

export function SetupForm({ myName, onSubmit }: SetupFormProps) {
  const [maxHP, setMaxHP] = useState('')
  const [initiative, setInitiative] = useState('')
  const [submitted, setSubmitted] = useState(false)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    const init = parseInt(initiative, 10)
    if (!hp || hp <= 0) return
    setSubmitted(true)
    onSubmit({ type: 'setup_character', max_hp: hp, initiative: init || 0 })
  }

  return (
    <div style={{ maxWidth: 360, margin: '80px auto', padding: 24, fontFamily: 'sans-serif' }}>
      <h2 style={{ marginBottom: 4 }}>Welcome, {myName}!</h2>
      <p style={{ color: '#666', marginTop: 0, marginBottom: 24 }}>Set up your character to join the tracker.</p>
      <form onSubmit={handleSubmit}>
        <label style={{ display: 'block', marginBottom: 16 }}>
          <div style={{ marginBottom: 4, fontWeight: 600 }}>Max HP</div>
          <input
            type="number"
            value={maxHP}
            onChange={e => setMaxHP(e.target.value)}
            min={1}
            required
            disabled={submitted}
            placeholder="e.g. 32"
            style={inputStyle}
          />
        </label>
        <label style={{ display: 'block', marginBottom: 24 }}>
          <div style={{ marginBottom: 4, fontWeight: 600 }}>Initiative</div>
          <input
            type="number"
            value={initiative}
            onChange={e => setInitiative(e.target.value)}
            disabled={submitted}
            placeholder="e.g. 14"
            style={inputStyle}
          />
        </label>
        <button
          type="submit"
          disabled={submitted || !maxHP}
          style={{ width: '100%', padding: '12px 0', fontSize: 16, cursor: submitted ? 'wait' : 'pointer' }}
        >
          {submitted ? 'Joining tracker…' : 'Enter Combat'}
        </button>
      </form>
    </div>
  )
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 12px',
  fontSize: 18,
  boxSizing: 'border-box',
}
