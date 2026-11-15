import { useState } from 'react'
import type { ImportChange, ImportPreview, ImportResult } from '../types'
import { color, radius } from '../theme'

interface Props {
  /** Where the PDF is posted: an existing PC's import route, or the new-character one. */
  uploadUrl: string
  title: string
  onClose: () => void
  onConfirmed: (result: ImportResult) => void
}

type Phase =
  | { kind: 'pick' }
  | { kind: 'uploading' }
  | { kind: 'preview'; preview: ImportPreview }
  | { kind: 'confirming'; preview: ImportPreview }

const show = (v: unknown) => (typeof v === 'string' ? v : JSON.stringify(v))

async function messageOf(res: Response, fallback: string) {
  const text = (await res.text().catch(() => '')).trim()
  return text || fallback
}

async function discard(importId: string) {
  try {
    await fetch(`/api/sheet-imports/${encodeURIComponent(importId)}`, { method: 'DELETE' })
  } catch {
    // The draft expires on its own.
  }
}

/**
 * Uploads an MPMB PDF, shows what importing it would change, and applies it
 * only on confirm. Cancelling discards the server-side draft.
 */
export function ImportSheetDialog({ uploadUrl, title, onClose, onConfirmed }: Props) {
  const [phase, setPhase] = useState<Phase>({ kind: 'pick' })
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState(false)

  async function upload(file: File) {
    setError(null)
    setConflict(false)
    setPhase({ kind: 'uploading' })
    const body = new FormData()
    body.append('file', file)
    try {
      const res = await fetch(uploadUrl, { method: 'POST', body })
      if (res.status === 429) {
        setError('Another import is being processed. Try again in a moment.')
      } else if (!res.ok) {
        setError(await messageOf(res, 'The file could not be imported.'))
      } else {
        setPhase({ kind: 'preview', preview: await res.json() as ImportPreview })
        return
      }
    } catch {
      setError('The upload failed. Check your connection and try again.')
    }
    setPhase({ kind: 'pick' })
  }

  async function confirm(preview: ImportPreview) {
    setError(null)
    setPhase({ kind: 'confirming', preview })
    try {
      const res = await fetch(`/api/sheet-imports/${encodeURIComponent(preview.import_id)}/confirm`, { method: 'POST' })
      if (res.ok) {
        onConfirmed(await res.json() as ImportResult)
        return
      }
      if (res.status === 409) {
        setConflict(true)
        void discard(preview.import_id)
        setPhase({ kind: 'pick' })
        return
      }
      setError(await messageOf(res, 'The import could not be applied.'))
    } catch {
      setError('The import failed. Check your connection and try again.')
    }
    setPhase({ kind: 'preview', preview })
  }

  function cancel() {
    if (phase.kind === 'preview') void discard(phase.preview.import_id)
    onClose()
  }

  const busy = phase.kind === 'uploading' || phase.kind === 'confirming'
  const preview = phase.kind === 'preview' || phase.kind === 'confirming' ? phase.preview : null

  return (
    <div role="dialog" aria-modal="true" aria-label={title} style={overlay}>
      <div style={panel}>
        <h2 style={{ margin: '0 0 12px', fontSize: 18 }}>{title}</h2>

        {conflict && (
          <div role="alert" style={{ ...alertStyle, borderColor: color.accent, color: color.accent }}>
            The sheet changed after you uploaded this file. Upload it again to review the new differences.
          </div>
        )}
        {error && <div role="alert" style={alertStyle}>{error}</div>}

        {!preview && (
          <label style={{ display: 'block', fontSize: 13, color: color.textSecondary }}>
            {conflict ? 'Upload again' : 'MPMB character sheet (PDF, not flattened)'}
            <input
              type="file"
              accept="application/pdf,.pdf"
              disabled={busy}
              onChange={e => { const f = e.target.files?.[0]; if (f) void upload(f) }}
              style={{ display: 'block', marginTop: 8 }}
            />
          </label>
        )}
        {phase.kind === 'uploading' && <div style={{ marginTop: 12, fontSize: 13 }}>Reading the sheet…</div>}

        {preview && <PreviewBody preview={preview} />}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
          <button onClick={cancel} disabled={phase.kind === 'confirming'} style={secondaryBtn}>Cancel</button>
          {preview && (
            <button onClick={() => confirm(preview)} disabled={busy} style={primaryBtn}>
              {phase.kind === 'confirming' ? 'Importing…' : preview.new_character ? 'Create character' : 'Replace sheet'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function PreviewBody({ preview }: { preview: ImportPreview }) {
  const bySection = new Map<string, ImportChange[]>()
  for (const c of preview.changes) bySection.set(c.section, [...(bySection.get(c.section) ?? []), c])
  const { companions, inventory } = preview
  const noDifferences = preview.changes.length === 0 && companions.removed.length === 0 && companions.added.length === 0

  return (
    <div style={{ fontSize: 13 }}>
      <p style={{ margin: '0 0 8px' }}>
        {preview.new_character ? 'New character' : 'Importing into'} <strong>{preview.name || '(no name)'}</strong>
        {preview.max_hp > 0 && <> · {preview.max_hp} HP</>}
        {preview.source.file_name && <span style={{ color: color.textMuted }}> · {preview.source.file_name}</span>}
      </p>

      {preview.warnings.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          <strong>Warnings</strong>
          <ul aria-label="Warnings" style={{ margin: '4px 0', paddingLeft: 18, color: color.accent }}>
            {preview.warnings.map((w, i) => <li key={i}>{w}</li>)}
          </ul>
        </div>
      )}

      {!preview.new_character && (
        <p style={{ margin: '0 0 8px', color: color.textSecondary }}>
          Replacing overwrites the whole sheet. Anything you edited by hand that is not in the PDF will be lost.
        </p>
      )}
      {noDifferences && <p>No differences.</p>}

      {[...bySection.entries()].map(([section, changes]) => (
        <div key={section} style={{ marginBottom: 10 }}>
          <strong>{section.replace(/_/g, ' ')}</strong>
          <ul aria-label={`Changes in ${section}`} style={{ margin: '4px 0', paddingLeft: 18 }}>
            {changes.map((c, i) => <li key={i}><ChangeLine change={c} /></li>)}
          </ul>
        </div>
      ))}

      {(companions.removed.length > 0 || companions.added.length > 0) && (
        <div style={{ marginBottom: 10 }}>
          <strong>companions</strong>
          <ul aria-label="Changes in companions" style={{ margin: '4px 0', paddingLeft: 18 }}>
            {companions.removed.map(n => <li key={`r-${n}`}>Removed: {n}</li>)}
            {companions.added.map(n => <li key={`a-${n}`}>Added: {n}</li>)}
          </ul>
        </div>
      )}

      <div>
        <strong>inventory</strong>
        <div>
          {inventory.action === 'keep'
            ? 'The PDF has no items or coins, so the current inventory is kept.'
            : inventory.changed
              ? `Items and coins will be replaced (${inventory.old_items} → ${inventory.new_items} items).`
              : 'Items and coins match the PDF.'}
        </div>
      </div>
    </div>
  )
}

function ChangeLine({ change }: { change: ImportChange }) {
  const label = change.path || '(whole section)'
  if (change.kind === 'added') return <>Added {label}: {show(change.new)}</>
  if (change.kind === 'removed') return <>Removed {label}: {show(change.old)}</>
  return <>{label}: {show(change.old)} → {show(change.new)}</>
}

const overlay: React.CSSProperties = {
  position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex',
  alignItems: 'center', justifyContent: 'center', zIndex: 100, padding: 16,
}
const panel: React.CSSProperties = {
  background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.md,
  padding: 20, width: '100%', maxWidth: 640, maxHeight: '90vh', overflowY: 'auto', color: color.textPrimary,
}
const alertStyle: React.CSSProperties = {
  padding: '10px 12px', border: `1px solid ${color.danger}`, borderRadius: radius.md,
  color: color.danger, fontSize: 13, marginBottom: 12,
}
const primaryBtn: React.CSSProperties = {
  background: color.accent, color: '#fff', border: 'none', borderRadius: radius.md, padding: '8px 14px', cursor: 'pointer',
}
const secondaryBtn: React.CSSProperties = {
  background: 'none', color: color.textSecondary, border: `1px solid ${color.border}`, borderRadius: radius.md, padding: '8px 14px', cursor: 'pointer',
}
