import { useState } from 'react'
import type { FormEvent } from 'react'
import type { Entity, RoomState } from '../types'
import { CompanionForm } from './CompanionForm'
import { InventoryPanel } from './InventoryPanel'
import { ConditionToggles } from './ConditionToggles'
import { entityVitalState, vitalRowBg, vitalTextColor } from '../entityVitals'

function hpLabel(current: number, max: number): string {
  if (max === 0) return 'Unknown'
  const ratio = current / max
  if (ratio > 0.75) return 'Healthy'
  if (ratio > 0.5) return 'Hurt'
  if (ratio > 0.25) return 'Injured'
  if (ratio > 0) return 'Dying'
  return 'Dead'
}

interface EntityEditorProps {
  entity: Entity
  sendMessage: (msg: object) => void
}

function EntityEditor({ entity, sendMessage }: EntityEditorProps) {
  const [editingHP, setEditingHP] = useState(false)
  const [hpInput, setHpInput] = useState(String(entity.current_hp))

  function applyHP(newHP: number) {
    const clamped = Math.max(0, Math.min(newHP, entity.max_hp))
    sendMessage({
      type: 'update_entity',
      entity_id: entity.id,
      current_hp: clamped,
      temp_hp: entity.temp_hp,
      conditions: entity.conditions,
    })
  }

  function handleDelta(delta: number) {
    applyHP(entity.current_hp + delta)
  }

  function handleDirectSet() {
    const val = parseInt(hpInput, 10)
    if (!isNaN(val)) applyHP(val)
    setEditingHP(false)
  }

  function toggleCondition(cond: string) {
    const next = entity.conditions.includes(cond)
      ? entity.conditions.filter(c => c !== cond)
      : [...entity.conditions, cond]
    sendMessage({
      type: 'update_entity',
      entity_id: entity.id,
      current_hp: entity.current_hp,
      temp_hp: entity.temp_hp,
      conditions: next,
    })
  }

  return (
    <div style={{ padding: '12px 0' }}>
      {/* HP editor */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
        {([-10, -5, -1] as const).map(d => (
          <button key={d} onClick={() => handleDelta(d)} style={deltaBtn}>
            {d}
          </button>
        ))}

        {editingHP ? (
          <input
            type="number"
            value={hpInput}
            autoFocus
            onChange={e => setHpInput(e.target.value)}
            onBlur={handleDirectSet}
            onKeyDown={e => { if (e.key === 'Enter') handleDirectSet() }}
            style={{ width: 64, fontSize: 18, textAlign: 'center', padding: '4px 8px' }}
          />
        ) : (
          <button
            onClick={() => { setHpInput(String(entity.current_hp)); setEditingHP(true) }}
            style={{ fontSize: 18, fontWeight: 'bold', background: 'none', border: '1px solid #2e2e48', borderRadius: 4, padding: '4px 12px', cursor: 'pointer', minWidth: 72, textAlign: 'center', color: '#d4d4e8' }}
          >
            {entity.current_hp}/{entity.max_hp}
          </button>
        )}

        {([1, 5, 10] as const).map(d => (
          <button key={d} onClick={() => handleDelta(d)} style={deltaBtn}>
            +{d}
          </button>
        ))}

        {entity.temp_hp > 0 && (
          <span style={{ fontSize: 13, color: '#3498db', marginLeft: 4 }}>+{entity.temp_hp} tmp</span>
        )}
      </div>

      {/* Condition toggles */}
      <ConditionToggles conditions={entity.conditions} onToggle={toggleCondition} />
    </div>
  )
}

interface InitiativeFormProps {
  sendMessage: (msg: object) => void
}

function InitiativeForm({ sendMessage }: InitiativeFormProps) {
  const [initiative, setInitiative] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const init = parseInt(initiative, 10)
    if (isNaN(init)) return
    sendMessage({ type: 'set_initiative', initiative: init })
  }

  return (
    <div style={{ padding: '12px 16px', background: '#1a1608', border: '1px solid #e67e22', borderRadius: 6, marginBottom: 16 }}>
      <div style={{ fontWeight: 600, color: '#e67e22', marginBottom: 8, fontSize: 14 }}>Set your initiative</div>
      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8 }}>
        <input
          type="number"
          value={initiative}
          onChange={e => setInitiative(e.target.value)}
          placeholder="e.g. 14"
          required
          autoFocus
          style={{ flex: 1, padding: '8px 12px', fontSize: 16, boxSizing: 'border-box' }}
        />
        <button
          type="submit"
          disabled={!initiative}
          style={{
            padding: '8px 16px', fontSize: 14, fontWeight: 600,
            background: '#e67e22', color: '#fff', border: 'none', borderRadius: 4,
            cursor: !initiative ? 'not-allowed' : 'pointer',
            opacity: !initiative ? 0.45 : 1,
          }}
        >
          Set
        </button>
      </form>
    </div>
  )
}

