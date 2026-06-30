import { useState, useRef, useCallback } from 'react'
import type { RoomState, Role } from './types'
import { JoinScreen } from './components/JoinScreen'
import { PlayerView } from './components/PlayerView'
import { DMView } from './components/DMView'
import { SetupForm } from './components/SetupForm'

type AppStatus = 'joining' | 'connecting' | 'connected' | 'disconnected'

const WS_ERRORS: Record<number, string> = {
  4004: 'Room not found.',
  4003: 'Wrong DM token.',
  4009: 'Name already taken.',
}

function buildWsUrl(roomId: string, name: string, role: Role, dmToken: string): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({ room_id: roomId, name, role })
  if (role === 'dm') params.set('dm_token', dmToken)
  return `${proto}//${location.host}/ws?${params}`
}

export default function App() {
  const [status, setStatus] = useState<AppStatus>('joining')
  const [role, setRole] = useState<Role>('player')
  const [myName, setMyName] = useState('')
  const [myEntityId, setMyEntityId] = useState<string | null>(null)
  const [needsSetup, setNeedsSetup] = useState(false)
  const [roomState, setRoomState] = useState<RoomState | null>(null)
  const [joinError, setJoinError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const statusRef = useRef<AppStatus>('joining')

  function setStatusSync(s: AppStatus) {
    statusRef.current = s
    setStatus(s)
  }

  const sendMessage = useCallback((msg: object) => {
    wsRef.current?.send(JSON.stringify(msg))
  }, [])

  const connect = useCallback((roomId: string, name: string, selectedRole: Role, dmToken: string) => {
    setJoinError(null)
    setStatusSync('connecting')

    const ws = new WebSocket(buildWsUrl(roomId, name, selectedRole, dmToken))
    wsRef.current = ws

    ws.onopen = () => {
      setRole(selectedRole)
      setMyName(name)
      setStatusSync('connected')
    }

    ws.onmessage = (event) => {
      try {
        const state = JSON.parse(event.data) as RoomState
        setRoomState(state)

        if (selectedRole === 'player') {
          const myEntity = state.entities.find(e => e.name === name && e.type === 'player')
          setNeedsSetup(!myEntity)
          if (myEntity) setMyEntityId(myEntity.id)
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
  }, [])

  const backToJoin = useCallback(() => {
    wsRef.current?.close()
    setRoomState(null)
    setMyEntityId(null)
    setNeedsSetup(false)
    setStatusSync('joining')
  }, [])

  if (status === 'disconnected') {
    return (
      <div style={{ textAlign: 'center', marginTop: 80, fontFamily: 'sans-serif' }}>
        <div style={{ fontSize: 18, marginBottom: 16 }}>Disconnected from server.</div>
        <button onClick={backToJoin} style={{ padding: '10px 24px', fontSize: 16 }}>
          Back to Join Screen
        </button>
      </div>
    )
  }

  if (status === 'joining' || status === 'connecting') {
    return (
      <JoinScreen
        onJoin={connect}
        error={joinError}
        connecting={status === 'connecting'}
      />
    )
  }

  if (!roomState) return null

  if (role === 'dm') {
    return <DMView roomState={roomState} sendMessage={sendMessage} />
  }

  if (needsSetup) {
    return <SetupForm myName={myName} onSubmit={sendMessage} />
  }

  return (
    <PlayerView
      roomState={roomState}
      myEntityId={myEntityId}
      sendMessage={sendMessage}
    />
  )
}
