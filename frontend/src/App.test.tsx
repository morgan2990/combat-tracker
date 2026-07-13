import { act } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import App from './App'
import type { MeResponse, RoomState } from './types'
import { useLayoutTier } from './hooks/useLayoutTier'

vi.mock('./hooks/useLayoutTier')

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: ((event: { code: number }) => void) | null = null
  onerror: (() => void) | null = null
  sent: string[] = []
  closed = false

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  send(data: string) { this.sent.push(data) }
  close() { this.closed = true }
}

function lastSocket(): FakeWebSocket {
  return FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
}

function meResponse(overrides: Partial<MeResponse> = {}): MeResponse {
  return {
    user: { username: 'aria', display_name: 'Aria' },
    rooms: [], pcs: [], recent_rooms: [], parties: [],
    ...overrides,
  }
}

function room(overrides: Partial<RoomState> = {}): RoomState {
  return { room_id: 'ROOM1', edition: '5e', is_started: false, round: 1, active_index: 0, entities: [], ...overrides }
}

beforeEach(() => {
  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket)
  vi.mocked(useLayoutTier).mockReturnValue('desktop')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.mocked(useLayoutTier).mockReset()
})

function jsonOk(body: unknown) {
  return { ok: true, json: () => Promise.resolve(body), text: () => Promise.resolve('') }
}

// Always answers /api/me, /api/custom-monsters, and /api/encounters (both
// fetched unconditionally by Dashboard/App on mount) sensibly by default, so
// individual tests only need to layer in the routes they actually care about.
function stubFetch(me: MeResponse | null, extra?: (url: string, init?: RequestInit) => unknown) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => {
    const custom = extra?.(url, init)
    if (custom !== undefined) return Promise.resolve(custom)
    if (url === '/api/me') return Promise.resolve(me ? jsonOk(me) : { ok: false, json: () => Promise.resolve(null) })
    if (url.includes('/party-inventory')) return Promise.resolve(jsonOk({ members: [] }))
    return Promise.resolve(jsonOk([]))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderApp(initialEntries: string[] = ['/']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <App />
    </MemoryRouter>,
  )
}

