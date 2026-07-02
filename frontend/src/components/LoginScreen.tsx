import { useState } from 'react'
import type { FormEvent } from 'react'
import { color, radius } from '../theme'

interface LoginScreenProps {
  onAuthed: () => void
}

export function LoginScreen({ onAuthed }: LoginScreenProps) {
  const [mode, setMode] = useState<'login' | 'signup'>('login')
  const [username, setUsername] = useState('')
  const [passphrase, setPassphrase] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!username.trim() || !passphrase) return
    setSubmitting(true)
    setError(null)
    try {
      const res = await fetch(mode === 'login' ? '/api/login' : '/api/signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: username.trim(), passphrase }),
      })
      if (!res.ok) {
        if (res.status === 401) setError('Wrong username or passphrase.')
        else if (res.status === 409) setError('That username is already taken.')
        else setError('Something went wrong. Please try again.')
        return
      }
      onAuthed()
    } catch {
      setError('Service unavailable. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div style={{ maxWidth: 340, margin: '100px auto', padding: 24 }}>
      <h1 style={{ marginBottom: 24, fontSize: 26, fontWeight: 700, textAlign: 'center' }}>⚔ Combat Tracker</h1>

      <div style={{ display: 'flex', marginBottom: 20, borderBottom: `2px solid ${color.border}` }}>
        {(['login', 'signup'] as const).map(m => (
          <button
            key={m}
            type="button"
            onClick={() => { setMode(m); setError(null) }}
            style={{
              flex: 1, padding: '8px 0', fontSize: 14,
              fontWeight: mode === m ? 700 : 400,
              background: 'none', border: 'none',
              borderBottom: mode === m ? `2px solid ${color.info}` : '2px solid transparent',
              marginBottom: -2, cursor: 'pointer',
              color: mode === m ? color.info : color.textSecondary,
            }}
          >
            {m === 'login' ? 'Log In' : 'Create Account'}
          </button>
        ))}
      </div>

      <form onSubmit={handleSubmit}>
        <label style={{ display: 'block', marginBottom: 12 }}>
          <div style={labelText}>Username</div>
          <input
            value={username}
            onChange={e => setUsername(e.target.value)}
            placeholder="e.g. aria"
            autoFocus
            style={inputStyle}
          />
        </label>
        <label style={{ display: 'block', marginBottom: 16 }}>
          <div style={labelText}>Passphrase</div>
          <input
            type="password"
            value={passphrase}
            onChange={e => setPassphrase(e.target.value)}
            placeholder="••••••••"
            style={inputStyle}
          />
        </label>

        {error && <div style={errorStyle}>{error}</div>}

        <button
          type="submit"
          disabled={submitting || !username.trim() || !passphrase}
          style={primaryBtn(submitting || !username.trim() || !passphrase)}
        >
          {submitting ? 'Please wait…' : mode === 'login' ? 'Log In' : 'Create Account'}
        </button>
      </form>
    </div>
  )
}

const labelText: React.CSSProperties = { fontSize: 13, color: color.textSecondary, marginBottom: 4 }

const inputStyle: React.CSSProperties = {
  width: '100%', padding: '10px 12px', fontSize: 16, boxSizing: 'border-box',
  background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const errorStyle: React.CSSProperties = {
  marginBottom: 12, padding: '8px 12px', fontSize: 13, color: color.danger,
  background: '#1a0808', border: `1px solid ${color.danger}`, borderRadius: radius.sm,
}

function primaryBtn(disabled: boolean): React.CSSProperties {
  return {
    width: '100%', padding: '12px 0', fontSize: 15, fontWeight: 700,
    background: color.info, color: '#fff', border: 'none', borderRadius: radius.sm,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}
