import { useState } from 'react'
import type { Encounter } from '../types'

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
        .then(res => res.ok ? res.json() : [])
        .then((data: Encounter[]) => { setEncounters(data); setLoaded(true) })
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
        style={{ padding: '10px 20px', fontSize: 14, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
      >
        📋 Encounter Templates
      </button>
      {open && (
        <div style={{
          position: 'absolute', top: '100%', left: 0, zIndex: 10, minWidth: 220,
          background: '#1a1a38', border: '1px solid #2e2e48', borderRadius: 4,
          marginTop: 2, maxHeight: 240, overflowY: 'auto',
        }}>
          {encounters.length === 0 && (
            <div style={{ padding: 10, fontSize: 13, color: '#7878a0' }}>No saved encounters for this edition.</div>
          )}
          {encounters.map(enc => (
            <div
              key={enc.id}
              onClick={() => inject(enc.id)}
              style={{ padding: '8px 10px', cursor: 'pointer', fontSize: 13, color: '#d4d4e8', borderBottom: '1px solid #2e2e48' }}
            >
              {enc.name}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
