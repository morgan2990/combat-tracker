import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

interface CompanionRow {
  name: string
  max_hp: string
  shares_initiative: boolean
}

function emptyCompanion(): CompanionRow {
  return { name: '', max_hp: '', shares_initiative: false }
}

export function CharacterForm() {
  const [searchParams] = useSearchParams()
  const [charName, setCharName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [companions, setCompanions] = useState<CompanionRow[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Pre-fill form if ?name= is set in the URL
  useEffect(() => {
    const nameParam = searchParams.get('name')
    if (!nameParam) return
    fetch(`/api/entities/${encodeURIComponent(nameParam)}`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (!data) return
        setCharName(data.profile.name)
        setMaxHP(String(data.profile.max_hp))
        setCompanions(
          (data.companions as Array<{ name: string; max_hp: number; shares_initiative: boolean }>).map(c => ({
            name: c.name,
            max_hp: String(c.max_hp),
            shares_initiative: c.shares_initiative,
          }))
        )
      })
      .catch(() => {})
  }, [searchParams])

  function addCompanion() {
    setCompanions(prev => [...prev, emptyCompanion()])
  }

  function removeCompanion(i: number) {
    setCompanions(prev => prev.filter((_, idx) => idx !== i))
  }

  function updateCompanion(i: number, field: keyof CompanionRow, value: string | boolean) {
    setCompanions(prev => prev.map((c, idx) => idx === i ? { ...c, [field]: value } : c))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    if (!charName.trim() || hp <= 0) return

    setSubmitting(true)
    setError(null)

    try {
      // Save player profile
      const playerRes = await fetch('/api/entities', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: charName.trim(), type: 'player', max_hp: hp }),
      })
      if (!playerRes.ok) throw new Error('Failed to save character')

      // Save each companion
      for (const c of companions) {
        if (!c.name.trim() || !c.max_hp) continue
        const cHP = parseInt(c.max_hp, 10)
        if (cHP <= 0) continue
        const res = await fetch('/api/entities', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: c.name.trim(),
            type: 'companion',
            max_hp: cHP,
            parent_pc_name: charName.trim(),
            shares_initiative: c.shares_initiative,
          }),
        })
        if (!res.ok) throw new Error(`Failed to save companion "${c.name}"`)
      }

      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setSubmitting(false)
    }
  }

  if (success) {
    return (
      <div style={{ maxWidth: 420, margin: '80px auto', padding: 24, textAlign: 'center' }}>
        <div style={{ fontSize: 32, marginBottom: 12 }}>✓</div>
        <h2 style={{ marginBottom: 8, color: '#27ae60' }}>Profile saved!</h2>
        <p style={{ color: '#7878a0', marginBottom: 24 }}>
          <strong style={{ color: '#d4d4e8' }}>{charName}</strong> is ready to use.
        </p>
        <Link
          to="/"
          style={{ display: 'inline-block', padding: '12px 32px', background: '#e67e22', color: '#fff', textDecoration: 'none', fontWeight: 600, fontSize: 16, borderRadius: 4 }}
        >
          Back to Join Screen
        </Link>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 480, margin: '40px auto', padding: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 24 }}>
        <Link to="/" style={{ color: '#7878a0', textDecoration: 'none', fontSize: 14 }}>← Back</Link>
        <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>Create / Edit Character</h1>
      </div>

      <form onSubmit={handleSubmit}>
        {/* Character fields */}
        <section style={sectionStyle}>
          <h3 style={sectionHeader}>Character</h3>
          <label style={labelStyle}>
            <span>Name</span>
            <input
              value={charName}
              onChange={e => setCharName(e.target.value)}
              placeholder="e.g. Aria Sunsong"
              required
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
              placeholder="e.g. 32"
              style={inputStyle}
            />
          </label>
        </section>

        {/* Companion rows */}
        <section style={{ ...sectionStyle, marginTop: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <h3 style={{ ...sectionHeader, marginBottom: 0 }}>Companions / Pets</h3>
            <button
              type="button"
              onClick={addCompanion}
              style={{ padding: '6px 12px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
            >
              + Add
            </button>
          </div>

          {companions.length === 0 && (
            <div style={{ color: '#5a5a78', fontSize: 13, padding: '8px 0' }}>No companions yet.</div>
          )}

          {companions.map((c, i) => (
            <div key={i} style={{ background: '#161626', border: '1px solid #2e2e48', borderRadius: 6, padding: 12, marginBottom: 10 }}>
              <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                <div style={{ flex: 2 }}>
                  <div style={fieldLabel}>Name</div>
                  <input
                    value={c.name}
                    onChange={e => updateCompanion(i, 'name', e.target.value)}
                    placeholder="e.g. Rex"
                    required
                    style={{ ...inputStyle, marginTop: 0 }}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={fieldLabel}>Max HP</div>
                  <input
                    type="number"
                    value={c.max_hp}
                    onChange={e => updateCompanion(i, 'max_hp', e.target.value)}
                    min={1}
                    required
                    placeholder="HP"
                    style={{ ...inputStyle, marginTop: 0 }}
                  />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, color: '#8888aa', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={c.shares_initiative}
                    onChange={e => updateCompanion(i, 'shares_initiative', e.target.checked)}
                  />
                  Shares initiative with character
                </label>
                <button
                  type="button"
                  onClick={() => removeCompanion(i)}
                  style={{ padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: 'none', color: '#e74c3c', border: '1px solid #3a1010', borderRadius: 4 }}
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </section>

        {error && (
          <div style={{ marginTop: 12, padding: '10px 12px', background: '#1a0808', border: '1px solid #e74c3c', borderRadius: 6, fontSize: 13, color: '#e74c3c' }}>
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={submitting || !charName.trim() || !maxHP}
          style={{
            marginTop: 20, width: '100%', padding: '12px 0', fontSize: 16, fontWeight: 600,
            cursor: submitting || !charName.trim() || !maxHP ? 'not-allowed' : 'pointer',
            opacity: submitting || !charName.trim() || !maxHP ? 0.45 : 1,
            background: '#e67e22', color: '#fff', border: 'none', borderRadius: 4,
          }}
        >
          {submitting ? 'Saving…' : 'Save Profile'}
        </button>
      </form>
    </div>
  )
}

const sectionStyle: React.CSSProperties = {
  border: '1px solid #2e2e48',
  borderRadius: 6,
  padding: 16,
}

const sectionHeader: React.CSSProperties = {
  margin: '0 0 12px',
  fontSize: 14,
  fontWeight: 600,
  color: '#7878a0',
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
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

const fieldLabel: React.CSSProperties = {
  fontSize: 12,
  color: '#5a5a78',
  marginBottom: 4,
  fontWeight: 600,
}

const inputStyle: React.CSSProperties = {
  padding: '8px 12px',
  fontSize: 16,
  width: '100%',
  boxSizing: 'border-box',
  fontWeight: 'normal',
  marginTop: 2,
}
