import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import type { PC } from '../types'
import { fetchJSON } from '../fetchJSON'
import { deleteWithConfirm } from '../deleteWithConfirm'
import { color, radius } from '../theme'

interface CompanionRow {
  name: string
  max_hp: string
  shares_initiative: boolean
}

type ExistingCompanionRow = CompanionRow & { id: string }

function emptyCompanion(): CompanionRow {
  return { name: '', max_hp: '', shares_initiative: false }
}

// Trims/parses a companion row's fields; returns null if invalid (matching
// the guard every companion-saving call site needs before submitting).
function parseCompanionRow(row: CompanionRow): { name: string; max_hp: number } | null {
  const hp = Number.parseInt(row.max_hp, 10)
  if (!row.name.trim() || !hp || hp <= 0) return null
  return { name: row.name.trim(), max_hp: hp }
}

interface CharacterFormProps {
  onSaved: () => void
}

export function CharacterForm({ onSaved }: CharacterFormProps) {
  const navigate = useNavigate()
  const { id } = useParams<{ id?: string }>()
  const editing = Boolean(id)

  const [charName, setCharName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [existingCompanions, setExistingCompanions] = useState<ExistingCompanionRow[]>([])
  const [savingCompanionId, setSavingCompanionId] = useState<string | null>(null)
  const [companionActionError, setCompanionActionError] = useState<string | null>(null)
  const [newCompanions, setNewCompanions] = useState<CompanionRow[]>([])
  const [loading, setLoading] = useState(editing)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // In edit mode, load the existing PC and its companions.
  useEffect(() => {
    if (!id) return
    fetchJSON<{ pc: PC; companions: PC[] } | null>(`/api/pcs/${encodeURIComponent(id)}`, null)
      .then(data => {
        if (!data) return
        setCharName(data.pc.name)
        setMaxHP(String(data.pc.max_hp))
        setExistingCompanions(data.companions.map(c => ({
          id: c.id, name: c.name, max_hp: String(c.max_hp), shares_initiative: c.shares_initiative,
        })))
      })
      .finally(() => setLoading(false))
  }, [id])

  function updateExistingCompanion(companionId: string, field: keyof CompanionRow, value: string | boolean) {
    setExistingCompanions(prev => prev.map(c => c.id === companionId ? { ...c, [field]: value } : c))
  }

  async function saveExistingCompanion(companionId: string) {
    const row = existingCompanions.find(c => c.id === companionId)
    const parsed = row && parseCompanionRow(row)
    if (!row || !parsed) return

    setCompanionActionError(null)
    setSavingCompanionId(companionId)
    try {
      const res = await fetch(`/api/pcs/${encodeURIComponent(companionId)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: parsed.name, max_hp: parsed.max_hp, shares_initiative: row.shares_initiative }),
      })
      if (!res.ok) throw new Error('Failed to save companion')
    } catch (err) {
      setCompanionActionError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setSavingCompanionId(null)
    }
  }

  function deleteExistingCompanion(companionId: string) {
    return deleteWithConfirm(
      'Delete this companion? This cannot be undone.',
      `/api/pcs/${encodeURIComponent(companionId)}`,
      () => setExistingCompanions(prev => prev.filter(c => c.id !== companionId)),
      () => setCompanionActionError('Failed to delete companion'),
    )
  }

  function addCompanionRow() {
    setNewCompanions(prev => [...prev, emptyCompanion()])
  }

  function removeCompanionRow(i: number) {
    setNewCompanions(prev => prev.filter((_, idx) => idx !== i))
  }

  function updateCompanionRow(i: number, field: keyof CompanionRow, value: string | boolean) {
    setNewCompanions(prev => prev.map((c, idx) => idx === i ? { ...c, [field]: value } : c))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    if (!charName.trim() || hp <= 0) return

    setSubmitting(true)
    setError(null)

    try {
      let pcId = id
      if (editing && pcId) {
        const res = await fetch(`/api/pcs/${encodeURIComponent(pcId)}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: charName.trim(), max_hp: hp }),
        })
        if (!res.ok) throw new Error('Failed to save character')
      } else {
        const res = await fetch('/api/pcs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: charName.trim(), max_hp: hp }),
        })
        if (!res.ok) throw new Error('Failed to save character')
        const created = await res.json()
        pcId = created.id
      }

      for (const c of newCompanions) {
        const parsed = parseCompanionRow(c)
        if (!parsed) continue
        const res = await fetch(`/api/pcs/${encodeURIComponent(pcId!)}/companions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: parsed.name,
            max_hp: parsed.max_hp,
            shares_initiative: c.shares_initiative,
          }),
        })
        if (!res.ok) throw new Error(`Failed to save companion "${parsed.name}"`)
      }

      await onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return null

  return (
    <div style={{ maxWidth: 480, margin: '40px auto', padding: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 24 }}>
        <button onClick={() => navigate('/')} style={{ background: 'none', border: 'none', color: color.textSecondary, fontSize: 14, cursor: 'pointer', padding: 0 }}>← Back</button>
        <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>{editing ? 'Edit Character' : 'Create Character'}</h1>
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

        {/* Existing companions (edit mode) */}
        {existingCompanions.length > 0 && (
          <section style={{ ...sectionStyle, marginTop: 20 }}>
            <h3 style={sectionHeader}>Existing Companions / Pets</h3>
            {companionActionError && (
              <div style={{ marginBottom: 10, padding: '10px 12px', background: '#1a0808', border: `1px solid ${color.danger}`, borderRadius: radius.md, fontSize: 13, color: color.danger }}>
                {companionActionError}
              </div>
            )}
            {existingCompanions.map(c => {
              const saving = savingCompanionId === c.id
              return (
                <div key={c.id} style={{ background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: radius.md, padding: 12, marginBottom: 10 }}>
                  <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                    <div style={{ flex: 2 }}>
                      <div style={fieldLabel}>Name</div>
                      <input
                        value={c.name}
                        onChange={e => updateExistingCompanion(c.id, 'name', e.target.value)}
                        style={{ ...inputStyle, marginTop: 0 }}
                      />
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={fieldLabel}>Max HP</div>
                      <input
                        type="number"
                        value={c.max_hp}
                        onChange={e => updateExistingCompanion(c.id, 'max_hp', e.target.value)}
                        min={1}
                        style={{ ...inputStyle, marginTop: 0 }}
                      />
                    </div>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, color: '#8888aa', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={c.shares_initiative}
                        onChange={e => updateExistingCompanion(c.id, 'shares_initiative', e.target.checked)}
                      />{' '}
                      Shares initiative with character
                    </label>
                    <span style={{ display: 'flex', gap: 8 }}>
                      <button
                        type="button"
                        onClick={() => saveExistingCompanion(c.id)}
                        disabled={saving || !c.name.trim() || !c.max_hp}
                        style={{ padding: '4px 10px', fontSize: 12, cursor: saving ? 'not-allowed' : 'pointer', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm }}
                      >
                        {saving ? 'Saving…' : 'Save'}
                      </button>
                      <button
                        type="button"
                        onClick={() => deleteExistingCompanion(c.id)}
                        style={{ padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: 'none', color: color.danger, border: '1px solid #3a1010', borderRadius: radius.sm }}
                      >
                        Delete
                      </button>
                    </span>
                  </div>
                </div>
              )
            })}
          </section>
        )}

        {/* New companion rows */}
        <section style={{ ...sectionStyle, marginTop: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <h3 style={{ ...sectionHeader, marginBottom: 0 }}>Add Companions / Pets</h3>
            <button
              type="button"
              onClick={addCompanionRow}
              style={{ padding: '6px 12px', fontSize: 13, cursor: 'pointer', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm }}
            >
              + Add
            </button>
          </div>

          {newCompanions.length === 0 && (
            <div style={{ color: color.textMuted, fontSize: 13, padding: '8px 0' }}>No new companions.</div>
          )}

          {newCompanions.map((c, i) => (
            <div key={i} style={{ background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: radius.md, padding: 12, marginBottom: 10 }}>
              <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                <div style={{ flex: 2 }}>
                  <div style={fieldLabel}>Name</div>
                  <input
                    value={c.name}
                    onChange={e => updateCompanionRow(i, 'name', e.target.value)}
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
                    onChange={e => updateCompanionRow(i, 'max_hp', e.target.value)}
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
                    onChange={e => updateCompanionRow(i, 'shares_initiative', e.target.checked)}
                  />
                  Shares initiative with character
                </label>
                <button
                  type="button"
                  onClick={() => removeCompanionRow(i)}
                  style={{ padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: 'none', color: color.danger, border: '1px solid #3a1010', borderRadius: radius.sm }}
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </section>

        {error && (
          <div style={{ marginTop: 12, padding: '10px 12px', background: '#1a0808', border: `1px solid ${color.danger}`, borderRadius: radius.md, fontSize: 13, color: color.danger }}>
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
            background: color.accent, color: '#fff', border: 'none', borderRadius: radius.sm,
          }}
        >
          {submitting ? 'Saving…' : 'Save Character'}
        </button>
      </form>
    </div>
  )
}

const sectionStyle: React.CSSProperties = {
  border: `1px solid ${color.border}`,
  borderRadius: radius.md,
  padding: 16,
}

const sectionHeader: React.CSSProperties = {
  margin: '0 0 12px',
  fontSize: 14,
  fontWeight: 600,
  color: color.textSecondary,
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
  color: color.textSecondary,
}

const fieldLabel: React.CSSProperties = {
  fontSize: 12,
  color: color.textMuted,
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
