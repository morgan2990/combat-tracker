import { useState } from 'react'
import type { FormEvent } from 'react'
import type { Role } from '../types'

interface JoinScreenProps {
  onJoin: (roomId: string, name: string, role: Role, dmToken: string) => void
  error: string | null
  connecting: boolean
}

export function JoinScreen({ onJoin, error, connecting }: JoinScreenProps) {
  const [roomId, setRoomId] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState<Role>('player')
  const [dmToken, setDmToken] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!roomId.trim() || !name.trim()) return
    onJoin(roomId.trim().toUpperCase(), name.trim(), role, dmToken.trim())
  }

  return (
    <div style={{ maxWidth: 360, margin: '80px auto', padding: 24, fontFamily: 'sans-serif' }}>
      <h1 style={{ marginBottom: 24 }}>⚔ Combat Tracker</h1>
      <form onSubmit={handleSubmit}>
        <label>
          <div>Room Code</div>
          <input
            value={roomId}
            onChange={e => setRoomId(e.target.value.toUpperCase())}
            maxLength={6}
            placeholder="XXXXX"
            required
            style={inputStyle}
          />
        </label>

        <label style={{ marginTop: 12, display: 'block' }}>
          <div>Character Name</div>
          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Your name"
            required
            style={inputStyle}
          />
        </label>

        <div style={{ marginTop: 12 }}>
          <div>Role</div>
          <label style={{ marginRight: 16 }}>
            <input type="radio" value="player" checked={role === 'player'} onChange={() => setRole('player')} />
            {' '}Player
          </label>
          <label>
            <input type="radio" value="dm" checked={role === 'dm'} onChange={() => setRole('dm')} />
            {' '}Dungeon Master
          </label>
        </div>

        {role === 'dm' && (
          <label style={{ marginTop: 12, display: 'block' }}>
            <div>DM Token</div>
            <input
              value={dmToken}
              onChange={e => setDmToken(e.target.value)}
              placeholder="Token from room creation"
              required
              style={inputStyle}
            />
          </label>
        )}

        {error && (
          <div style={{ marginTop: 12, color: '#c0392b', fontWeight: 'bold' }}>
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={connecting}
          style={{ marginTop: 20, width: '100%', padding: '12px 0', fontSize: 16, cursor: connecting ? 'wait' : 'pointer' }}
        >
          {connecting ? 'Connecting…' : 'Join Room'}
        </button>
      </form>
    </div>
  )
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 12px',
  fontSize: 16,
  boxSizing: 'border-box',
  marginTop: 4,
}
