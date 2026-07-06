import { useState } from 'react'
import type { Encounter } from '../types'
import { color, radius } from '../theme'

interface EncounterTemplatesControlProps {
  sendMessage: (msg: object) => void
  edition: string
}

export function EncounterTemplatesControl({ sendMessage, edition }: EncounterTemplatesControlProps) {
  const [open, setOpen] = useState(false)
  const [encounters, setEncounters] = useState<Encounter[]>([])
  const [loaded, setLoaded] = useState(false)

  function toggleOpen() {
    const next = !open
    setOpen(next)
    if (next && !loaded) {
      fetch(`/api/encounters?edition=${encodeURIComponent(edition)}`)
        .then(res => res.ok ? res.json() as Promise<Encounter[]> : undefined)
        .then(data => {
          if (data === undefined) { setEncounters([]); return }
          setEncounters(data)
          setLoaded(true)
        })
        .catch(() => setEncounters([]))
    }
  }

  function inject(encounterId: string) {
    sendMessage({ type: 'inject_encounter', encounter_id: encounterId })
    setOpen(false)
  }

  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={toggleOpen}
        style={{ padding: '10px 20px', fontSize: 14, cursor: 'pointer', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm }}
      >
        📋 Encounter Templates
      </button>
      {open && (
        <div style={{
          position: 'absolute', top: '100%', left: 0, zIndex: 10, minWidth: 220,
          background: '#1a1a38', border: `1px solid ${color.border}`, borderRadius: radius.sm,
          marginTop: 2, maxHeight: 240, overflowY: 'auto',
        }}>
          {encounters.length === 0 && (
            <div style={{ padding: 10, fontSize: 13, color: color.textSecondary }}>No saved encounters for this edition.</div>
          )}
          {encounters.map(enc => (
            <button
              key={enc.id}
              type="button"
              onClick={() => inject(enc.id)}
              style={{
                width: '100%', padding: '8px 10px', cursor: 'pointer', fontSize: 13, color: color.textPrimary,
                border: 'none', borderRadius: 0, borderBottom: `1px solid ${color.border}`,
                background: 'none', font: 'inherit', textAlign: 'left',
              }}
            >
              {enc.name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
