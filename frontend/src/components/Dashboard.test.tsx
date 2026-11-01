import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import { Dashboard } from './Dashboard'
import type { MeResponse, PC, Party } from '../types'
import { useLayoutTier } from '../hooks/useLayoutTier'
import type { LayoutTier } from '../hooks/useLayoutTier'

vi.mock('../hooks/useLayoutTier')

afterEach(() => {
  vi.unstubAllGlobals()
  vi.mocked(useLayoutTier).mockReset()
  vi.restoreAllMocks()
})

function setTier(tier: LayoutTier) {
  vi.mocked(useLayoutTier).mockReturnValue(tier)
}

function pc(overrides: Partial<PC> = {}): PC {
  return {
    id: 'pc1', owner_user_id: 'o1', name: 'Aria', type: 'pc', max_hp: 30,
    shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
    ...overrides,
  }
}

function party(overrides: Partial<Party> = {}): Party {
  return {
    id: 'party1', name: 'The Fellowship', join_code: 'ABCDE', member_pc_ids: [],
    member_names: {}, currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
    ...overrides,
  }
}

function me(overrides: Partial<MeResponse> = {}): MeResponse {
  return {
    user: { username: 'aria', display_name: 'Aria' },
    rooms: [], pcs: [], recent_rooms: [], parties: [],
    ...overrides,
  }
}

