import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import type { MeResponse, CustomMonster } from '../types'

interface DashboardProps {
  me: MeResponse
  onOpenRoomAsDM: (roomId: string) => void
  onJoinAsPlayer: (roomId: string, pcId: string) => void
  onLogout: () => void
}

export function Dashboard({ me, onOpenRoomAsDM, onJoinAsPlayer, onLogout }: DashboardProps) {
  const [dmEdition, setDmEdition] = useState<'5e' | '5.5e'>('5e')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const [joinRoomCode, setJoinRoomCode] = useState('')
  const [selectedPcId, setSelectedPcId] = useState(me.pcs[0]?.id ?? '')

  const [myMonsters, setMyMonsters] = useState<CustomMonster[]>([])

  useEffect(() => {
    fetch('/api/custom-monsters')
      .then(res => res.ok ? res.json() : [])
      .then((data: CustomMonster[]) => setMyMonsters(data))
      .catch(() => setMyMonsters([]))
  }, [])

  async function handleDeleteMonster(id: string) {
    if (!window.confirm('Delete this monster? This cannot be undone.')) return
    const res = await fetch(`/api/custom-monsters/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (res.ok) {
      setMyMonsters(prev => prev.filter(m => m.id !== id))
    }
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

  return (
    <div style={{ maxWidth: 760, margin: '0 auto', padding: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h1 style={{ margin: 0, fontSize: 24 }}>⚔ Combat Tracker</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 13, color: '#7878a0' }}>{me.user.display_name}</span>
          <button onClick={onLogout} style={logoutBtn}>Log Out</button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 20, flexWrap: 'wrap' }}>
        {/* As DM */}
        <div style={{ flex: 1, minWidth: 300 }}>
          <h2 style={sectionTitle}>As DM</h2>

          <div style={panelStyle}>
            <div style={{ fontSize: 13, color: '#7878a0', marginBottom: 8 }}>My Rooms</div>
            {me.rooms.length === 0 && (
              <div style={{ fontSize: 13, color: '#5a5a78', marginBottom: 12 }}>No rooms yet.</div>
            )}
            {me.rooms.map(room => (
              <div key={room.room_id} style={listRow}>
                <span>
                  {room.room_id}
                  {room.is_combat_active && <span style={{ marginLeft: 8, fontSize: 11, color: '#e67e22' }}>● in combat</span>}
                </span>
                <button onClick={() => onOpenRoomAsDM(room.room_id)} style={actionBtn}>Open</button>
              </div>
            ))}

            <div style={{ marginTop: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
              {(['5e', '5.5e'] as const).map(ed => (
                <button
                  key={ed}
                  type="button"
                  onClick={() => setDmEdition(ed)}
                  style={{
                    padding: '6px 10px', fontSize: 13, cursor: 'pointer',
                    fontWeight: dmEdition === ed ? 700 : 400,
                    background: dmEdition === ed ? '#1a1a38' : '#2e2e48',
                    color: dmEdition === ed ? '#3498db' : '#7878a0',
                    border: dmEdition === ed ? '1px solid #3498db' : '1px solid transparent',
                    borderRadius: 4,
                  }}
                >
                  {ed}
                </button>
              ))}
              <button onClick={handleCreateRoom} disabled={creating} style={primaryBtn(creating)}>
                {creating ? 'Creating…' : '+ New Room'}
              </button>
            </div>
            {createError && <div style={errorStyle}>{createError}</div>}

            <div style={{ fontSize: 13, color: '#7878a0', marginTop: 20, marginBottom: 8 }}>My Monsters</div>
            {myMonsters.length === 0 && (
              <div style={{ fontSize: 13, color: '#5a5a78', marginBottom: 12 }}>No custom monsters yet.</div>
            )}
            {myMonsters.map(m => (
              <div key={m.id} style={listRow}>
                <span>
                  {m.name}
                  {m.private && <span style={{ marginLeft: 8, fontSize: 11, color: '#7878a0' }}>● private</span>}
                </span>
                <span style={{ display: 'flex', gap: 8 }}>
                  <Link to={`/monsters/custom/${m.id}/edit`} style={{ fontSize: 12, color: '#3498db' }}>Edit</Link>
                  <button onClick={() => handleDeleteMonster(m.id)} style={deleteBtn}>Delete</button>
                </span>
              </div>
            ))}
            <Link to="/monsters/new" style={{ display: 'inline-block', marginTop: 8, fontSize: 13, color: '#3498db' }}>
              + New Monster
            </Link>
          </div>
        </div>

        {/* As Player */}
        <div style={{ flex: 1, minWidth: 300 }}>
          <h2 style={sectionTitle}>As Player</h2>

          <div style={panelStyle}>
            <div style={{ fontSize: 13, color: '#7878a0', marginBottom: 8 }}>My Characters</div>
            {me.pcs.length === 0 && (
              <div style={{ fontSize: 13, color: '#5a5a78', marginBottom: 12 }}>No characters yet.</div>
            )}
            {me.pcs.map(pc => (
              <div key={pc.id} style={listRow}>
                <span>{pc.name} <span style={{ color: '#5a5a78', fontSize: 11 }}>{pc.max_hp} HP</span></span>
                <Link to={`/characters/${pc.id}/edit`} style={{ fontSize: 12, color: '#3498db' }}>Edit</Link>
              </div>
            ))}
            <Link to="/characters/new" style={{ display: 'inline-block', marginTop: 8, fontSize: 13, color: '#3498db' }}>
              + New Character
            </Link>

            {me.recent_rooms.length > 0 && (
              <>
                <div style={{ fontSize: 13, color: '#7878a0', marginTop: 20, marginBottom: 8 }}>Recent Rooms</div>
                {me.recent_rooms.map(rm => {
                  const pc = me.pcs.find(p => p.id === rm.last_pc_id)
                  return (
                    <div key={rm.room_id} style={listRow}>
                      <span>{rm.room_id} <span style={{ color: '#5a5a78', fontSize: 11 }}>as {pc?.name ?? 'unknown'}</span></span>
                      <button
                        onClick={() => onJoinAsPlayer(rm.room_id, rm.last_pc_id)}
                        disabled={!pc}
                        style={actionBtn}
                      >
                        Go
                      </button>
                    </div>
                  )
                })}
              </>
            )}

            {me.pcs.length > 0 && (
              <div style={{ marginTop: 20 }}>
                <div style={{ fontSize: 13, color: '#7878a0', marginBottom: 8 }}>Join a new room</div>
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
        </div>
      </div>
    </div>
  )
}

const sectionTitle: React.CSSProperties = { fontSize: 16, marginBottom: 10, color: '#d4d4e8' }

const panelStyle: React.CSSProperties = {
  padding: '14px 16px', background: '#161626', border: '1px solid #2e2e48', borderRadius: 6,
}

const listRow: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
  padding: '8px 0', borderBottom: '1px solid #2e2e48', fontSize: 14,
}

const actionBtn: React.CSSProperties = {
  padding: '5px 12px', fontSize: 12, cursor: 'pointer',
  background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4,
}

const deleteBtn: React.CSSProperties = {
  fontSize: 12, cursor: 'pointer', padding: 0,
  background: 'none', color: '#e74c3c', border: 'none',
}

const logoutBtn: React.CSSProperties = {
  padding: '6px 12px', fontSize: 12, cursor: 'pointer',
  background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4,
}

const inputStyle: React.CSSProperties = {
  padding: '8px 10px', fontSize: 14, background: '#1a1a2c', border: '1px solid #2e2e48', borderRadius: 4, color: '#d4d4e8',
}

const selectStyle: React.CSSProperties = {
  ...inputStyle, maxWidth: 140,
}

const errorStyle: React.CSSProperties = {
  marginTop: 8, fontSize: 13, color: '#e74c3c',
}

function primaryBtn(disabled: boolean): React.CSSProperties {
  return {
    padding: '8px 16px', fontSize: 13, fontWeight: 700,
    background: '#27ae60', color: '#fff', border: 'none', borderRadius: 4,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}
