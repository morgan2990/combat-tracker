import { useState, useEffect, useRef } from 'react'
import type { RoomState } from '../types'
import { StatblockDrawer } from './StatblockDrawer'
import { StatblockColumn } from './StatblockColumn'
import { DMNavColumn } from './DMNavColumn'
import { InventoryPanel } from './InventoryPanel'
import { useLayoutTier } from '../hooks/useLayoutTier'
import { EntityRow } from './EntityRow'
import { AddCreatureForm } from './AddCreatureForm'
import type { AddCreatureFormHandle } from './AddCreatureForm'
import { EncounterTemplatesControl } from './EncounterTemplatesControl'

interface DMViewProps {
  roomState: RoomState
  sendMessage: (msg: object) => void
  onBackToDashboard: () => void
}

// Column sizing shared by the tablet (Nav | Tracker) and desktop
// (Nav | Tracker | Statblock) layouts. Tracker's max-width is reduced from
// the phone tier's 720px so the desktop tier is reachable on common
// ~1366px-wide laptops, not just large monitors.
const NAV_COLUMN_WIDTH = 220
const TRACKER_MAX_WIDTH = 580
const TRACKER_MIN_BASIS = 460
const STATBLOCK_COLUMN_WIDTH = 420
const COLUMN_GAP = 16
const TABLET_LAYOUT_CAP = NAV_COLUMN_WIDTH + TRACKER_MAX_WIDTH + COLUMN_GAP + 32
const DESKTOP_LAYOUT_CAP = NAV_COLUMN_WIDTH + TRACKER_MAX_WIDTH + STATBLOCK_COLUMN_WIDTH + COLUMN_GAP * 2 + 32

export function DMView({ roomState, sendMessage, onBackToDashboard }: DMViewProps) {
  const { entities, active_index, is_started, round } = roomState
  const hasDeadCreatures = entities.some(e => e.dead && e.type === 'creature')
  const pendingInitiative = entities.filter(e => (e.type === 'pc' || e.type === 'companion') && e.initiative === null)
  const [confirmingEnd, setConfirmingEnd] = useState(false)
  const [openDrawerEntityId, setOpenDrawerEntityId] = useState<string | null>(null)
  const [inventoryPcId, setInventoryPcId] = useState<string | null>(null)
  const addCreatureFormRef = useRef<AddCreatureFormHandle>(null)
  const tier = useLayoutTier()

  useEffect(() => {
    if (!is_started) setConfirmingEnd(false)
  }, [is_started])

  const openEntity = entities.find(e => e.id === openDrawerEntityId) ?? null

  const trackerColumnBody = (
    <>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <button
          onClick={onBackToDashboard}
          style={{ padding: '6px 12px', fontSize: 13, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
        >
          ← Dashboard
        </button>
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
        <button
          onClick={() => sendMessage({ type: 'add_lair_action' })}
          style={{ padding: '10px 20px', fontSize: 14, cursor: 'pointer', background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4 }}
        >
          + Add Lair Action
        </button>
        {tier === 'phone' && <EncounterTemplatesControl sendMessage={sendMessage} edition={roomState.edition} />}
      </div>

      {/* Add creature form */}
      <div style={{ border: '1px solid #2e2e48', borderRadius: 6, padding: '0 14px', marginBottom: 12, background: '#161626' }}>
        <AddCreatureForm
          ref={addCreatureFormRef}
          sendMessage={sendMessage}
          edition={roomState.edition}
          showMyCreaturesInline={tier === 'phone'}
        />
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
            onOpenInventory={setInventoryPcId}
          />
        ))}
      </div>
    </>
  )

  const statblockDrawers = entities.filter(e => e.type === 'creature' && e.source_type).map(e => (
    <StatblockDrawer
      key={e.id}
      entity={e}
      open={openDrawerEntityId === e.id}
      onClose={() => setOpenDrawerEntityId(null)}
    />
  ))

  const inventoryPanel = inventoryPcId && (
    <InventoryPanel pcId={inventoryPcId} onClose={() => setInventoryPcId(null)} />
  )

  if (tier === 'phone') {
    return (
      <div style={{ maxWidth: 720, margin: '0 auto', padding: 16 }}>
        {trackerColumnBody}
        {statblockDrawers}
        {inventoryPanel}
      </div>
    )
  }

  return (
    <div
      style={{
        maxWidth: tier === 'desktop' ? DESKTOP_LAYOUT_CAP : TABLET_LAYOUT_CAP,
        margin: '0 auto', padding: 16, boxSizing: 'border-box',
        display: 'flex', gap: COLUMN_GAP, height: '100vh',
      }}
    >
      <div style={{ flex: `0 0 ${NAV_COLUMN_WIDTH}px`, minHeight: 0 }}>
        <DMNavColumn
          sendMessage={sendMessage}
          edition={roomState.edition}
          onSelectCustomMonster={m => addCreatureFormRef.current?.selectCustomMonster(m)}
        />
      </div>

      <div style={{ flex: `1 1 ${TRACKER_MIN_BASIS}px`, maxWidth: TRACKER_MAX_WIDTH, minHeight: 0, overflowY: 'auto' }}>
        {trackerColumnBody}
      </div>

      {tier === 'desktop' && (
        <div style={{ flex: `0 0 ${STATBLOCK_COLUMN_WIDTH}px`, minHeight: 0 }}>
          <StatblockColumn entity={openEntity} onClose={() => setOpenDrawerEntityId(null)} />
        </div>
      )}

      {tier === 'tablet' && statblockDrawers}
      {inventoryPanel}
    </div>
  )
}
