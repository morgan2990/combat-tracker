import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import type { CustomMonster } from '../types'
import { fetchJSON } from '../fetchJSON'
import { EditionToggle } from './EditionToggle'
import { labelStyle, labelText } from '../formFieldStyles'

type SourceType = 'none' | 'url' | 'pdf'

export function MonsterForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id?: string }>()
  const editing = Boolean(id)

  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [edition, setEdition] = useState<'5e' | '5.5e'>('5e')
  const [initiativeModifier, setInitiativeModifier] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [sourceType, setSourceType] = useState<SourceType>('none')
  const [referenceURL, setReferenceURL] = useState('')
  const [pdfFile, setPdfFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [loading, setLoading] = useState(editing)
  const [submitting, setSubmitting] = useState(false)

  // In edit mode, load the existing custom monster.
  useEffect(() => {
    if (!id) return
    fetchJSON<CustomMonster | null>(`/api/custom-monsters/${encodeURIComponent(id)}`, null)
      .then(data => {
        if (!data) return
        setName(data.name)
        setMaxHP(String(data.max_hp))
        setEdition(data.edition as '5e' | '5.5e')
        setInitiativeModifier(data.initiative_modifier != null ? String(data.initiative_modifier) : '')
        setIsPrivate(Boolean(data.private))
        if (data.source_type === 'url' || data.source_type === 'pdf') {
          setSourceType(data.source_type)
          setReferenceURL(data.reference_url ?? '')
        }
      })
      .finally(() => setLoading(false))
  }, [id])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)

    try {
      let res: Response
      if (!editing && sourceType === 'pdf' && pdfFile) {
        const form = new FormData()
        form.append('name', name.trim())
        form.append('max_hp', maxHP)
        form.append('edition', edition)
        form.append('source_type', 'pdf')
        form.append('pdf', pdfFile)
        form.append('private', String(isPrivate))
        if (initiativeModifier.trim() !== '') form.append('initiative_modifier', initiativeModifier.trim())
        res = await fetch('/api/monsters', { method: 'POST', body: form })
      } else {
        const body: Record<string, unknown> = { name: name.trim(), max_hp: parseInt(maxHP, 10), edition, private: isPrivate }
        if (sourceType === 'url' && referenceURL.trim()) {
          body.source_type = 'url'
          body.reference_url = referenceURL.trim()
        }
        if (initiativeModifier.trim() !== '') {
          body.initiative_modifier = parseInt(initiativeModifier.trim(), 10)
        }
        res = await fetch(editing ? `/api/custom-monsters/${encodeURIComponent(id!)}` : '/api/monsters', {
          method: editing ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        })
      }

      if (res.status === 413) { setError('PDF file is too large (max 20 MB).'); return }
      if (!res.ok) { setError(`Error: ${await res.text()}`); return }

      if (editing) {
        navigate('/')
      } else {
        setSaved(true)
      }
    } catch {
      setError('Network error. Is the server running?')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return null

  if (saved) {
    return (
      <div style={{ maxWidth: 480, margin: '60px auto', padding: 24, textAlign: 'center' }}>
        <div style={{ fontSize: 20, marginBottom: 16, color: '#27ae60' }}>Monster saved!</div>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
          <button
            onClick={() => { setName(''); setMaxHP(''); setEdition('5e'); setInitiativeModifier(''); setIsPrivate(false); setSourceType('none'); setReferenceURL(''); setPdfFile(null); setSaved(false) }}
            style={btnStyle('#2e2e48')}
          >
            Add Another
          </button>
          <button onClick={() => navigate('/')} style={btnStyle('#2980b9')}>Back to Dashboard</button>
        </div>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 480, margin: '40px auto', padding: 24 }}>
      <h2 style={{ marginTop: 0 }}>{editing ? 'Edit Monster' : 'Register Monster'}</h2>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>

        <label style={labelStyle}>
          <span style={labelText}>Monster Name</span>
          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Goblin"
            required
            style={fieldStyle}
          />
        </label>

        <label style={labelStyle}>
          <span style={labelText}>Max HP</span>
          <input
            type="number"
            value={maxHP}
            onChange={e => setMaxHP(e.target.value)}
            placeholder="7"
            min={1}
            required
            style={{ ...fieldStyle, width: 100 }}
          />
        </label>

        <div style={labelStyle}>
          <span style={labelText}>Edition</span>
          <EditionToggle edition={edition} onChange={setEdition} />
        </div>

        <label style={labelStyle}>
          <span style={labelText}>Initiative Modifier (optional)</span>
          <input
            type="number"
            value={initiativeModifier}
            onChange={e => setInitiativeModifier(e.target.value)}
            placeholder="e.g. 2 or -1"
            style={{ ...fieldStyle, width: 100 }}
          />
        </label>

        <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#d4d4e8', cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={isPrivate}
            onChange={e => setIsPrivate(e.target.checked)}
          />
          Mark as Private Campaign Content
          <span
            title="Private monsters are only visible to you — in your dashboard and your active rooms. Other DMs won't see them, even in search."
            style={{ cursor: 'help', color: '#7878a0' }}
          >
            ⓘ
          </span>
        </label>

        <div style={labelStyle}>
          <span style={labelText}>Statblock Reference</span>
          <div style={{ display: 'flex', gap: 8 }}>
            {(editing ? ['none', 'url'] as SourceType[] : ['none', 'url', 'pdf'] as SourceType[]).map(t => (
              <button
                key={t}
                type="button"
                onClick={() => setSourceType(t)}
                style={{
                  padding: '6px 14px', fontSize: 13, cursor: 'pointer', borderRadius: 4,
                  border: '1px solid',
                  borderColor: sourceType === t ? '#e67e22' : '#2e2e48',
                  background: sourceType === t ? '#2a1a08' : '#1a1a2c',
                  color: sourceType === t ? '#e67e22' : '#8888aa',
                }}
              >
                {t === 'none' ? 'None' : t.toUpperCase()}
              </button>
            ))}
          </div>
        </div>

        {sourceType === 'url' && (
          <label style={labelStyle}>
            <span style={labelText}>Image URL (e.g. 5etools .webp link)</span>
            <input
              type="url"
              value={referenceURL}
              onChange={e => setReferenceURL(e.target.value)}
              placeholder="https://..."
              style={fieldStyle}
            />
          </label>
        )}

        {sourceType === 'pdf' && !editing && (
          <label style={labelStyle}>
            <span style={labelText}>PDF File (max 20 MB)</span>
            <input
              type="file"
              accept="application/pdf"
              onChange={e => setPdfFile(e.target.files?.[0] ?? null)}
              style={{ color: '#d4d4e8' }}
            />
          </label>
        )}

        {sourceType === 'pdf' && editing && (
          <div style={{ fontSize: 12, color: '#7878a0' }}>
            PDF statblock uploaded at creation — replacing it isn't supported here.
          </div>
        )}

        {error && <div style={{ color: '#e74c3c', fontSize: 13 }}>{error}</div>}

        <button
          type="submit"
          disabled={submitting}
          style={btnStyle('#e67e22', submitting)}
        >
          {submitting ? 'Saving…' : editing ? 'Save Changes' : 'Save Monster'}
        </button>
      </form>
    </div>
  )
}

const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: '100%', boxSizing: 'border-box' }
function btnStyle(bg: string, disabled = false): React.CSSProperties {
  return { padding: '10px 20px', fontSize: 14, background: disabled ? '#444' : bg, color: '#fff', border: 'none', borderRadius: 4, cursor: disabled ? 'default' : 'pointer' }
}
