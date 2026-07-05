import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import type { MeResponse, CustomMonster, Encounter, Party, PC, Currency } from '../types'
import { InventoryPanel } from './InventoryPanel'
import { CurrencyEditor } from './CurrencyEditor'
import { CollapsibleSection } from './CollapsibleSection'
import { fetchJSON } from '../fetchJSON'
import { deleteWithConfirm } from '../deleteWithConfirm'
import { color, radius } from '../theme'
import { useLayoutTier } from '../hooks/useLayoutTier'

interface DashboardProps {
  me: MeResponse
  onOpenRoomAsDM: (roomId: string) => void
  onJoinAsPlayer: (roomId: string, pcId: string) => void
  onLogout: () => void
}

export function Dashboard({ me, onOpenRoomAsDM, onJoinAsPlayer, onLogout }: DashboardProps) {
  const tier = useLayoutTier()
  const [activePanel, setActivePanel] = useState<'dm' | 'player'>('dm')

  const [dmEdition, setDmEdition] = useState<'5e' | '5.5e'>('5e')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const [joinRoomCode, setJoinRoomCode] = useState('')
  const [selectedPcId, setSelectedPcId] = useState(me.pcs[0]?.id ?? '')

  // Rooms/recent-rooms are derived from `me` rather than mirrored into their
  // own state — a deleted/left id is hidden locally without waiting on a
  // fresh `me` fetch, and there's no state to go stale if Dashboard remounts
  // with an out-of-date `me` prop (e.g. navigating back from a room races the
  // in-flight refreshMe() fetch).
  const [hiddenOwnedRoomIds, setHiddenOwnedRoomIds] = useState<Set<string>>(new Set())
  const [hiddenMembershipRoomIds, setHiddenMembershipRoomIds] = useState<Set<string>>(new Set())
  const [hiddenPcIds, setHiddenPcIds] = useState<Set<string>>(new Set())
  const rooms = me.rooms.filter(r => !hiddenOwnedRoomIds.has(r.room_id))
  const recentRooms = me.recent_rooms.filter(rm => !hiddenMembershipRoomIds.has(rm.room_id))
  const pcs = me.pcs.filter(pc => !hiddenPcIds.has(pc.id))

  const [myMonsters, setMyMonsters] = useState<CustomMonster[]>([])
  const [myEncounters, setMyEncounters] = useState<Encounter[]>([])

  const [parties, setParties] = useState<Party[]>(me.parties ?? [])
  const [newPartyName, setNewPartyName] = useState('')
  const [creatingParty, setCreatingParty] = useState(false)
  const [inventoryPcId, setInventoryPcId] = useState<string | null>(null)
  const [joinPartyCode, setJoinPartyCode] = useState('')
  const [joinPartyPcId, setJoinPartyPcId] = useState(me.pcs[0]?.id ?? '')
  const [joiningParty, setJoiningParty] = useState(false)
  const [joinPartyError, setJoinPartyError] = useState<string | null>(null)

  useEffect(() => {
    fetchJSON<CustomMonster[]>('/api/custom-monsters', []).then(setMyMonsters)
  }, [])

  useEffect(() => {
    fetchJSON<Encounter[]>('/api/encounters', []).then(setMyEncounters)
  }, [])

  function handleDeleteMonster(id: string) {
    return deleteWithConfirm(
      'Delete this monster? This cannot be undone.',
      `/api/custom-monsters/${encodeURIComponent(id)}`,
      () => setMyMonsters(prev => prev.filter(m => m.id !== id)),
    )
  }

  function handleDeleteEncounter(id: string) {
    return deleteWithConfirm(
      'Delete this encounter? This cannot be undone.',
      `/api/encounters/${encodeURIComponent(id)}`,
      () => setMyEncounters(prev => prev.filter(enc => enc.id !== id)),
    )
  }

  function handleDeleteRoom(roomId: string) {
    return deleteWithConfirm(
      'Delete this room? This cannot be undone.',
      `/api/rooms/${encodeURIComponent(roomId)}`,
      () => setHiddenOwnedRoomIds(prev => new Set(prev).add(roomId)),
    )
  }

  function handleLeaveRoom(roomId: string) {
    return deleteWithConfirm(
      'Leave this room?',
      `/api/rooms/${encodeURIComponent(roomId)}/membership`,
      () => setHiddenMembershipRoomIds(prev => new Set(prev).add(roomId)),
    )
  }

  function handleDeleteCharacter(pcId: string) {
    return deleteWithConfirm(
      "Delete this character? This cannot be undone. If it's currently active in a room, you won't be able to reconnect to that room afterward.",
      `/api/pcs/${encodeURIComponent(pcId)}`,
      () => setHiddenPcIds(prev => new Set(prev).add(pcId)),
    )
  }

  async function handleCreateRoom() {
    setCreating(true)
    setCreateError(null)
    try {
      const res = await fetch('/api/rooms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ edition: dmEdition }),
      })
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      const data = await res.json()
      onOpenRoomAsDM(data.room_id)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create room')
      setCreating(false)
    }
  }

  function handleJoinNewRoom() {
    if (!joinRoomCode.trim() || !selectedPcId) return
    onJoinAsPlayer(joinRoomCode.trim().toUpperCase(), selectedPcId)
  }

  async function handleCreateParty() {
    if (!newPartyName.trim()) return
    setCreatingParty(true)
    try {
      const res = await fetch('/api/parties', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newPartyName.trim() }),
      })
      if (res.ok) {
        const created: Party = await res.json()
        setParties(prev => [...prev, created])
        setNewPartyName('')
      }
    } finally {
      setCreatingParty(false)
    }
  }

  async function handleUpdateParty(partyId: string, memberPCIDs: string[], currency: Currency) {
    const res = await fetch(`/api/parties/${encodeURIComponent(partyId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ member_pc_ids: memberPCIDs, currency }),
    })
    if (res.ok) {
      const updated: Party = await res.json()
      setParties(prev => prev.map(p => p.id === updated.id ? updated : p))
    }
  }

  async function handleJoinParty() {
    if (!joinPartyCode.trim() || !joinPartyPcId) return
    setJoiningParty(true)
    setJoinPartyError(null)
    try {
      const res = await fetch('/api/parties/join', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ join_code: joinPartyCode.trim().toUpperCase(), pc_id: joinPartyPcId }),
      })
      if (!res.ok) throw new Error(res.status === 404 ? 'No party found with that code' : 'Failed to join party')
      const joined: Party = await res.json()
      setParties(prev => prev.some(p => p.id === joined.id) ? prev.map(p => p.id === joined.id ? joined : p) : [...prev, joined])
      setJoinPartyCode('')
    } catch (err) {
      setJoinPartyError(err instanceof Error ? err.message : 'Failed to join party')
    } finally {
      setJoiningParty(false)
    }
  }

  const dmPanel = (
    <div style={{ flex: 1, minWidth: 300 }}>
      <h2 style={sectionTitle}>As DM</h2>

      <div style={panelStyle}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 20 }}>
          <span style={{ fontSize: 13, color: color.textSecondary }}>Edition</span>
          {(['5e', '5.5e'] as const).map(ed => (
            <button
              key={ed}
              type="button"
              onClick={() => setDmEdition(ed)}
              style={segmentButtonStyle(dmEdition === ed)}
            >
              {ed}
            </button>
          ))}
        </div>

        <CollapsibleSection
          id="dm.rooms"
          title="My Rooms"
          count={rooms.length}
          headerAction={
            <button onClick={handleCreateRoom} disabled={creating} style={primaryBtn(creating)}>
              {creating ? 'Creating…' : '+ New Room'}
            </button>
          }
        >
          {rooms.length === 0 && (
            <div style={{ fontSize: 13, color: color.textMuted, marginBottom: 12 }}>No rooms yet.</div>
          )}
          {rooms.map(room => (
            <div key={room.room_id} style={listRow}>
              <span>
                {room.room_id}
                {room.is_combat_active && <span style={{ marginLeft: 8, fontSize: 11, color: color.accent }}>● in combat</span>}
              </span>
              <span style={{ display: 'flex', gap: 8 }}>
                <button onClick={() => onOpenRoomAsDM(room.room_id)} style={actionBtn}>Open</button>
                <button onClick={() => handleDeleteRoom(room.room_id)} style={deleteBtn}>Delete</button>
              </span>
            </div>
          ))}
          {createError && <div style={errorStyle}>{createError}</div>}
        </CollapsibleSection>

        <CollapsibleSection
          id="dm.monsters"
          title="My Monsters"
          count={myMonsters.length}
          headerAction={
            <Link to="/monsters/new" style={headerActionButton}>+ New Monster</Link>
          }
        >
          {myMonsters.length === 0 && (
            <div style={{ fontSize: 13, color: color.textMuted, marginBottom: 12 }}>No custom monsters yet.</div>
          )}
          {myMonsters.map(m => (
            <div key={m.id} style={listRow}>
              <span>
                {m.name}
                {m.private && <span style={{ marginLeft: 8, fontSize: 11, color: color.textSecondary }}>● private</span>}
              </span>
              <span style={{ display: 'flex', gap: 8 }}>
                <Link to={`/monsters/custom/${m.id}/edit`} style={{ fontSize: 12, color: color.info }}>Edit</Link>
                <button onClick={() => handleDeleteMonster(m.id)} style={deleteBtn}>Delete</button>
              </span>
            </div>
          ))}
        </CollapsibleSection>

        <CollapsibleSection
          id="dm.encounters"
          title="My Encounters"
          count={myEncounters.length}
          headerAction={
            <Link to="/encounters/new" style={headerActionButton}>+ New Encounter</Link>
          }
        >
          {myEncounters.length === 0 && (
            <div style={{ fontSize: 13, color: color.textMuted, marginBottom: 12 }}>No encounters yet.</div>
          )}
          {myEncounters.map(enc => (
            <div key={enc.id} style={listRow}>
              <span>{enc.name}</span>
              <span style={{ display: 'flex', gap: 8 }}>
                <Link to={`/encounters/${enc.id}/edit`} style={{ fontSize: 12, color: color.info }}>Edit</Link>
                <button onClick={() => handleDeleteEncounter(enc.id)} style={deleteBtn}>Delete</button>
              </span>
            </div>
          ))}
        </CollapsibleSection>
      </div>
    </div>
  )

  const playerPanel = (
    <div style={{ flex: 1, minWidth: 300 }}>
      <h2 style={sectionTitle}>As Player</h2>

      <div style={panelStyle}>
        <CollapsibleSection
          id="player.characters"
          title="My Characters"
          count={pcs.length}
          headerAction={
            <Link to="/characters/new" style={headerActionButton}>+ New Character</Link>
          }
        >
          {pcs.length === 0 && (
            <div style={{ fontSize: 13, color: color.textMuted, marginBottom: 12 }}>No characters yet.</div>
          )}
          {pcs.map(pc => (
            <div key={pc.id} style={listRow}>
              <span>{pc.name} <span style={{ color: color.textMuted, fontSize: 11 }}>{pc.max_hp} HP</span></span>
              <span style={{ display: 'flex', gap: 8 }}>
                <button
                  onClick={() => setInventoryPcId(pc.id)}
                  style={{ fontSize: 12, cursor: 'pointer', padding: 0, background: 'none', color: color.info, border: 'none' }}
                >
                  Inventory
                </button>
                <Link to={`/characters/${pc.id}/edit`} style={{ fontSize: 12, color: color.info }}>Edit</Link>
                <button onClick={() => handleDeleteCharacter(pc.id)} style={deleteBtn}>Delete</button>
              </span>
            </div>
          ))}
        </CollapsibleSection>

        <CollapsibleSection id="player.parties" title="My Parties" count={parties.length}>
          {parties.length === 0 && (
            <div style={{ fontSize: 13, color: color.textMuted, marginBottom: 12 }}>No parties yet.</div>
          )}
          {parties.map(party => (
            <PartyCard
              key={party.id}
              party={party}
              myPcs={me.pcs}
              onUpdate={(memberPCIDs, currency) => handleUpdateParty(party.id, memberPCIDs, currency)}
            />
          ))}
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <input
              value={newPartyName}
              onChange={e => setNewPartyName(e.target.value)}
              placeholder="Party name"
              style={{ ...inputStyle, flex: 1 }}
            />
            <button
              onClick={handleCreateParty}
              disabled={creatingParty || !newPartyName.trim()}
              style={primaryBtn(creatingParty || !newPartyName.trim())}
            >
              {creatingParty ? 'Creating…' : '+ New Party'}
            </button>
          </div>

          {me.pcs.length > 0 && (
            <div style={{ marginTop: 8 }}>
              <div style={{ display: 'flex', gap: 8 }}>
                <select value={joinPartyPcId} onChange={e => setJoinPartyPcId(e.target.value)} style={selectStyle}>
                  {me.pcs.map(pc => (
                    <option key={pc.id} value={pc.id}>{pc.name}</option>
                  ))}
                </select>
                <input
                  value={joinPartyCode}
                  onChange={e => setJoinPartyCode(e.target.value.toUpperCase())}
                  maxLength={5}
                  placeholder="Party code"
                  style={{ ...inputStyle, flex: 1 }}
                />
                <button onClick={handleJoinParty} disabled={joiningParty || !joinPartyCode.trim()} style={actionBtn}>
                  {joiningParty ? 'Joining…' : 'Join'}
                </button>
              </div>
              {joinPartyError && <div style={errorStyle}>{joinPartyError}</div>}
            </div>
          )}
        </CollapsibleSection>

        <CollapsibleSection id="player.rooms" title="Rooms" count={recentRooms.length}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
            {recentRooms.length > 0 && (
              <div>
                <div style={{ fontSize: 13, color: color.textSecondary, marginBottom: 8 }}>Recent Rooms</div>
                {recentRooms.map(rm => {
                  const pc = me.pcs.find(p => p.id === rm.last_pc_id)
                  return (
                    <div key={rm.room_id} style={listRow}>
                      <span>{rm.room_id} <span style={{ color: color.textMuted, fontSize: 11 }}>as {pc?.name ?? 'unknown'}</span></span>
                      <span style={{ display: 'flex', gap: 8 }}>
                        <button
                          onClick={() => onJoinAsPlayer(rm.room_id, rm.last_pc_id)}
                          disabled={!pc}
                          style={actionBtn}
                        >
                          Go
                        </button>
                        <button onClick={() => handleLeaveRoom(rm.room_id)} style={deleteBtn}>Leave</button>
                      </span>
                    </div>
                  )
                })}
              </div>
            )}

            {me.pcs.length > 0 && (
              <div>
                <div style={{ fontSize: 13, color: color.textSecondary, marginBottom: 8 }}>Join a new room</div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <select value={selectedPcId} onChange={e => setSelectedPcId(e.target.value)} style={selectStyle}>
                    {me.pcs.map(pc => (
                      <option key={pc.id} value={pc.id}>{pc.name}</option>
                    ))}
                  </select>
                  <input
                    value={joinRoomCode}
                    onChange={e => setJoinRoomCode(e.target.value.toUpperCase())}
                    maxLength={6}
                    placeholder="XXXXX"
                    style={{ ...inputStyle, flex: 1 }}
                  />
                  <button onClick={handleJoinNewRoom} disabled={!joinRoomCode.trim()} style={primaryBtn(!joinRoomCode.trim())}>
                    Go
                  </button>
                </div>
              </div>
            )}
          </div>
        </CollapsibleSection>
      </div>
    </div>
  )

  return (
    <div style={{ maxWidth: 760, margin: '0 auto', padding: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h1 style={{ margin: 0, fontSize: 24 }}>⚔ Combat Tracker</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 13, color: color.textSecondary }}>{me.user.display_name}</span>
          <button onClick={onLogout} style={logoutBtn}>Log Out</button>
        </div>
      </div>

      {tier === 'phone' ? (
        <div>
          <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
            {(['dm', 'player'] as const).map(panel => (
              <button
                key={panel}
                type="button"
                onClick={() => setActivePanel(panel)}
                style={{ flex: 1, ...segmentButtonStyle(activePanel === panel, { padding: '8px 12px', fontSize: 14 }) }}
              >
                {panel === 'dm' ? 'As DM' : 'As Player'}
              </button>
            ))}
          </div>
          {activePanel === 'dm' ? dmPanel : playerPanel}
        </div>
      ) : (
        <div style={{ display: 'flex', gap: 20, flexWrap: 'wrap', alignItems: 'flex-start' }}>
          {dmPanel}
          {playerPanel}
        </div>
      )}

      {inventoryPcId && (
        <InventoryPanel pcId={inventoryPcId} onClose={() => setInventoryPcId(null)} />
      )}
    </div>
  )
}

interface PartyCardProps {
  party: Party
  myPcs: PC[]
  onUpdate: (memberPCIDs: string[], currency: Currency) => void
}

function PartyCard({ party, myPcs, onUpdate }: PartyCardProps) {
  const [currency, setCurrency] = useState<Currency>(party.currency)

  useEffect(() => { setCurrency(party.currency) }, [party.currency])

  const [addPcId, setAddPcId] = useState('')
  const availablePcs = myPcs.filter(pc => !party.member_pc_ids.includes(pc.id))

  function memberName(pcId: string): string {
    return party.member_names[pcId] ?? pcId
  }

  function addMember() {
    if (!addPcId) return
    onUpdate([...party.member_pc_ids, addPcId], party.currency)
    setAddPcId('')
  }

  function removeMember(pcId: string) {
    onUpdate(party.member_pc_ids.filter(id => id !== pcId), party.currency)
  }

  function saveCurrency() {
    onUpdate(party.member_pc_ids, currency)
  }

  return (
    <div style={{ ...panelStyle, marginBottom: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 8 }}>
        <div style={{ fontWeight: 600 }}>{party.name}</div>
        <div style={{ fontSize: 11, color: color.textMuted }}>Code: {party.join_code}</div>
      </div>

      {party.member_pc_ids.length === 0 && (
        <div style={{ fontSize: 12, color: color.textMuted, marginBottom: 8 }}>No members yet.</div>
      )}
      {party.member_pc_ids.map(pcId => (
        <div key={pcId} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 13, padding: '4px 0' }}>
          <span>{memberName(pcId)}</span>
          <button onClick={() => removeMember(pcId)} style={deleteBtn}>Remove</button>
        </div>
      ))}

      {availablePcs.length > 0 && (
        <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
          <select value={addPcId} onChange={e => setAddPcId(e.target.value)} style={selectStyle}>
            <option value="">Add character…</option>
            {availablePcs.map(pc => (
              <option key={pc.id} value={pc.id}>{pc.name}</option>
            ))}
          </select>
          <button onClick={addMember} disabled={!addPcId} style={actionBtn}>Add</button>
        </div>
      )}

      <div style={{ marginTop: 12 }}>
        <div style={{ fontSize: 12, color: color.textSecondary, marginBottom: 4 }}>Pooled Currency</div>
        <div style={{ display: 'flex', gap: 6, alignItems: 'flex-end', flexWrap: 'wrap' }}>
          <CurrencyEditor currency={currency} onChange={setCurrency} inputWidth={50} />
          <button onClick={saveCurrency} style={actionBtn}>Save</button>
        </div>
      </div>
    </div>
  )
}

const sectionTitle: React.CSSProperties = { fontSize: 16, marginBottom: 10, color: color.textPrimary }

const panelStyle: React.CSSProperties = {
  padding: '14px 16px', background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: radius.md,
}

const listRow: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
  padding: '8px 0', borderBottom: `1px solid ${color.border}`, fontSize: 14,
}

const actionBtn: React.CSSProperties = {
  padding: '5px 12px', fontSize: 12, cursor: 'pointer',
  background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm,
}

const deleteBtn: React.CSSProperties = {
  fontSize: 12, cursor: 'pointer', padding: 0,
  background: 'none', color: color.danger, border: 'none',
}

const logoutBtn: React.CSSProperties = {
  padding: '6px 12px', fontSize: 12, cursor: 'pointer',
  background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm,
}

const inputStyle: React.CSSProperties = {
  padding: '8px 10px', fontSize: 14, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const selectStyle: React.CSSProperties = {
  ...inputStyle, maxWidth: 140, minWidth: 0,
}

const errorStyle: React.CSSProperties = {
  marginTop: 8, fontSize: 13, color: color.danger,
}

function primaryBtn(disabled: boolean): React.CSSProperties {
  return {
    padding: '8px 16px', fontSize: 13, fontWeight: 700,
    background: color.success, color: '#fff', border: 'none', borderRadius: radius.sm,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}

// A bare navigation link styled to match `primaryBtn`'s enabled look, for header
// actions that route to another page rather than submit/create in place.
const headerActionButton: React.CSSProperties = { ...primaryBtn(false), textDecoration: 'none' }

function segmentButtonStyle(active: boolean, sizing: { padding: string; fontSize: number } = { padding: '6px 10px', fontSize: 13 }): React.CSSProperties {
  return {
    ...sizing, cursor: 'pointer',
    fontWeight: active ? 700 : 400,
    background: active ? '#1a1a38' : color.border,
    color: active ? color.info : color.textSecondary,
    border: active ? `1px solid ${color.info}` : '1px solid transparent',
    borderRadius: radius.sm,
  }
}
