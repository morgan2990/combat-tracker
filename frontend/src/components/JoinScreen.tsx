import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import type { Role, ProfileData } from '../types'

interface JoinScreenProps {
  onJoin: (roomId: string, name: string, role: Role, dmToken: string, profile?: ProfileData) => void
  error: string | null
  connecting: boolean
}

export function JoinScreen({ onJoin, error, connecting }: JoinScreenProps) {
  const [tab, setTab] = useState<Role>('player')

  // Player fields
  const [playerRoomId, setPlayerRoomId] = useState('')
  const [playerName, setPlayerName] = useState('')
  const [profile, setProfile] = useState<ProfileData | null>(null)
  const [fetchingProfile, setFetchingProfile] = useState(false)
  const [profileError, setProfileError] = useState<string | null>(null)

  // DM shared name + rejoin fields
  const [dmName, setDmName] = useState('')
  const [rejoinRoomId, setRejoinRoomId] = useState('')
  const [dmToken, setDmToken] = useState('')

  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  function resetProfile() {
    setProfile(null)
    setProfileError(null)
  }

  async function handleFindCharacter() {
    if (!playerName.trim()) return
    setFetchingProfile(true)
    setProfileError(null)
    setProfile(null)
    try {
      const res = await fetch(`/api/entities/${encodeURIComponent(playerName.trim())}`)
      if (res.status === 404) {
        setProfileError('not_found')
        return
      }
      if (!res.ok) {
        setProfileError('error')
        return
      }
      const data = await res.json()
      setProfile({
        max_hp: data.profile.max_hp,
        companions: (data.companions as Array<{ name: string; max_hp: number; shares_initiative: boolean }>).map(c => ({
          name: c.name,
          max_hp: c.max_hp,
          shares_initiative: c.shares_initiative,
        })),
      })
    } catch {
      setProfileError('error')
    } finally {
      setFetchingProfile(false)
    }
  }

  function handlePlayerJoin(e: FormEvent) {
    e.preventDefault()
    if (!playerRoomId.trim() || !playerName.trim() || !profile) return
    onJoin(playerRoomId.trim().toUpperCase(), playerName.trim(), 'player', '', profile)
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
            <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
              <input
                value={playerName}
                onChange={e => { setPlayerName(e.target.value); resetProfile() }}
                placeholder="Your character's name"
                required
                style={{ ...inputStyle, marginTop: 0, flex: 1 }}
              />
              <button
                type="button"
                onClick={handleFindCharacter}
                disabled={fetchingProfile || !playerName.trim()}
                style={{
                  padding: '8px 12px', fontSize: 13, cursor: fetchingProfile || !playerName.trim() ? 'not-allowed' : 'pointer',
                  opacity: fetchingProfile || !playerName.trim() ? 0.45 : 1,
                  background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4, whiteSpace: 'nowrap',
                }}
              >
                {fetchingProfile ? '…' : 'Find'}
              </button>
            </div>
          </label>

          {/* Profile found */}
          {profile && (
            <div style={{ marginTop: 12, padding: '10px 12px', background: '#0f1f10', border: '1px solid #27ae60', borderRadius: 6, fontSize: 13 }}>
              <div style={{ color: '#27ae60', fontWeight: 600, marginBottom: 4 }}>Profile found</div>
              <div style={{ color: '#d4d4e8' }}>Max HP: <strong>{profile.max_hp}</strong></div>
              {profile.companions.length > 0 && (
                <div style={{ color: '#8888aa', marginTop: 4 }}>
                  Companions: {profile.companions.map(c => c.name).join(', ')}
                </div>
              )}
            </div>
          )}

          {/* Profile not found */}
          {profileError === 'not_found' && (
            <div style={{ marginTop: 12, padding: '10px 12px', background: '#1a0808', border: '1px solid #e74c3c', borderRadius: 6, fontSize: 13, color: '#e74c3c' }}>
              Character not found.{' '}
              <Link to="/characters/new" style={{ color: '#e07070' }}>Create a profile</Link> first.
            </div>
          )}

          {/* Service error */}
          {profileError === 'error' && (
            <div style={{ marginTop: 12, padding: '10px 12px', background: '#1a0808', border: '1px solid #e74c3c', borderRadius: 6, fontSize: 13, color: '#e74c3c' }}>
              Service unavailable. Please try again.
            </div>
          )}

          {error && <div style={errorStyle}>{error}</div>}

          <button
            type="submit"
            disabled={connecting || !profile}
            style={primaryBtn(connecting || !profile)}
          >
            {connecting ? 'Connecting…' : 'Join Room'}
          </button>

          <div style={{ marginTop: 12, textAlign: 'center', fontSize: 13, color: '#5a5a78' }}>
            No profile yet?{' '}
            <Link to="/characters/new" style={{ color: '#7878a0' }}>Create your character</Link>
          </div>
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
