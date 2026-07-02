import { useState } from 'react'
import type { Entity } from '../types'
import { ConditionToggles } from './ConditionToggles'
import { entityVitalState, vitalRowBg, vitalTextColor } from '../entityVitals'

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
  onOpenInventory: (pcId: string) => void
}

export function EntityRow({ entity, isActive, sendMessage, onStatblock, onOpenInventory }: EntityRowProps) {
  const [expanded, setExpanded] = useState(false)
  const [hpInput, setHpInput] = useState('')
  const [initInput, setInitInput] = useState(entity.initiative != null ? String(entity.initiative) : '')
  const [nameInput, setNameInput] = useState(entity.name)
  const [aliasInput, setAliasInput] = useState(entity.display_name ?? '')

  function sendUpdate(overrides: Partial<{
    name: string; current_hp: number; temp_hp: number
    initiative: number; conditions: string[]; dead: boolean; display_name: string
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
      display_name: entity.display_name ?? '',
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

  function applyDisplayName() {
    if (aliasInput.trim() !== (entity.display_name ?? '')) {
      sendUpdate({ display_name: aliasInput.trim() })
    }
  }

  function toggleCondition(cond: string) {
    const next = entity.conditions.includes(cond)
      ? entity.conditions.filter(c => c !== cond)
      : [...entity.conditions, cond]
    sendUpdate({ conditions: next })
  }

  const vitalState = entity.type === 'lair_action' ? 'alive' : entityVitalState(entity.dead, entity.current_hp)
  const rowBg = vitalRowBg(vitalState, isActive)
  const textColor = vitalTextColor(vitalState)

  return (
    <div style={{ borderBottom: '1px solid #2e2e48' }}>
      {/* Main row */}
      <div
        style={{ padding: '10px 14px', background: rowBg, display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', opacity: entity.is_hidden ? 0.5 : 1 }}
        onClick={() => setExpanded(e => !e)}
      >
        <span style={{ width: 16, color: '#e67e22', flexShrink: 0 }}>{isActive ? '▶' : ''}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <span style={{ fontWeight: 600, color: textColor }}>
            {entity.display_name ? `${entity.display_name} (${entity.name})` : entity.name}
          </span>
          {entity.type === 'pc' && entity.pc_id && (
            <button
              onClick={e => { e.stopPropagation(); onOpenInventory(entity.pc_id!) }}
              title="View inventory"
              style={{ marginLeft: 6, background: 'none', border: 'none', color: '#7878a0', cursor: 'pointer', fontSize: 13, padding: '0 2px', lineHeight: 1 }}
            >
              🎒
            </button>
          )}
          {entity.type === 'creature' && entity.source_type && onStatblock && (
            <button
              onClick={e => { e.stopPropagation(); onStatblock() }}
              title="View statblock"
              style={{ marginLeft: 6, background: 'none', border: 'none', color: '#7878a0', cursor: 'pointer', fontSize: 13, padding: '0 2px', lineHeight: 1 }}
            >
              📋
            </button>
          )}
          {(entity.type === 'creature' || entity.type === 'lair_action') && (
            <button
              onClick={e => { e.stopPropagation(); sendMessage({ type: 'toggle_entity_visibility', entity_id: entity.id }) }}
              title={entity.is_hidden ? 'Hidden from players — click to reveal' : 'Visible to players — click to hide'}
              style={{ marginLeft: 6, background: 'none', border: 'none', color: '#7878a0', cursor: 'pointer', fontSize: 13, padding: '0 2px', lineHeight: 1 }}
            >
              {entity.is_hidden ? '🙈' : '👁'}
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
        {entity.type !== 'lair_action' && (
          <div style={{ fontSize: 13, color: textColor, flexShrink: 0, textAlign: 'right' }}>
            {entity.current_hp}/{entity.max_hp} HP
            {entity.temp_hp > 0 && <span style={{ color: '#3498db' }}> +{entity.temp_hp}</span>}
          </div>
        )}
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
            {entity.type !== 'lair_action' && (
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
            )}

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

            {/* Creature name (creatures and lair actions) */}
            {(entity.type === 'creature' || entity.type === 'lair_action') && (
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

            {/* Creature alias (creatures and lair actions) */}
            {(entity.type === 'creature' || entity.type === 'lair_action') && (
              <div>
                <div style={{ fontSize: 11, color: '#7878a0', marginBottom: 4 }}>Alias</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <input
                    type="text"
                    value={aliasInput}
                    onChange={e => setAliasInput(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') applyDisplayName() }}
                    placeholder="none"
                    style={{ width: 120, padding: '6px 8px', fontSize: 14 }}
                  />
                  <button onClick={applyDisplayName} style={actionBtn}>Set</button>
                </div>
              </div>
            )}
          </div>

          {/* Condition toggles */}
          {entity.type !== 'lair_action' && (
            <div style={{ marginBottom: 12 }}>
              <ConditionToggles conditions={entity.conditions} onToggle={toggleCondition} />
            </div>
          )}

          {/* Action buttons */}
          <div style={{ display: 'flex', gap: 8 }}>
            {entity.type !== 'lair_action' && (
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
            )}
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
