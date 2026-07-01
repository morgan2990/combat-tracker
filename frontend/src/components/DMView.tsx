import { useState, useEffect, useRef } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import type { Entity, RoomState, MonsterSearchHit } from '../types'
import { StatblockDrawer } from './StatblockDrawer'

const SEARCH_MIN_CHARS = 3
const SEARCH_DEBOUNCE_MS = 175

const CONDITIONS = ['Prone', 'Stunned', 'Poisoned', 'Blinded', 'Frightened', 'Incapacitated', 'Restrained', 'Paralyzed']

function entityVitalState(entity: Entity): 'dead' | 'unconscious' | 'alive' {
  if (entity.dead) return 'dead'
  if (entity.current_hp === 0) return 'unconscious'
  return 'alive'
}

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
  onStatblock?: () => void
}

function EntityRow({ entity, isActive, sendMessage, onStatblock }: EntityRowProps) {
  const [expanded, setExpanded] = useState(false)
  const [hpInput, setHpInput] = useState('')
  const [initInput, setInitInput] = useState(entity.initiative != null ? String(entity.initiative) : '')
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

  const vitalState = entityVitalState(entity)
  const rowBg =
    vitalState === 'dead'        ? '#141414' :
    vitalState === 'unconscious' ? '#1a1608' :
    isActive                     ? '#1f1508' : '#1a1a2c'
  const textColor =
    vitalState === 'dead'        ? '#585858' :
    vitalState === 'unconscious' ? '#9090a0' : '#d4d4e8'

  return (
    <div style={{ borderBottom: '1px solid #2e2e48' }}>
      {/* Main row */}
      <div
        style={{ padding: '10px 14px', background: rowBg, display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}
        onClick={() => setExpanded(e => !e)}
      >
        <span style={{ width: 16, color: '#e67e22', flexShrink: 0 }}>{isActive ? '▶' : ''}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <span style={{ fontWeight: 600, color: textColor }}>{entity.name}</span>
          {entity.type === 'creature' && entity.source_type && onStatblock && (
            <button
              onClick={e => { e.stopPropagation(); onStatblock() }}
              title="View statblock"
              style={{ marginLeft: 6, background: 'none', border: 'none', color: '#7878a0', cursor: 'pointer', fontSize: 13, padding: '0 2px', lineHeight: 1 }}
            >
              📋
            </button>
          )}
          {vitalState === 'dead' && <span style={{ marginLeft: 6, fontSize: 11, color: '#e74c3c' }}>💀 Dead</span>}
          {vitalState === 'unconscious' && <span style={{ marginLeft: 6, fontSize: 11, color: '#e67e22' }}>😵 Unconscious</span>}
          <span style={{ marginLeft: 6, fontSize: 11, color: '#454568' }}>{entity.type}</span>
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
        <div
          style={{ fontSize: 12, color: '#7878a0', width: 36, textAlign: 'right', flexShrink: 0, cursor: entity.initiative_roll != null ? 'help' : 'default' }}
          title={entity.initiative_roll != null && entity.initiative_modifier != null
            ? `d20: ${entity.initiative_roll} ${entity.initiative_modifier >= 0 ? '+' : ''}${entity.initiative_modifier} = ${entity.initiative}`
            : undefined}
        >
          {entity.initiative != null ? entity.initiative : '--'}
        </div>
        <span style={{ fontSize: 11, color: '#5a5a78', flexShrink: 0 }}>{expanded ? '▲' : '▼'}</span>
      </div>

      {/* Expanded edit panel */}
      {expanded && (
        <div style={{ padding: '12px 16px 16px 16px', background: '#161626', borderTop: '1px solid #2e2e48' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 12 }}>

            {/* HP smart input */}
            <div>
              <div style={{ fontSize: 11, color: '#7878a0', marginBottom: 4 }}>HP (+5, -12, or 20)</div>
              <div style={{ display: 'flex', gap: 4 }}>
                <input
                  type="text"
                  value={hpInput}
                  onChange={e => setHpInput(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') applyHP() }}
                  placeholder={String(entity.current_hp)}
                  style={{ width: 90, padding: '6px 8px', fontSize: 14 }}
                />
                <button onClick={applyHP} style={actionBtn}>Apply</button>
              </div>
            </div>

            {/* Initiative */}
            <div>
              <div style={{ fontSize: 11, color: '#7878a0', marginBottom: 4 }}>Initiative</div>
              <div style={{ display: 'flex', gap: 4 }}>
                <input
                  type="number"
                  value={initInput}
                  onChange={e => setInitInput(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') applyInitiative() }}
                  style={{ width: 70, padding: '6px 8px', fontSize: 14 }}
                />
                <button onClick={applyInitiative} style={actionBtn}>Set</button>
              </div>
            </div>

            {/* Creature name (creatures only) */}
            {entity.type === 'creature' && (
              <div>
                <div style={{ fontSize: 11, color: '#7878a0', marginBottom: 4 }}>Name</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <input
                    type="text"
                    value={nameInput}
                    onChange={e => setNameInput(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') applyName() }}
                    style={{ width: 120, padding: '6px 8px', fontSize: 14 }}
                  />
                  <button onClick={applyName} style={actionBtn}>Set</button>
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
                    borderColor: active ? '#e74c3c' : '#2e2e48',
                    background: active ? '#2a0808' : '#1a1a2c',
                    color: active ? '#e74c3c' : '#8888aa',
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
              onClick={() => entity.dead
                ? sendUpdate({ dead: false })
                : sendUpdate({ dead: true, current_hp: 0 })}
              style={{
                padding: '6px 14px', fontSize: 13, cursor: 'pointer',
                background: entity.dead ? '#27ae60' : '#e74c3c',
                color: '#fff', border: 'none', borderRadius: 4,
              }}
            >
              {entity.dead ? 'Revive' : 'Kill'}
            </button>
            <button
              onClick={() => sendMessage({ type: 'remove_entity', entity_id: entity.id })}
              style={{ padding: '6px 14px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
            >
              Remove
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

const actionBtn: React.CSSProperties = {
  padding: '6px 10px', fontSize: 13, cursor: 'pointer',
  background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4,
}

interface AddCreatureFormProps {
  sendMessage: (msg: object) => void
  edition: string
}

interface MonsterRef { source_type: string; reference_url: string; pdf_object_key: string; initiative_modifier: number | null }

function AddCreatureForm({ sendMessage, edition }: AddCreatureFormProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<MonsterSearchHit[]>([])
  const [showDropdown, setShowDropdown] = useState(false)

  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [monsterRef, setMonsterRef] = useState<MonsterRef | null>(null)

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)

    if (query.trim().length < SEARCH_MIN_CHARS) {
      setResults([])
      setShowDropdown(false)
      return
    }

    debounceRef.current = setTimeout(async () => {
      try {
        const ed = edition || '5e'
        const res = await fetch(`/api/search/monsters?q=${encodeURIComponent(query.trim())}&edition=${encodeURIComponent(ed)}`)
        if (!res.ok) return
        const hits = await res.json() as MonsterSearchHit[]
        setResults(hits)
        setShowDropdown(true)
      } catch {
        setResults([])
        setShowDropdown(false)
      }
    }, SEARCH_DEBOUNCE_MS)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [query, edition])

  async function selectMonster(hit: MonsterSearchHit) {
    setQuery('')
    setResults([])
    setShowDropdown(false)
    setName(hit.name)
    setMaxHP(String(hit.max_hp))

    try {
      const res = await fetch(`/api/monsters/${encodeURIComponent(hit.name)}`)
      if (!res.ok) { setMonsterRef(null); return }
      const m = await res.json() as { source_type?: string; reference_url?: string; pdf_object_key?: string; initiative_modifier?: number | null }
      setMonsterRef({
        source_type: m.source_type ?? '',
        reference_url: m.reference_url ?? '',
        pdf_object_key: m.pdf_object_key ?? '',
        initiative_modifier: m.initiative_modifier ?? hit.initiative_modifier,
      })
    } catch {
      setMonsterRef(null)
    }
  }

  function handleSearchKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && showDropdown && results[0]) {
      e.preventDefault()
      selectMonster(results[0])
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    const qty = Math.max(1, parseInt(quantity, 10) || 1)
    if (!name.trim() || !hp || hp <= 0) return
    const msg: Record<string, unknown> = {
      type: 'add_creature',
      name: name.trim(),
      max_hp: hp,
      quantity: qty,
      source_type: monsterRef?.source_type ?? '',
      reference_url: monsterRef?.reference_url ?? '',
      pdf_object_key: monsterRef?.pdf_object_key ?? '',
    }
    if (monsterRef?.initiative_modifier != null) {
      msg.initiative_modifier = monsterRef.initiative_modifier
    }
    sendMessage(msg)
    setName('')
    setMaxHP('')
    setQuantity('1')
    setMonsterRef(null)
  }

  return (
    <div style={{ padding: '12px 0' }}>
      <div style={{ position: 'relative', marginBottom: 8, maxWidth: 240 }}>
        <label style={labelStyle}>
          <span style={labelText}>Search monsters</span>
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleSearchKeyDown}
            placeholder="Type 3+ characters…"
            style={fieldStyle}
          />
        </label>
        {showDropdown && results.length > 0 && (
          <div style={{
            position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 10,
            background: '#1a1a38', border: '1px solid #2e2e48', borderRadius: 4,
            marginTop: 2, maxHeight: 220, overflowY: 'auto',
          }}>
            {results.map(hit => (
              <div
                key={hit.id}
                onClick={() => selectMonster(hit)}
                style={{
                  padding: '8px 10px', cursor: 'pointer', display: 'flex',
                  justifyContent: 'space-between', alignItems: 'center', fontSize: 13,
                  borderBottom: '1px solid #2e2e48',
                }}
              >
                <span style={{ color: '#d4d4e8' }}>{hit.name}</span>
                <span style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <span style={{ fontSize: 10, color: '#3498db', border: '1px solid #3498db', borderRadius: 3, padding: '1px 5px' }}>
                    {edition || '5e'}
                  </span>
                  <span style={{ color: '#7878a0' }}>{hit.max_hp} HP</span>
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <label style={labelStyle}>
          <span style={labelText}>Name</span>
          <input
            value={name}
            onChange={e => { setName(e.target.value); setMonsterRef(null) }}
            placeholder="Goblin"
            required
            style={fieldStyle}
          />
        </label>
        <label style={labelStyle}>
          <span style={labelText}>Max HP</span>
          <input type="number" value={maxHP} onChange={e => setMaxHP(e.target.value)} placeholder="14" min={1} required style={{ ...fieldStyle, width: 70 }} />
        </label>
        <label style={labelStyle}>
          <span style={labelText}>Qty</span>
          <input type="number" value={quantity} onChange={e => setQuantity(e.target.value)} min={1} style={{ ...fieldStyle, width: 52 }} />
        </label>
        {monsterRef && monsterRef.source_type && (
          <span style={{ fontSize: 11, color: '#27ae60', alignSelf: 'center', paddingBottom: 2 }}>statblock ready</span>
        )}
        <button type="submit" style={{ padding: '8px 16px', fontSize: 14, alignSelf: 'flex-end', background: '#e67e22', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
          Add Creature
        </button>
      </form>
    </div>
  )
}

const labelStyle: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 2 }
const labelText: React.CSSProperties = { fontSize: 11, color: '#7878a0' }
const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: 120 }

interface DMViewProps {
  roomState: RoomState
  sendMessage: (msg: object) => void
}

export function DMView({ roomState, sendMessage }: DMViewProps) {
  const { entities, active_index, is_started, round } = roomState
  const hasDeadCreatures = entities.some(e => e.dead && e.type === 'creature')
  const pendingInitiative = entities.filter(e => (e.type === 'pc' || e.type === 'companion') && e.initiative === null)
  const [confirmingEnd, setConfirmingEnd] = useState(false)
  const [openDrawerEntityId, setOpenDrawerEntityId] = useState<string | null>(null)

  useEffect(() => {
    if (!is_started) setConfirmingEnd(false)
  }, [is_started])

  return (
    <div style={{ maxWidth: 720, margin: '0 auto', padding: 16 }}>

      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <h2 style={{ margin: 0 }}>⚔ DM Panel</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {is_started && <span style={{ fontSize: 18, fontWeight: 'bold', color: '#e67e22' }}>Round {round}</span>}
        </div>
      </div>

      {/* Room info bar */}
      <div style={{ display: 'flex', gap: 20, marginBottom: 16, padding: '8px 12px', background: '#161626', border: '1px solid #2e2e48', borderRadius: 6, fontSize: 13, flexWrap: 'wrap' }}>
        <span><span style={{ color: '#7878a0' }}>Room Code: </span><strong style={{ letterSpacing: 1, color: '#d4d4e8' }}>{roomState.room_id}</strong></span>
      </div>

      {/* Combat controls */}
      <div style={{ display: 'flex', gap: 10, marginBottom: 16, alignItems: 'center', flexWrap: 'wrap' }}>
        {!is_started ? (
          <>
            <button
              onClick={() => sendMessage({ type: 'start_combat' })}
              disabled={pendingInitiative.length > 0}
              style={{
                padding: '10px 20px', fontSize: 15, fontWeight: 'bold',
                background: '#27ae60', color: '#fff', border: 'none', borderRadius: 4,
                cursor: pendingInitiative.length > 0 ? 'not-allowed' : 'pointer',
                opacity: pendingInitiative.length > 0 ? 0.45 : 1,
              }}
            >
              ▶ Start Combat
            </button>
            {pendingInitiative.length > 0 && (
              <span style={{ fontSize: 13, color: '#e67e22' }}>
                Waiting on initiative: {pendingInitiative.map(e => e.name).join(', ')}
              </span>
            )}
          </>
        ) : (
          <>
            <button
              onClick={() => sendMessage({ type: 'next_turn' })}
              style={{ padding: '10px 20px', fontSize: 15, fontWeight: 'bold', cursor: 'pointer', background: '#2980b9', color: '#fff', border: 'none', borderRadius: 4 }}
            >
              Next Turn →
            </button>
            {!confirmingEnd ? (
              <button
                onClick={() => setConfirmingEnd(true)}
                style={{ padding: '10px 20px', fontSize: 14, cursor: 'pointer', background: '#c0392b', color: '#fff', border: 'none', borderRadius: 4 }}
              >
                ⚠ End Combat
              </button>
            ) : (
              <span style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 13, color: '#e74c3c' }}>End this encounter? Creatures will be removed.</span>
                <button
                  onClick={() => setConfirmingEnd(false)}
                  style={{ padding: '8px 14px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
                >
                  Cancel
                </button>
                <button
                  onClick={() => { sendMessage({ type: 'end_combat' }); setConfirmingEnd(false) }}
                  style={{ padding: '8px 14px', fontSize: 13, cursor: 'pointer', background: '#c0392b', color: '#fff', border: 'none', borderRadius: 4 }}
                >
                  Yes, End Combat
                </button>
              </span>
            )}
          </>
        )}
      </div>

      {/* Add creature form */}
      <div style={{ border: '1px solid #2e2e48', borderRadius: 6, padding: '0 14px', marginBottom: 12, background: '#161626' }}>
        <AddCreatureForm sendMessage={sendMessage} edition={roomState.edition} />
      </div>

      {/* Remove all dead creatures */}
      {hasDeadCreatures && (
        <div style={{ marginBottom: 12 }}>
          <button
            onClick={() => sendMessage({ type: 'remove_dead_creatures' })}
            style={{ padding: '7px 14px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
          >
            🗑 Remove All Dead Creatures
          </button>
        </div>
      )}

      {/* Initiative tracker */}
      <div style={{ border: '1px solid #2e2e48', borderRadius: 6, overflow: 'hidden' }}>
        {entities.length === 0 && (
          <div style={{ padding: 16, color: '#7878a0' }}>No combatants yet. Add creatures above or wait for players to join.</div>
        )}
        {entities.map((entity, i) => (
          <EntityRow
            key={entity.id}
            entity={entity}
            isActive={is_started && i === active_index}
            sendMessage={sendMessage}
            onStatblock={entity.type === 'creature' && entity.source_type
              ? () => setOpenDrawerEntityId(id => id === entity.id ? null : entity.id)
              : undefined}
          />
        ))}
      </div>

      {entities.filter(e => e.type === 'creature' && e.source_type).map(e => (
        <StatblockDrawer
          key={e.id}
          entity={e}
          open={openDrawerEntityId === e.id}
          onClose={() => setOpenDrawerEntityId(null)}
        />
      ))}
    </div>
  )
}
