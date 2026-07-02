import { useEffect, useState } from 'react'
import type { CustomMonster, Encounter } from '../types'

interface DMNavColumnProps {
  sendMessage: (msg: object) => void
  edition: string
  onSelectCustomMonster: (monster: CustomMonster) => void
}

const sectionHeading: React.CSSProperties = {
  fontSize: 12, fontWeight: 600, color: '#7878a0',
  textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 8,
}
const emptyText: React.CSSProperties = { fontSize: 12, color: '#454568', padding: '4px 0' }
const navItem: React.CSSProperties = {
  padding: '8px 10px', cursor: 'pointer', fontSize: 13, color: '#d4d4e8',
  borderRadius: 4, marginBottom: 4, background: '#1a1a2c', border: '1px solid transparent',
}

// Persistent, always-visible sibling to the phone tier's Encounter Templates
// dropdown and inline "My Creatures" quick-pick — same fetches, same click
// behaviors, just not collapsed behind a toggle.
export function DMNavColumn({ sendMessage, edition, onSelectCustomMonster }: DMNavColumnProps) {
  const [encounters, setEncounters] = useState<Encounter[]>([])
  const [myCreatures, setMyCreatures] = useState<CustomMonster[]>([])

  useEffect(() => {
    fetch(`/api/encounters?edition=${encodeURIComponent(edition)}`)
      .then(res => res.ok ? res.json() : [])
      .then((data: Encounter[]) => setEncounters(data))
      .catch(() => setEncounters([]))
  }, [edition])

  useEffect(() => {
    fetch(`/api/custom-monsters?edition=${encodeURIComponent(edition)}`)
      .then(res => res.ok ? res.json() : [])
      .then((data: CustomMonster[]) => setMyCreatures(data))
      .catch(() => setMyCreatures([]))
  }, [edition])

  function inject(encounterId: string) {
    sendMessage({ type: 'inject_encounter', encounter_id: encounterId })
  }

  return (
    <div style={{
      height: '100%', minHeight: 0, overflowY: 'auto',
      border: '1px solid #2e2e48', borderRadius: 6, background: '#161626', padding: 14,
    }}>
      <div style={{ marginBottom: 20 }}>
        <div style={sectionHeading}>📋 Encounters</div>
        {encounters.length === 0 && (
          <div style={emptyText}>No saved encounters for this edition.</div>
        )}
        {encounters.map(enc => (
          <div key={enc.id} onClick={() => inject(enc.id)} style={navItem}>
            {enc.name}
          </div>
        ))}
      </div>
      <div>
        <div style={sectionHeading}>My Creatures</div>
        {myCreatures.length === 0 && (
          <div style={emptyText}>No custom monsters for this edition.</div>
        )}
        {myCreatures.map(m => (
          <div key={m.id} onClick={() => onSelectCustomMonster(m)} style={navItem}>
            {m.name} <span style={{ color: '#7878a0' }}>{m.max_hp} HP</span>
          </div>
        ))}
      </div>
    </div>
  )
}