function stubFetch(router: (url: string, init?: RequestInit) => unknown = () => ({ ok: true, json: () => Promise.resolve([]) })) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => Promise.resolve(router(url, init)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderDashboard(meResponse: MeResponse, overrides: Partial<{ onOpenRoomAsDM: (id: string) => void; onJoinAsPlayer: (id: string, pc: string) => void; onLogout: () => void }> = {}) {
  const onOpenRoomAsDM = overrides.onOpenRoomAsDM ?? vi.fn()
  const onJoinAsPlayer = overrides.onJoinAsPlayer ?? vi.fn()
  const onLogout = overrides.onLogout ?? vi.fn()
  render(
    <MemoryRouter>
      <Dashboard me={meResponse} onOpenRoomAsDM={onOpenRoomAsDM} onJoinAsPlayer={onJoinAsPlayer} onLogout={onLogout} />
    </MemoryRouter>,
  )
  return { onOpenRoomAsDM, onJoinAsPlayer, onLogout }
}

describe('Dashboard', () => {
  it('renders the display name and calls onLogout', async () => {
    setTier('desktop')
    stubFetch()
    const user = userEvent.setup()
    const { onLogout } = renderDashboard(me())

    expect(screen.getByText('Aria')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Log Out' }))
    expect(onLogout).toHaveBeenCalled()
  })

  it('phone tier: toggles between DM and Player panels', async () => {
    setTier('phone')
    stubFetch()
    const user = userEvent.setup()
    renderDashboard(me())

    expect(screen.getByRole('button', { name: 'As DM' })).toBeInTheDocument()
    expect(screen.getByText('My Rooms')).toBeInTheDocument()
    expect(screen.queryByText('My Characters')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'As Player' }))
    expect(screen.getByText('My Characters')).toBeInTheDocument()
    expect(screen.queryByText('My Rooms')).not.toBeInTheDocument()
  })

  it('desktop tier: shows both panels at once', () => {
    setTier('desktop')
    stubFetch()
    renderDashboard(me())
    expect(screen.getByText('My Rooms')).toBeInTheDocument()
    expect(screen.getByText('My Characters')).toBeInTheDocument()
  })

  describe('DM panel: rooms', () => {
    it('lists rooms with an in-combat badge, and opens one', async () => {
      setTier('desktop')
      stubFetch()
      const user = userEvent.setup()
      const { onOpenRoomAsDM } = renderDashboard(me({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: true }] }))

      expect(screen.getByText('● in combat')).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Open' }))
      expect(onOpenRoomAsDM).toHaveBeenCalledWith('ROOM1')
    })

    it('deletes a room after confirming', async () => {
      setTier('desktop')
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
        if (init?.method === 'DELETE') return { ok: true }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({ rooms: [{ room_id: 'ROOM1', edition: '5e', is_combat_active: false }] }))

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      await waitFor(() => expect(screen.queryByText('ROOM1')).not.toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/rooms/ROOM1', { method: 'DELETE' })
    })

    it('creates a room and opens it as DM', async () => {
      setTier('desktop')
      const fetchMock = stubFetch((url: string) => {
        if (url === '/api/rooms') return { ok: true, json: () => Promise.resolve({ room_id: 'NEWROOM' }) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      const { onOpenRoomAsDM } = renderDashboard(me())

      await user.click(screen.getByRole('button', { name: '+ New Room' }))

      await waitFor(() => expect(onOpenRoomAsDM).toHaveBeenCalledWith('NEWROOM'))
      const postCall = fetchMock.mock.calls.find(c => c[0] === '/api/rooms')!
      expect(JSON.parse((postCall[1] as RequestInit).body as string)).toEqual({ edition: '5e' })
    })

    it('shows an error when room creation fails', async () => {
      setTier('desktop')
      stubFetch((url: string) => {
        if (url === '/api/rooms') return { ok: false, status: 500 }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me())

      await user.click(screen.getByRole('button', { name: '+ New Room' }))

      expect(await screen.findByText('Server error 500')).toBeInTheDocument()
    })
  })

  describe('DM panel: monsters and encounters', () => {
    it('lists My Monsters (with private badge) and deletes one', async () => {
      setTier('desktop')
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const fetchMock = stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/custom-monsters') return { ok: true, json: () => Promise.resolve([{ id: 'cm1', name: 'Bandit', edition: '5e', max_hp: 10, initiative_modifier: null, is_custom: true, private: true, owner_id: 'o1', owner_display_name: 'Aria' }]) }
        if (init?.method === 'DELETE') return { ok: true }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me())

      expect(await screen.findByText('● private')).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Delete' }))

      await waitFor(() => expect(screen.queryByText('Bandit')).not.toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters/cm1', { method: 'DELETE' })
    })

    it('lists My Encounters and deletes one', async () => {
      setTier('desktop')
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const fetchMock = stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/encounters') return { ok: true, json: () => Promise.resolve([{ id: 'enc1', name: 'Goblin Ambush', owner_id: 'o1', edition: '5e', monsters: [] }]) }
        if (init?.method === 'DELETE') return { ok: true }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me())

      expect(await screen.findByText('Goblin Ambush')).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Delete' }))

      await waitFor(() => expect(screen.queryByText('Goblin Ambush')).not.toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/encounters/enc1', { method: 'DELETE' })
    })
  })

  describe('Player panel: characters', () => {
    it('links each character to its sheet page', () => {
      setTier('desktop')
      stubFetch()
      renderDashboard(me({ pcs: [pc()] }))

      expect(screen.getByRole('link', { name: 'Sheet' })).toHaveAttribute('href', '/characters/pc1/sheet')
    })

    it('lists characters with HP, opens inventory, and deletes one', async () => {
      setTier('desktop')
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      stubFetch((url: string, init?: RequestInit) => {
        if (init?.method === 'DELETE') return { ok: true }
        if (url === '/api/pcs/pc1') return { ok: true, json: () => Promise.resolve({ pc: { name: 'Aria', max_hp: 30, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } } }) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({ pcs: [pc()] }))

      expect(screen.getByText('30 HP')).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: 'Inventory' }))
      expect(await screen.findByText("Aria's Inventory")).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: '✕' }))

      await user.click(screen.getByRole('button', { name: 'Delete' }))
      await waitFor(() => expect(screen.queryByText('30 HP')).not.toBeInTheDocument())
    })
  })

  describe('Player panel: parties', () => {
    it('creates a party', async () => {
      setTier('desktop')
      const fetchMock = stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/parties' && init?.method === 'POST') return { ok: true, json: () => Promise.resolve(party({ id: 'party-new', name: 'New Party' })) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me())

      await user.type(screen.getByPlaceholderText('Party name'), 'New Party')
      await user.click(screen.getByRole('button', { name: '+ New Party' }))

      await waitFor(() => expect(screen.getByText('New Party')).toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/parties', expect.objectContaining({ method: 'POST' }))
    })

    it('joins a party by code', async () => {
      setTier('desktop')
      stubFetch((url: string) => {
        if (url === '/api/parties/join') return { ok: true, json: () => Promise.resolve(party({ id: 'party-joined', name: 'Joined Party' })) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({ pcs: [pc()] }))

      await user.type(screen.getByPlaceholderText('Party code'), 'abcde')
      await user.click(screen.getByRole('button', { name: 'Join' }))

      expect(await screen.findByText('Joined Party')).toBeInTheDocument()
    })

    it('shows a not-found error when the party code is invalid', async () => {
      setTier('desktop')
      stubFetch((url: string) => {
        if (url === '/api/parties/join') return { ok: false, status: 404 }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({ pcs: [pc()] }))

      await user.type(screen.getByPlaceholderText('Party code'), 'ZZZZZ')
      await user.click(screen.getByRole('button', { name: 'Join' }))

      expect(await screen.findByText('No party found with that code')).toBeInTheDocument()
    })

    it('renders party members, removes a member, and adds an available character', async () => {
      setTier('desktop')
      const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
        if (init?.method === 'PUT') return { ok: true, json: () => Promise.resolve(party({ member_pc_ids: [], member_names: {} })) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({
        pcs: [pc({ id: 'pc1', name: 'Aria' }), pc({ id: 'pc2', name: 'Bram' })],
        parties: [party({ member_pc_ids: ['pc1'], member_names: { pc1: 'Aria' } })],
      }))

      expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Remove' }))

      await waitFor(() => {
        const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
        expect(putCall).toBeTruthy()
        expect(JSON.parse((putCall![1] as RequestInit).body as string).member_pc_ids).toEqual([])
      })
    })

    it('saves pooled currency for a party', async () => {
      setTier('desktop')
      const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
        if (init?.method === 'PUT') return { ok: true, json: () => Promise.resolve(party()) }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({ parties: [party()] }))

      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
        expect(putCall).toBeTruthy()
      })
    })
  })

  describe('Player panel: rooms', () => {
    it('lists recent rooms and joins one', async () => {
      setTier('desktop')
      stubFetch()
      const user = userEvent.setup()
      const { onJoinAsPlayer } = renderDashboard(me({
        pcs: [pc({ id: 'pc1', name: 'Aria' })],
        recent_rooms: [{ user_id: 'u1', room_id: 'ROOM1', last_pc_id: 'pc1', last_joined_at: '2026-01-01' }],
      }))

      const goButtons = screen.getAllByRole('button', { name: 'Go' })
      await user.click(goButtons.find(b => !b.hasAttribute('disabled'))!)
      expect(onJoinAsPlayer).toHaveBeenCalledWith('ROOM1', 'pc1')
    })

    it('leaves a recent room after confirming', async () => {
      setTier('desktop')
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
        if (init?.method === 'DELETE') return { ok: true }
        return { ok: true, json: () => Promise.resolve([]) }
      })
      const user = userEvent.setup()
      renderDashboard(me({
        pcs: [pc({ id: 'pc1', name: 'Aria' })],
        recent_rooms: [{ user_id: 'u1', room_id: 'ROOM1', last_pc_id: 'pc1', last_joined_at: '2026-01-01' }],
      }))

      await user.click(screen.getByRole('button', { name: 'Leave' }))

      await waitFor(() => expect(screen.queryByText(/ROOM1/)).not.toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/rooms/ROOM1/membership', { method: 'DELETE' })
    })

    it('joins a new room via code', async () => {
      setTier('desktop')
      stubFetch()
      const user = userEvent.setup()
      const { onJoinAsPlayer } = renderDashboard(me({ pcs: [pc({ id: 'pc1', name: 'Aria' })] }))

      await user.type(screen.getByPlaceholderText('XXXXX'), 'roomx')
      const goButtons = screen.getAllByRole('button', { name: 'Go' })
      await user.click(goButtons[goButtons.length - 1])

      expect(onJoinAsPlayer).toHaveBeenCalledWith('ROOMX', 'pc1')
    })
  })
})
