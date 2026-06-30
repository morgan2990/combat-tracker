import { useState, useRef, useCallback } from 'react'
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import type { RoomState, Role, ProfileData } from './types'
import { JoinScreen } from './components/JoinScreen'
import { PlayerView } from './components/PlayerView'
import { DMView } from './components/DMView'
import { CharacterForm } from './components/CharacterForm'

type AppStatus = 'joining' | 'connecting' | 'connected' | 'disconnected'

const WS_ERRORS: Record<number, string> = {
  4004: 'Room not found.',
  4003: 'Wrong DM token.',
  4009: 'Name already taken.',
}

function buildWsUrl(roomId: string, name: string, role: Role, dmToken: string, maxHP?: number): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({ room_id: roomId, name, role })
  if (role === 'dm') params.set('dm_token', dmToken)
  if (maxHP) params.set('max_hp', String(maxHP))
  return `${proto}//${location.host}/ws?${params}`
}

export default function App() {
  const navigate = useNavigate()
  const [status, setStatus] = useState<AppStatus>('joining')
  const [role, setRole] = useState<Role>('player')
  const [_myName, setMyName] = useState('')
  const [myEntityId, setMyEntityId] = useState<string | null>(null)
  const [needsInitiative, setNeedsInitiative] = useState(false)
  const [roomState, setRoomState] = useState<RoomState | null>(null)
  const [joinError, setJoinError] = useState<string | null>(null)
  const [savedDmToken, setSavedDmToken] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const statusRef = useRef<AppStatus>('joining')

  function setStatusSync(s: AppStatus) {
    statusRef.current = s
    setStatus(s)
  }

  const sendMessage = useCallback((msg: object) => {
    wsRef.current?.send(JSON.stringify(msg))
  }, [])

  const connect = useCallback((
    roomId: string,
    name: string,
    selectedRole: Role,
    dmToken: string,
    profile?: ProfileData,
  ) => {
    setJoinError(null)
    setStatusSync('connecting')

    const ws = new WebSocket(buildWsUrl(roomId, name, selectedRole, dmToken, profile?.max_hp))
    wsRef.current = ws

    ws.onopen = () => {
      setRole(selectedRole)
      setMyName(name)
      if (selectedRole === 'dm') setSavedDmToken(dmToken)

      if (selectedRole === 'player' && profile) {
        ws.send(JSON.stringify({ type: 'setup_character' }))
        for (const companion of profile.companions) {
          ws.send(JSON.stringify({
            type: 'add_companion',
            name: companion.name,
            max_hp: companion.max_hp,
            shares_initiative: companion.shares_initiative,
          }))
        }
      }

      setStatusSync('connected')
      navigate('/room')
    }

    ws.onmessage = (event) => {
      try {
        const state = JSON.parse(event.data) as RoomState
        setRoomState(state)

        if (selectedRole === 'player') {
          const myEntity = state.entities.find(e => e.name === name && e.type === 'player')
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
        setStatusSync('joining')
      } else {
        setStatusSync('disconnected')
      }
    }

    ws.onerror = () => {
      // onclose always fires after onerror; handled there
    }
  }, [navigate])

  const backToJoin = useCallback(() => {
    wsRef.current?.close()
    setRoomState(null)
    setMyEntityId(null)
    setNeedsInitiative(false)
    setStatusSync('joining')
    navigate('/')
  }, [navigate])

  function GameView() {
    if (status === 'disconnected') {
      return (
        <div style={{ textAlign: 'center', marginTop: 80 }}>
          <div style={{ fontSize: 18, marginBottom: 16, color: '#d4d4e8' }}>Disconnected from server.</div>
          <button
            onClick={backToJoin}
            style={{ padding: '10px 24px', fontSize: 16, background: '#2e2e48', color: '#d4d4e8', border: 'none', borderRadius: 4, cursor: 'pointer' }}
          >
            Back to Join Screen
          </button>
        </div>
      )
    }

    if (!roomState) return null

    if (role === 'dm') {
      return <DMView roomState={roomState} sendMessage={sendMessage} dmToken={savedDmToken} />
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

  return (
    <Routes>
      <Route
        path="/"
        element={
          status === 'connected'
            ? <Navigate to="/room" replace />
            : <JoinScreen onJoin={connect} error={joinError} connecting={status === 'connecting'} />
        }
      />
      <Route path="/characters/new" element={<CharacterForm />} />
      <Route
        path="/room"
        element={status === 'joining' ? <Navigate to="/" replace /> : <GameView />}
      />
    </Routes>
  )
}
