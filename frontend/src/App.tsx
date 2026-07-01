import { useState, useRef, useCallback, useEffect } from 'react'
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import type { RoomState, Role, MeResponse } from './types'
import { LoginScreen } from './components/LoginScreen'
import { Dashboard } from './components/Dashboard'
import { PlayerView } from './components/PlayerView'
import { DMView } from './components/DMView'
import { CharacterForm } from './components/CharacterForm'
import { MonsterForm } from './components/MonsterForm'

type AppStatus = 'idle' | 'connecting' | 'connected' | 'disconnected'

const WS_ERRORS: Record<number, string> = {
  4001: 'Not logged in.',
  4003: 'Forbidden.',
  4004: 'Room not found.',
  4009: 'That character is already in this room.',
}

function buildWsUrl(roomId: string, role: Role, pcId?: string): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({ room_id: roomId, role })
  if (role === 'player' && pcId) params.set('pc_id', pcId)
  return `${proto}//${location.host}/ws?${params}`
}

export default function App() {
  const navigate = useNavigate()
  const [me, setMe] = useState<MeResponse | null>(null)
  const [authChecked, setAuthChecked] = useState(false)
  const [status, setStatus] = useState<AppStatus>('idle')
  const [role, setRole] = useState<Role>('player')
  const [myPcId, setMyPcId] = useState<string | null>(null)
  const [myEntityId, setMyEntityId] = useState<string | null>(null)
  const [needsInitiative, setNeedsInitiative] = useState(false)
  const [roomState, setRoomState] = useState<RoomState | null>(null)
  const [joinError, setJoinError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const statusRef = useRef<AppStatus>('idle')

  const refreshMe = useCallback(async () => {
    try {
      const res = await fetch('/api/me')
      if (res.ok) {
        setMe(await res.json())
      } else {
        setMe(null)
      }
    } catch {
      setMe(null)
    } finally {
      setAuthChecked(true)
    }
  }, [])

  useEffect(() => {
    refreshMe()
  }, [refreshMe])

  function setStatusSync(s: AppStatus) {
    statusRef.current = s
    setStatus(s)
  }

  const sendMessage = useCallback((msg: object) => {
    wsRef.current?.send(JSON.stringify(msg))
  }, [])

  const connect = useCallback((roomId: string, selectedRole: Role, pcId?: string) => {
    setJoinError(null)
    setStatusSync('connecting')

    const ws = new WebSocket(buildWsUrl(roomId, selectedRole, pcId))
    wsRef.current = ws

    ws.onopen = () => {
      setRole(selectedRole)
      setMyPcId(pcId ?? null)

      if (selectedRole === 'player') {
        ws.send(JSON.stringify({ type: 'setup_character' }))
      }

      setStatusSync('connected')
      navigate('/room')
    }

    ws.onmessage = (event) => {
      try {
        const state = JSON.parse(event.data) as RoomState
        setRoomState(state)

        if (selectedRole === 'player' && pcId) {
          const myEntity = state.entities.find(e => e.pc_id === pcId && e.type === 'pc')
          if (myEntity) {
            setMyEntityId(myEntity.id)
            setNeedsInitiative(myEntity.initiative === null)
          }
        }
      } catch {
        // ignore malformed messages
      }
    }

    ws.onclose = (event) => {
      if (statusRef.current !== 'connected') {
        setJoinError(WS_ERRORS[event.code] ?? 'Failed to connect. Check the room code and try again.')
        setStatusSync('idle')
      } else {
        setStatusSync('disconnected')
      }
    }

    ws.onerror = () => {
      // onclose always fires after onerror; handled there
    }
  }, [navigate])

  const backToDashboard = useCallback(() => {
    wsRef.current?.close()
    setRoomState(null)
    setMyEntityId(null)
    setMyPcId(null)
    setNeedsInitiative(false)
    setStatusSync('idle')
    refreshMe()
    navigate('/')
  }, [navigate, refreshMe])

  const handleLogout = useCallback(async () => {
    await fetch('/api/logout', { method: 'POST' })
    wsRef.current?.close()
    setMe(null)
    setRoomState(null)
    navigate('/')
  }, [navigate])

  function GameView() {
    if (status === 'disconnected') {
      return (
        <div style={{ textAlign: 'center', marginTop: 80 }}>
          <div style={{ fontSize: 18, marginBottom: 16, color: '#d4d4e8' }}>Disconnected from server.</div>
          <button
            onClick={backToDashboard}
            style={{ padding: '10px 24px', fontSize: 16, background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4, cursor: 'pointer' }}
          >
            Back to Dashboard
          </button>
        </div>
      )
    }

    if (!roomState) return null

    if (role === 'dm') {
      return <DMView roomState={roomState} sendMessage={sendMessage} />
    }

    return (
      <PlayerView
        roomState={roomState}
        myEntityId={myEntityId}
        needsInitiative={needsInitiative}
        sendMessage={sendMessage}
      />
    )
  }

  if (!authChecked) return null

  return (
    <Routes>
      <Route
        path="/"
        element={
          status === 'connected'
            ? <Navigate to="/room" replace />
            : me
              ? (
                <>
                  {joinError && (
                    <div style={{ maxWidth: 760, margin: '12px auto 0', padding: '10px 16px', background: '#1a0808', border: '1px solid #e74c3c', borderRadius: 6, color: '#e74c3c', fontSize: 13 }}>
                      {joinError}
                    </div>
                  )}
                  <Dashboard
                    me={me}
                    onOpenRoomAsDM={roomId => connect(roomId, 'dm')}
                    onJoinAsPlayer={(roomId, pcId) => connect(roomId, 'player', pcId)}
                    onLogout={handleLogout}
                  />
                </>
              )
              : <LoginScreen onAuthed={refreshMe} />
        }
      />
      <Route path="/characters/new" element={<CharacterForm onSaved={async () => { await refreshMe(); navigate('/') }} />} />
      <Route path="/characters/:id/edit" element={<CharacterForm onSaved={async () => { await refreshMe(); navigate('/') }} />} />
      <Route path="/monsters/new" element={<MonsterForm />} />
      <Route
        path="/room"
        element={status === 'idle' ? <Navigate to="/" replace /> : <GameView />}
      />
    </Routes>
  )
}
