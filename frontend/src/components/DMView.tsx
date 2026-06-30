import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import type { Entity, RoomState } from '../types'

const CONDITIONS = ['Prone', 'Stunned', 'Poisoned', 'Blinded', 'Frightened', 'Incapacitated', 'Restrained', 'Paralyzed']

function parseHP(input: string, current: number, max: number): number {
  const s = input.trim()
  let result: number
  if (s.startsWith('+')) {
    const delta = parseInt(s.slice(1), 10)
    result = isNaN(delta) ? current : current + delta
  } else if (s.startsWith('-')) {
    const delta = parseInt(s.slice(1), 10)
    result = isNaN(delta) ? current : current - delta
  } else {
    const val = parseInt(s, 10)
    result = isNaN(val) ? current : val
  }
  return Math.max(0, Math.min(result, max))
}

interface EntityRowProps {
  entity: Entity
  isActive: boolean
  sendMessage: (msg: object) => void
}

function EntityRow({ entity, isActive, sendMessage }: EntityRowProps) {
  const [expanded, setExpanded] = useState(false)
  const [hpInput, setHpInput] = useState('')
  const [initInput, setInitInput] = useState(String(entity.initiative))
  const [nameInput, setNameInput] = useState(entity.name)

  function sendUpdate(overrides: Partial<{
    name: string; current_hp: number; temp_hp: number
    initiative: number; conditions: string[]; dead: boolean
  }>) {
    sendMessage({
      type: 'dm_update_entity',
      entity_id: entity.id,
      name: entity.name,
      current_hp: entity.current_hp,
      temp_hp: entity.temp_hp,
      initiative: entity.initiative,
      conditions: entity.conditions,
      dead: entity.dead,
      ...overrides,
    })
  }

  function applyHP() {
    if (!hpInput.trim()) return
    sendUpdate({ current_hp: parseHP(hpInput, entity.current_hp, entity.max_hp) })
    setHpInput('')
  }

  function applyInitiative() {
    const val = parseInt(initInput, 10)
    if (!isNaN(val) && val !== entity.initiative) {
      sendUpdate({ initiative: val })
    }
  }

  function applyName() {
    if (nameInput.trim() && nameInput.trim() !== entity.name) {
      sendUpdate({ name: nameInput.trim() })
    }
  }

  function toggleCondition(cond: string) {
    const next = entity.conditions.includes(cond)
      ? entity.conditions.filter(c => c !== cond)
      : [...entity.conditions, cond]
    sendUpdate({ conditions: next })
  }

  const rowBg = entity.dead ? '#f0f0f0' : isActive ? '#fff8e1' : 'white'
  const textColor = entity.dead ? '#aaa' : 'inherit'

  return (
    <div style={{ borderBottom: '1px solid #eee' }}>
      {/* Main row */}
      <div
        style={{ padding: '10px 14px', background: rowBg, display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}
        onClick={() => setExpanded(e => !e)}
      >
        <span style={{ width: 16, color: '#e67e22', flexShrink: 0 }}>{isActive ? '▶' : ''}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <span style={{ fontWeight: 600, color: textColor }}>{entity.name}</span>
          {entity.dead && <span style={{ marginLeft: 6, fontSize: 11, color: '#e74c3c' }}>💀 dead</span>}
          <span style={{ marginLeft: 6, fontSize: 11, color: '#aaa' }}>{entity.type}</span>
          {entity.conditions.length > 0 && (
            <div style={{ fontSize: 11, color: '#e67e22', marginTop: 2 }}>
              {entity.conditions.join(' · ')}
            </div>
          )}
        </div>
        <div style={{ fontSize: 13, color: textColor, flexShrink: 0, textAlign: 'right' }}>
          {entity.current_hp}/{entity.max_hp} HP
          {entity.temp_hp > 0 && <span style={{ color: '#3498db' }}> +{entity.temp_hp}</span>}
        </div>
        <div style={{ fontSize: 12, color: '#888', width: 36, textAlign: 'right', flexShrink: 0 }}>
          {entity.initiative}
        </div>
        <span style={{ fontSize: 11, color: '#aaa', flexShrink: 0 }}>{expanded ? '▲' : '▼'}</span>
      </div>

      {/* Expanded edit panel */}
      {expanded && (
        <div style={{ padding: '12px 16px 16px 16px', background: '#fafafa', borderTop: '1px solid #eee' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 12 }}>

            {/* HP smart input */}
            <div>
              <div style={{ fontSize: 11, color: '#888', marginBottom: 4 }}>HP (+5, -12, or 20)</div>
              <div style={{ display: 'flex', gap: 4 }}>
                <input
                  type="text"
                  value={hpInput}
                  onChange={e => setHpInput(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') applyHP() }}
                  placeholder={String(entity.current_hp)}
                  style={{ width: 90, padding: '6px 8px', fontSize: 14 }}
                />
                <button onClick={applyHP} style={{ padding: '6px 10px', fontSize: 13 }}>Apply</button>
              </div>
            </div>

            {/* Initiative */}
            <div>
              <div style={{ fontSize: 11, color: '#888', marginBottom: 4 }}>Initiative</div>
              <div style={{ display: 'flex', gap: 4 }}>
                <input
                  type="number"
                  value={initInput}
                  onChange={e => setInitInput(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') applyInitiative() }}
                  style={{ width: 70, padding: '6px 8px', fontSize: 14 }}
                />
                <button onClick={applyInitiative} style={{ padding: '6px 10px', fontSize: 13 }}>Set</button>
              </div>
            </div>

            {/* Creature name (creatures only) */}
            {entity.type === 'creature' && (
              <div>
                <div style={{ fontSize: 11, color: '#888', marginBottom: 4 }}>Name</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <input
                    type="text"
                    value={nameInput}
                    onChange={e => setNameInput(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') applyName() }}
                    style={{ width: 120, padding: '6px 8px', fontSize: 14 }}
                  />
                  <button onClick={applyName} style={{ padding: '6px 10px', fontSize: 13 }}>Set</button>
                </div>
              </div>
            )}
          </div>

          {/* Condition toggles */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
            {CONDITIONS.map(cond => {
              const active = entity.conditions.includes(cond)
              return (
                <button
                  key={cond}
                  onClick={() => toggleCondition(cond)}
                  style={{
                    padding: '3px 9px', fontSize: 12, borderRadius: 12, border: '1px solid',
                    borderColor: active ? '#e74c3c' : '#ccc',
                    background: active ? '#fde8e8' : 'white',
                    color: active ? '#c0392b' : '#555',
                    cursor: 'pointer',
                  }}
                >
                  {cond}
                </button>
              )
            })}
          </div>

          {/* Action buttons */}
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              onClick={() => sendUpdate({ dead: !entity.dead })}
              style={{
                padding: '6px 14px', fontSize: 13, cursor: 'pointer',
                background: entity.dead ? '#27ae60' : '#e74c3c',
                color: 'white', border: 'none', borderRadius: 4,
              }}
            >
              {entity.dead ? 'Revive' : 'Kill'}
            </button>
            <button
              onClick={() => sendMessage({ type: 'remove_entity', entity_id: entity.id })}
              style={{ padding: '6px 14px', fontSize: 13, cursor: 'pointer', background: '#666', color: 'white', border: 'none', borderRadius: 4 }}
            >
              Remove
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

interface AddCreatureFormProps {
  sendMessage: (msg: object) => void
}

function AddCreatureForm({ sendMessage }: AddCreatureFormProps) {
  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [initiative, setInitiative] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    const init = parseInt(initiative, 10)
    if (!name.trim() || !hp || hp <= 0) return
    sendMessage({ type: 'add_creature', name: name.trim(), max_hp: hp, initiative: init || 0 })
    setName('')
    setMaxHP('')
    setInitiative('')
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end', padding: '12px 0' }}>
      <label style={labelStyle}>
        <span style={labelText}>Name</span>
        <input value={name} onChange={e => setName(e.target.value)} placeholder="Goblin" required style={fieldStyle} />
      </label>
      <label style={labelStyle}>
        <span style={labelText}>Max HP</span>
        <input type="number" value={maxHP} onChange={e => setMaxHP(e.target.value)} placeholder="14" min={1} required style={{ ...fieldStyle, width: 70 }} />
      </label>
      <label style={labelStyle}>
        <span style={labelText}>Initiative</span>
        <input type="number" value={initiative} onChange={e => setInitiative(e.target.value)} placeholder="11" style={{ ...fieldStyle, width: 70 }} />
      </label>
      <button type="submit" style={{ padding: '8px 16px', fontSize: 14, alignSelf: 'flex-end' }}>
        Add Creature
      </button>
    </form>
  )
}

const labelStyle: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 2 }
const labelText: React.CSSProperties = { fontSize: 11, color: '#888' }
const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: 120 }

interface DMViewProps {
  roomState: RoomState
  sendMessage: (msg: object) => void
}

export function DMView({ roomState, sendMessage }: DMViewProps) {
  const { entities, active_index, is_started, round } = roomState
  const hasDeadCreatures = entities.some(e => e.dead && e.type === 'creature')
  const [confirmingEnd, setConfirmingEnd] = useState(false)

  // Task 3.3: clear stale confirmation if combat ends externally
  useEffect(() => {
    if (!is_started) setConfirmingEnd(false)
  }, [is_started])

  return (
    <div style={{ maxWidth: 720, margin: '0 auto', padding: 16, fontFamily: 'sans-serif' }}>

      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>⚔ DM Panel</h2>
        {is_started && <span style={{ fontSize: 18, fontWeight: 'bold' }}>Round {round}</span>}
      </div>

      {/* Combat controls */}
      <div style={{ display: 'flex', gap: 10, marginBottom: 16, alignItems: 'center', flexWrap: 'wrap' }}>
        {!is_started ? (
          <button
            onClick={() => sendMessage({ type: 'start_combat' })}
            style={{ padding: '10px 20px', fontSize: 15, fontWeight: 'bold', cursor: 'pointer', background: '#27ae60', color: 'white', border: 'none', borderRadius: 4 }}
          >
            ▶ Start Combat
          </button>
        ) : (
          <>
            <button
              onClick={() => sendMessage({ type: 'next_turn' })}
              style={{ padding: '10px 20px', fontSize: 15, fontWeight: 'bold', cursor: 'pointer', background: '#2980b9', color: 'white', border: 'none', borderRadius: 4 }}
            >
              Next Turn →
            </button>
            {!confirmingEnd ? (
              <button
                onClick={() => setConfirmingEnd(true)}
                style={{ padding: '10px 20px', fontSize: 14, cursor: 'pointer', background: '#c0392b', color: 'white', border: 'none', borderRadius: 4 }}
              >
                ⚠ End Combat
              </button>
            ) : (
              <span style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 13, color: '#c0392b' }}>End this encounter? Creatures will be removed.</span>
                <button
                  onClick={() => setConfirmingEnd(false)}
                  style={{ padding: '8px 14px', fontSize: 13, cursor: 'pointer', borderRadius: 4 }}
                >
                  Cancel
                </button>
                <button
                  onClick={() => { sendMessage({ type: 'end_combat' }); setConfirmingEnd(false) }}
                  style={{ padding: '8px 14px', fontSize: 13, cursor: 'pointer', background: '#c0392b', color: 'white', border: 'none', borderRadius: 4 }}
                >
                  Yes, End Combat
                </button>
              </span>
            )}
          </>
        )}
      </div>

      {/* Add creature form */}
      <div style={{ border: '1px solid #ddd', borderRadius: 6, padding: '0 14px', marginBottom: 12 }}>
        <AddCreatureForm sendMessage={sendMessage} />
      </div>

      {/* Remove all dead creatures */}
      {hasDeadCreatures && (
        <div style={{ marginBottom: 12 }}>
          <button
            onClick={() => sendMessage({ type: 'remove_dead_creatures' })}
            style={{ padding: '7px 14px', fontSize: 13, cursor: 'pointer', background: '#7f8c8d', color: 'white', border: 'none', borderRadius: 4 }}
          >
            🗑 Remove All Dead Creatures
          </button>
        </div>
      )}

      {/* Initiative tracker */}
      <div style={{ border: '1px solid #ddd', borderRadius: 6, overflow: 'hidden' }}>
        {entities.length === 0 && (
          <div style={{ padding: 16, color: '#888' }}>No combatants yet. Add creatures above or wait for players to join.</div>
        )}
        {entities.map((entity, i) => (
          <EntityRow
            key={entity.id}
            entity={entity}
            isActive={is_started && i === active_index}
            sendMessage={sendMessage}
          />
        ))}
      </div>
    </div>
  )
}
