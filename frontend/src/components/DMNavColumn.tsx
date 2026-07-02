import { useEffect, useState } from 'react'
import type { CustomMonster, Encounter } from '../types'
import { CustomMonsterList, sectionHeading, emptyText, navItem } from './CustomMonsterList'
import { fetchJSON } from '../fetchJSON'

interface DMNavColumnProps {
  sendMessage: (msg: object) => void
  edition: string
  onSelectCustomMonster: (monster: CustomMonster) => void
}

// Persistent, always-visible sibling to the phone tier's Encounter Templates
// dropdown and inline "My Creatures" quick-pick — same fetches, same click
// behaviors, just not collapsed behind a toggle.
export function DMNavColumn({ sendMessage, edition, onSelectCustomMonster }: DMNavColumnProps) {
  const [encounters, setEncounters] = useState<Encounter[]>([])
  const [myCreatures, setMyCreatures] = useState<CustomMonster[]>([])

  useEffect(() => {
    fetchJSON<Encounter[]>(`/api/encounters?edition=${encodeURIComponent(edition)}`, []).then(setEncounters)
  }, [edition])

  useEffect(() => {
    fetchJSON<CustomMonster[]>(`/api/custom-monsters?edition=${encodeURIComponent(edition)}`, []).then(setMyCreatures)
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
      <CustomMonsterList monsters={myCreatures} onSelect={onSelectCustomMonster} />
    </div>
  )
}