interface PlayerViewProps {
  roomState: RoomState
  myEntityId: string | null
  needsInitiative: boolean
  sendMessage: (msg: object) => void
  onBackToDashboard: () => void
}

export function PlayerView({ roomState, myEntityId, needsInitiative, sendMessage, onBackToDashboard }: PlayerViewProps) {
  const [showCompanionForm, setShowCompanionForm] = useState(false)
  const [showInventory, setShowInventory] = useState(false)
  const { entities, active_index, is_started, round } = roomState

  const myEntity = entities.find(e => e.id === myEntityId)
  const myCompanions = entities.filter(e => e.type === 'companion' && e.owner_id === myEntityId)
  const visibleEntities = entities.filter(e => (is_started || e.type !== 'creature') && !e.is_hidden)

  return (
    <div style={{ maxWidth: 480, margin: '0 auto', padding: 16 }}>

      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <button
          onClick={onBackToDashboard}
          style={{ padding: '6px 12px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
        >
          ← Dashboard
        </button>
        <h2 style={{ margin: 0 }}>⚔ Combat Tracker</h2>
        {is_started && <span style={{ fontWeight: 'bold', color: '#e67e22' }}>Round {round}</span>}
      </div>

      {/* Initiative prompt (shown until initiative is set) */}
      {needsInitiative && <InitiativeForm sendMessage={sendMessage} />}

      {!is_started && !needsInitiative && (
        <div style={{ padding: 10, background: '#161626', border: '1px solid #2e2e48', borderRadius: 6, marginBottom: 12, fontSize: 14, color: '#7878a0' }}>
          Waiting for DM to start combat…
        </div>
      )}

      {/* Initiative tracker */}
      <div style={{ border: '1px solid #2e2e48', borderRadius: 6, overflow: 'hidden', marginBottom: 16 }}>
        {visibleEntities.length === 0 && entities.length === 0 && (
          <div style={{ padding: 16, color: '#7878a0' }}>No combatants yet.</div>
        )}
        {visibleEntities.length === 0 && entities.length > 0 && (
          <div style={{ padding: 16, color: '#7878a0' }}>The DM is preparing the encounter…</div>
        )}
        {visibleEntities.map((entity, i) => {
          const isActive = is_started && i === active_index
          const isMe = entity.id === myEntityId
          const isCreature = entity.type === 'creature'
          const isLairAction = entity.type === 'lair_action'
          const vitalState = (isCreature || isLairAction) ? 'alive' : entityVitalState(entity.dead, entity.current_hp)

          const rowBg = vitalRowBg(vitalState, isActive, isMe)
          const textColor = vitalTextColor(vitalState)

          return (
            <div
              key={entity.id}
              style={{
                padding: '10px 14px',
                borderBottom: '1px solid #2e2e48',
                background: rowBg,
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              <span style={{ width: 16, color: '#e67e22' }}>{isActive ? '▶' : ''}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: isMe ? 700 : 400, color: textColor, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {entity.display_name || entity.name}
                  {isMe && <span style={{ fontSize: 11, color: '#3498db', marginLeft: 6 }}>you</span>}
                  {vitalState === 'dead' && <span style={{ fontSize: 11, color: '#e74c3c', marginLeft: 6 }}>💀 Dead</span>}
                  {vitalState === 'unconscious' && <span style={{ fontSize: 11, color: '#e67e22', marginLeft: 6 }}>😵 Unconscious</span>}
                </div>
                {entity.conditions.length > 0 && (
                  <div style={{ fontSize: 11, color: '#e74c3c', marginTop: 2 }}>
                    {entity.conditions.join(' · ')}
                  </div>
                )}
              </div>
              <div style={{ fontSize: 14, textAlign: 'right', flexShrink: 0, color: textColor }}>
                {isLairAction ? null : isCreature ? (
                  <span style={{ color: '#7878a0', fontStyle: 'italic' }}>{hpLabel(entity.current_hp, entity.max_hp)}</span>
                ) : (
                  <span>{entity.current_hp}/{entity.max_hp} HP</span>
                )}
              </div>
              <div style={{ fontSize: 12, color: '#5a5a78', width: 32, textAlign: 'right' }}>
                {entity.initiative !== null ? entity.initiative : '—'}
              </div>
            </div>
          )
        })}
      </div>

      {/* My entity editor */}
      {myEntity && (
        <div style={{ border: '1px solid #3498db', borderRadius: 6, padding: '12px 16px', marginBottom: 16, background: '#0c1929' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 8 }}>
            <div style={{ fontWeight: 700, color: '#d4d4e8' }}>
              {myEntity.name}
              {is_started && active_index !== -1 && entities[active_index]?.id === myEntityId && (
                <span style={{ marginLeft: 8, color: '#e67e22', fontSize: 13 }}>Your turn!</span>
              )}
              {entityVitalState(myEntity.dead, myEntity.current_hp) === 'dead' && (
                <span style={{ marginLeft: 8, fontSize: 12, color: '#e74c3c' }}>💀 Dead</span>
              )}
              {entityVitalState(myEntity.dead, myEntity.current_hp) === 'unconscious' && (
                <span style={{ marginLeft: 8, fontSize: 12, color: '#e67e22' }}>😵 Unconscious</span>
              )}
            </div>
            <div style={{ display: 'flex', gap: 6 }}>
              {myEntity.pc_id && (
                <button
                  onClick={() => setShowInventory(true)}
                  title="View inventory"
                  style={{ padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: '#1e1e30', color: '#7878a0', border: '1px solid #2e2e48', borderRadius: 4 }}
                >
                  🎒 Inventory
                </button>
              )}
              <button
                onClick={() => sendMessage({ type: 'refresh_from_profile' })}
                title="Refresh max HP from saved profile"
                style={{ padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: '#1e1e30', color: '#7878a0', border: '1px solid #2e2e48', borderRadius: 4 }}
              >
                ↻ Refresh profile
              </button>
            </div>
          </div>
          <EntityEditor entity={myEntity} sendMessage={sendMessage} />
          <button
            onClick={() => setShowCompanionForm(true)}
            style={{ marginTop: 12, padding: '8px 14px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
          >
            + Add Summon / Pet
          </button>
        </div>
      )}

      {/* Owned companion editors */}
      {myCompanions.map(companion => (
        <div key={companion.id} style={{ border: '1px solid #9b59b6', borderRadius: 6, padding: '12px 16px', marginBottom: 12, background: '#130f1d' }}>
          <div style={{ fontWeight: 700, marginBottom: 8, color: '#b07fe0' }}>
            {companion.name} <span style={{ fontSize: 11, fontWeight: 400, color: '#5a5a78' }}>companion</span>
            {entityVitalState(companion.dead, companion.current_hp) === 'dead' && (
              <span style={{ marginLeft: 8, fontSize: 12, color: '#e74c3c' }}>💀 Dead</span>
            )}
            {entityVitalState(companion.dead, companion.current_hp) === 'unconscious' && (
              <span style={{ marginLeft: 8, fontSize: 12, color: '#e67e22' }}>😵 Unconscious</span>
            )}
          </div>
          <EntityEditor entity={companion} sendMessage={sendMessage} />
        </div>
      ))}

      {showCompanionForm && (
        <CompanionForm
          onSubmit={sendMessage}
          onClose={() => setShowCompanionForm(false)}
        />
      )}

      {showInventory && myEntity?.pc_id && (
        <InventoryPanel pcId={myEntity.pc_id} onClose={() => setShowInventory(false)} />
      )}
    </div>
  )
}

const deltaBtn: React.CSSProperties = {
  padding: '6px 10px',
  fontSize: 14,
  cursor: 'pointer',
  borderRadius: 4,
  border: '1px solid #2e2e48',
  background: '#1e1e30',
  color: '#d4d4e8',
  minWidth: 38,
}
