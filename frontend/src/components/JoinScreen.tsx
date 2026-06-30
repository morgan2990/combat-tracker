import { useState } from 'react'
import type { FormEvent } from 'react'
import type { Role } from '../types'

interface JoinScreenProps {
  onJoin: (roomId: string, name: string, role: Role, dmToken: string) => void
  error: string | null
  connecting: boolean
}

export function JoinScreen({ onJoin, error, connecting }: JoinScreenProps) {
  const [tab, setTab] = useState<Role>('player')

  // Player fields
  const [playerRoomId, setPlayerRoomId] = useState('')
  const [playerName, setPlayerName] = useState('')

  // DM shared name + rejoin fields
  const [dmName, setDmName] = useState('')
  const [rejoinRoomId, setRejoinRoomId] = useState('')
  const [dmToken, setDmToken] = useState('')

  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  function handlePlayerJoin(e: FormEvent) {
    e.preventDefault()
    if (!playerRoomId.trim() || !playerName.trim()) return
    onJoin(playerRoomId.trim().toUpperCase(), playerName.trim(), 'player', '')
  }

  function handleRejoin(e: FormEvent) {
    e.preventDefault()
    if (!rejoinRoomId.trim() || !dmName.trim() || !dmToken.trim()) return
    onJoin(rejoinRoomId.trim().toUpperCase(), dmName.trim(), 'dm', dmToken.trim())
  }

  async function handleCreateRoom() {
    if (!dmName.trim()) return
    setCreating(true)
    setCreateError(null)
    try {
      const res = await fetch('/api/rooms', { method: 'POST' })
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      const data = await res.json()
      onJoin(data.room_id, dmName.trim(), 'dm', data.dm_token)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create room')
      setCreating(false)
    }
  }

  return (
    <div style={{ maxWidth: 380, margin: '80px auto', padding: 24 }}>
      <h1 style={{ marginBottom: 24, fontSize: 26, fontWeight: 700 }}>⚔ Combat Tracker</h1>

      {/* Tab toggle */}
      <div style={{ display: 'flex', marginBottom: 20, borderBottom: '2px solid #2e2e48' }}>
        {(['player', 'dm'] as Role[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              flex: 1, padding: '8px 0', fontSize: 14,
              fontWeight: tab === t ? 700 : 400,
              background: 'none', border: 'none',
              borderBottom: tab === t ? '2px solid #3498db' : '2px solid transparent',
              marginBottom: -2, cursor: 'pointer',
              color: tab === t ? '#3498db' : '#7878a0',
            }}
          >
            {t === 'player' ? 'Player' : 'Dungeon Master'}
          </button>
        ))}
      </div>

      {tab === 'player' ? (
        <form onSubmit={handlePlayerJoin}>
          <label>
            <div style={labelText}>Room Code</div>
            <input
              value={playerRoomId}
              onChange={e => setPlayerRoomId(e.target.value.toUpperCase())}
              maxLength={6}
              placeholder="XXXXX"
              required
              style={inputStyle}
            />
          </label>
          <label style={{ marginTop: 12, display: 'block' }}>
            <div style={labelText}>Character Name</div>
            <input
              value={playerName}
              onChange={e => setPlayerName(e.target.value)}
              placeholder="Your name"
              required
              style={inputStyle}
            />
          </label>
          {error && <div style={errorStyle}>{error}</div>}
          <button type="submit" disabled={connecting} style={primaryBtn(connecting)}>
            {connecting ? 'Connecting…' : 'Join Room'}
          </button>
        </form>
      ) : (
        <div>
          {/* Shared name field */}
          <label>
            <div style={labelText}>Your Name</div>
            <input
              value={dmName}
              onChange={e => setDmName(e.target.value)}
              placeholder="Dungeon Master"
              style={inputStyle}
            />
          </label>

          {/* Create New Room */}
          <div style={{ marginTop: 16 }}>
            <button
              onClick={handleCreateRoom}
              disabled={creating || connecting || !dmName.trim()}
              style={primaryBtn(creating || connecting || !dmName.trim())}
            >
              {creating ? 'Creating…' : '▶ Create New Room'}
            </button>
            {createError && <div style={errorStyle}>{createError}</div>}
          </div>

          {/* Divider */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, margin: '20px 0' }}>
            <div style={{ flex: 1, height: 1, background: '#2e2e48' }} />
            <span style={{ fontSize: 12, color: '#5a5a78', whiteSpace: 'nowrap' }}>or rejoin existing</span>
            <div style={{ flex: 1, height: 1, background: '#2e2e48' }} />
          </div>

          {/* Rejoin Existing Room */}
          <form onSubmit={handleRejoin}>
            <label>
              <div style={labelText}>Room Code</div>
              <input
                value={rejoinRoomId}
                onChange={e => setRejoinRoomId(e.target.value.toUpperCase())}
                maxLength={6}
                placeholder="XXXXX"
                style={inputStyle}
              />
            </label>
            <label style={{ marginTop: 12, display: 'block' }}>
              <div style={labelText}>DM Token</div>
              <input
                value={dmToken}
                onChange={e => setDmToken(e.target.value)}
                placeholder="Token from room creation"
                style={inputStyle}
              />
            </label>
            {error && <div style={errorStyle}>{error}</div>}
            <button
              type="submit"
              disabled={connecting || !dmName.trim() || !rejoinRoomId.trim() || !dmToken.trim()}
              style={{ ...primaryBtn(connecting || !dmName.trim() || !rejoinRoomId.trim() || !dmToken.trim()), marginTop: 12 }}
            >
              {connecting ? 'Connecting…' : '→ Rejoin Room'}
            </button>
          </form>
        </div>
      )}
    </div>
  )
}

const labelText: React.CSSProperties = { fontSize: 13, color: '#7878a0', marginBottom: 4 }

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 12px',
  fontSize: 16,
  boxSizing: 'border-box',
  marginTop: 4,
}

const errorStyle: React.CSSProperties = {
  marginTop: 10,
  color: '#e74c3c',
  fontWeight: 'bold',
  fontSize: 14,
}

function primaryBtn(disabled: boolean): React.CSSProperties {
  return {
    marginTop: 16,
    width: '100%',
    padding: '12px 0',
    fontSize: 16,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
    background: '#e67e22',
    color: '#fff',
    border: 'none',
    fontWeight: 600,
  }
}