describe('App', () => {
  it('shows the login screen when not authenticated', async () => {
    stubFetch(null)
    renderApp()
    expect(await screen.findByText('⚔ Combat Tracker')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('e.g. aria')).toBeInTheDocument()
  })

  it('shows the dashboard once authenticated', async () => {
    stubFetch(meResponse())
    renderApp()
    expect(await screen.findByText('Aria')).toBeInTheDocument()
    expect(screen.getByText('My Rooms')).toBeInTheDocument()
  })

  it('redirects /room to / when not connected', async () => {
    stubFetch(meResponse())
    renderApp(['/room'])
    expect(await screen.findByText('My Rooms')).toBeInTheDocument()
  })

  it('connects as DM, opens a WebSocket to the right URL, and renders DMView on the first message', async () => {
    stubFetch(meResponse({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')

    await user.click(screen.getByRole('button', { name: 'Open' }))

    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const ws = lastSocket()
    expect(ws.url).toBe(`ws://${location.host}/ws?room_id=ROOM1&role=dm`)

    act(() => ws.onopen?.())
    act(() => ws.onmessage?.({ data: JSON.stringify(room()) }))

    expect(await screen.findByText('⚔ DM Panel')).toBeInTheDocument()
  })

  it('connects as a player with a pc_id in the URL, sends setup_character, and derives myEntityId/needsInitiative', async () => {
    stubFetch(meResponse({
      pcs: [{ id: 'pc1', owner_user_id: 'o1', name: 'Aria', type: 'pc', max_hp: 30, shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }],
      recent_rooms: [{ user_id: 'o1', room_id: 'ROOM1', last_pc_id: 'pc1', last_joined_at: '2026-01-01' }],
    }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')

    // Desktop tier shows both DM and Player panels at once, so the recent
    // room's "Go" button is directly visible without switching panels. It's
    // ambiguous with the (disabled, code-empty) "join a new room" Go button.
    const goButtons = screen.getAllByRole('button', { name: 'Go' })
    await user.click(goButtons.find(b => !b.hasAttribute('disabled'))!)

    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const ws = lastSocket()
    expect(ws.url).toBe(`ws://${location.host}/ws?room_id=ROOM1&role=player&pc_id=pc1`)

    act(() => ws.onopen?.())
    expect(ws.sent).toEqual([JSON.stringify({ type: 'setup_character' })])

    act(() => ws.onmessage?.({
      data: JSON.stringify(room({ entities: [{ id: 'e1', name: 'Aria', type: 'pc', pc_id: 'pc1', max_hp: 30, current_hp: 30, temp_hp: 0, initiative: null, shares_initiative: false, conditions: [], dead: false, is_hidden: false }] })),
    }))

    expect(await screen.findByText('Set your initiative')).toBeInTheDocument()
  })

  it('shows a disconnected screen after the socket closes post-connect, and returns to the dashboard', async () => {
    stubFetch(meResponse({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')
    await user.click(screen.getByRole('button', { name: 'Open' }))
    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const ws = lastSocket()
    act(() => ws.onopen?.())
    act(() => ws.onmessage?.({ data: JSON.stringify(room()) }))
    await screen.findByText('⚔ DM Panel')

    act(() => ws.onclose?.({ code: 1006 }))

    expect(await screen.findByText('Disconnected from server.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Back to Dashboard' }))
    expect(await screen.findByText('My Rooms')).toBeInTheDocument()
  })

  it('shows a mapped join error when the socket closes before connecting', async () => {
    stubFetch(meResponse({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')
    await user.click(screen.getByRole('button', { name: 'Open' }))
    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const ws = lastSocket()

    act(() => ws.onclose?.({ code: 4004 }))

    expect(await screen.findByText('Room not found.')).toBeInTheDocument()
    expect(screen.getByText('My Rooms')).toBeInTheDocument()
  })

  it('shows a generic join error for an unmapped close code', async () => {
    stubFetch(meResponse({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')
    await user.click(screen.getByRole('button', { name: 'Open' }))
    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const ws = lastSocket()

    act(() => ws.onclose?.({ code: 1006 }))

    expect(await screen.findByText('Failed to connect. Check the room code and try again.')).toBeInTheDocument()
  })

  it('ignores a stale socket close after the ref has already been abandoned', async () => {
    stubFetch(meResponse({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('My Rooms')
    await user.click(screen.getByRole('button', { name: 'Open' }))
    await waitFor(() => expect(FakeWebSocket.instances.length).toBe(1))
    const staleWs = lastSocket()
    act(() => staleWs.onopen?.())
    act(() => staleWs.onmessage?.({ data: JSON.stringify(room()) }))
    await screen.findByText('⚔ DM Panel')

    await user.click(screen.getByRole('button', { name: '← Dashboard' }))
    await screen.findByText('My Rooms')

    // The abandoned socket's close event arrives late; it must not resurrect
    // a disconnect/join-error banner now that wsRef points elsewhere.
    act(() => staleWs.onclose?.({ code: 1006 }))
    expect(screen.queryByText('Disconnected from server.')).not.toBeInTheDocument()
    expect(screen.getByText('My Rooms')).toBeInTheDocument()
  })

  it('logs out: posts to /api/logout, closes the socket, and shows the login screen', async () => {
    const fetchMock = stubFetch(meResponse(), url => {
      if (url === '/api/logout') return { ok: true }
      return undefined
    })
    const user = userEvent.setup()
    renderApp()
    await screen.findByText('Aria')

    await user.click(screen.getByRole('button', { name: 'Log Out' }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/logout', { method: 'POST' }))
    expect(await screen.findByPlaceholderText('e.g. aria')).toBeInTheDocument()
  })

  it('routes to the monster/encounter/character forms directly', async () => {
    stubFetch(null)
    renderApp(['/monsters/new'])
    expect(await screen.findByText('Register Monster')).toBeInTheDocument()
  })
})
