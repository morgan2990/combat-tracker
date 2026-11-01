import { useState } from 'react'
import type { FormEvent } from 'react'
import type { SheetBundle, SheetRecord } from '../types'
import { CollapsibleSection } from '../components/CollapsibleSection'
import { color, radius } from '../theme'
import { FieldsView } from './SheetView'
import { FieldsEditor } from './SheetEditor'
import { normalizeRecord, sectionCount, sectionValue, specFields, toRequestData } from './sheetSchema'
import type { SectionSpec } from './sheetSchema'

interface SectionCardProps {
  spec: SectionSpec
  // The PC or companion this section belongs to; saves go to its id.
  bundle: SheetBundle
  title?: string
  // Prefix for the persisted expand/collapse key.
  idPrefix: string
  editable: boolean
  // Re-fetches the sheet; called after a save and from the conflict banner.
  onChanged: () => Promise<void>
}

export function SectionCard({ spec, bundle, title, idPrefix, editable, onChanged }: SectionCardProps) {
  const fields = specFields(spec)
  const value = sectionValue(spec, bundle)
  const heading = title ?? spec.title

  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<SheetRecord>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState(false)

  function startEditing() {
    setDraft(normalizeRecord(fields, value))
    setError(null)
    setConflict(false)
    setEditing(true)
  }

  function cancel() {
    setEditing(false)
    setError(null)
    setConflict(false)
  }

  async function save(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setConflict(false)
    try {
      const res = await fetch(`/api/pcs/${encodeURIComponent(bundle.pc.id)}/sheet/${spec.key}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ revision: bundle.sheet.revision, data: toRequestData(spec, draft) }),
      })
      if (res.status === 409) {
        // Keep the draft on screen: the user decides whether to reload and retry.
        setConflict(true)
        return
      }
      if (!res.ok) {
        const detail = (await res.text()).trim()
        throw new Error(detail || `Failed to save (${res.status})`)
      }
      setEditing(false)
      await onChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const ctx = { sheet: bundle.sheet }
  const count = sectionCount(spec, value)

  return (
    <CollapsibleSection
      id={`${idPrefix}.${spec.key}`}
      title={heading}
      count={count}
      headerAction={editable && !editing ? (
        <button type="button" aria-label={`Edit ${heading}`} onClick={startEditing} style={smallBtn}>Edit</button>
      ) : undefined}
    >
      {editing ? (
        <form onSubmit={save}>
          {spec.note && <div style={noteStyle}>{spec.note}</div>}
          <FieldsEditor fields={fields} value={draft} onChange={setDraft} />
          {error && <div role="alert" style={alertStyle}>{error}</div>}
          {conflict && (
            <div role="alert" style={alertStyle}>
              This sheet was changed elsewhere. Your input is still here.{' '}
              <button type="button" onClick={() => { setConflict(false); void onChanged() }} style={linkBtn}>
                Reload latest
              </button>{' '}
              and save again to keep your version.
            </div>
          )}
          <div style={{ display: 'flex', gap: 8 }}>
            <button type="submit" disabled={saving} style={{ ...primaryBtn, opacity: saving ? 0.5 : 1 }}>
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button type="button" onClick={cancel} disabled={saving} style={smallBtn}>Cancel</button>
          </div>
        </form>
      ) : count === 0 ? (
        <div style={{ fontSize: 13, color: color.textMuted }}>Nothing recorded yet.</div>
      ) : (
        <FieldsView fields={fields} value={value} ctx={ctx} />
      )}
    </CollapsibleSection>
  )
}

const smallBtn: React.CSSProperties = {
  padding: '3px 10px', fontSize: 12, cursor: 'pointer', background: color.border, color: color.textPrimary,
  border: 'none', borderRadius: radius.sm,
}
const primaryBtn: React.CSSProperties = {
  padding: '6px 14px', fontSize: 13, cursor: 'pointer', background: color.accent, color: '#fff',
  border: 'none', borderRadius: radius.sm,
}
const linkBtn: React.CSSProperties = {
  background: 'none', border: 'none', padding: 0, color: color.info, cursor: 'pointer', fontSize: 'inherit', textDecoration: 'underline',
}
const noteStyle: React.CSSProperties = { fontSize: 12, color: color.textMuted, marginBottom: 8 }
const alertStyle: React.CSSProperties = {
  margin: '8px 0', padding: '8px 10px', background: '#1a0808', border: `1px solid ${color.danger}`,
  borderRadius: radius.md, fontSize: 13, color: color.danger,
}
