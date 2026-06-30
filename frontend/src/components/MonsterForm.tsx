import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

type SourceType = 'none' | 'url' | 'pdf'

export function MonsterForm() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [sourceType, setSourceType] = useState<SourceType>('none')
  const [referenceURL, setReferenceURL] = useState('')
  const [pdfFile, setPdfFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      let res: Response
      if (sourceType === 'pdf' && pdfFile) {
        const form = new FormData()
        form.append('name', name.trim())
        form.append('max_hp', maxHP)
        form.append('source_type', 'pdf')
        form.append('pdf', pdfFile)
        res = await fetch('/api/monsters', { method: 'POST', body: form })
      } else {
        const body: Record<string, unknown> = { name: name.trim(), max_hp: parseInt(maxHP, 10) }
        if (sourceType === 'url' && referenceURL.trim()) {
          body.source_type = 'url'
          body.reference_url = referenceURL.trim()
        }
        res = await fetch('/api/monsters', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        })
      }

      if (res.status === 413) { setError('PDF file is too large (max 20 MB).'); return }
      if (!res.ok) { setError(`Error: ${await res.text()}`); return }

      setSaved(true)
    } catch {
      setError('Network error. Is the server running?')
    } finally {
      setLoading(false)
    }
  }

  if (saved) {
    return (
      <div style={{ maxWidth: 480, margin: '60px auto', padding: 24, textAlign: 'center' }}>
        <div style={{ fontSize: 20, marginBottom: 16, color: '#27ae60' }}>Monster saved!</div>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
          <button
            onClick={() => { setName(''); setMaxHP(''); setSourceType('none'); setReferenceURL(''); setPdfFile(null); setSaved(false) }}
            style={btnStyle('#2e2e48')}
          >
            Add Another
          </button>
          <button onClick={() => navigate('/')} style={btnStyle('#2980b9')}>Back to Join</button>
        </div>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 480, margin: '40px auto', padding: 24 }}>
      <h2 style={{ marginTop: 0 }}>Register Monster</h2>
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
          <span style={labelText}>Statblock Reference</span>
          <div style={{ display: 'flex', gap: 8 }}>
            {(['none', 'url', 'pdf'] as SourceType[]).map(t => (
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

        {sourceType === 'pdf' && (
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

        {error && <div style={{ color: '#e74c3c', fontSize: 13 }}>{error}</div>}

        <button
          type="submit"
          disabled={loading}
          style={btnStyle('#e67e22', loading)}
        >
          {loading ? 'Saving…' : 'Save Monster'}
        </button>
      </form>
    </div>
  )
}

const labelStyle: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 4 }
const labelText: React.CSSProperties = { fontSize: 12, color: '#7878a0' }
const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: '100%', boxSizing: 'border-box' }
function btnStyle(bg: string, disabled = false): React.CSSProperties {
  return { padding: '10px 20px', fontSize: 14, background: disabled ? '#444' : bg, color: '#fff', border: 'none', borderRadius: 4, cursor: disabled ? 'default' : 'pointer' }
}
