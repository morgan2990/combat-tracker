import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PartyInventoryPanel } from './PartyInventoryPanel'
import type { PartyInventoryResponse } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
})

function partyResponse(overrides: Partial<PartyInventoryResponse> = {}): PartyInventoryResponse {
  return {
    party_linked: true,
    items: [
      { owner_pc_id: 'pc1', owner_name: 'Aria', name: 'Shield', quantity: 1 },
      { owner_pc_id: 'pc2', owner_name: 'Bram', name: 'Axe', quantity: 2 },
    ],
    members: [
      { pc_id: 'pc1', name: 'Aria', currency: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 } },
      { pc_id: 'pc2', name: 'Bram', currency: { pp: 0, gp: 3, ep: 0, sp: 0, cp: 0 } },
    ],
    party_currency: { pp: 0, gp: 100, ep: 0, sp: 0, cp: 0 },
    members_currency_total: { pp: 0, gp: 8, ep: 0, sp: 0, cp: 0 },
    ...overrides,
  }
}

function stubFetchOk(body: unknown) {
  const fetchMock = vi.fn(() => Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('PartyInventoryPanel', () => {
  it('shows a loading state, then fetches the room-scoped endpoint', async () => {
    const fetchMock = stubFetchOk(partyResponse())
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)

    expect(screen.getByText('Loading…')).toBeInTheDocument()
    await screen.findByText('Aria', { selector: 'div' })
    expect(fetchMock).toHaveBeenCalledWith('/api/rooms/room1/party-inventory')
  })

  it('shows the not-linked message when no party is linked', async () => {
    stubFetchOk(partyResponse({ party_linked: false }))
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    expect(await screen.findByText(/No party is linked to this room yet/)).toBeInTheDocument()
  })

  it('shows a permission error for a 403 load failure', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, status: 403, json: () => Promise.resolve({}) })))
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    expect(await screen.findByText(/don't have permission to view this room's party inventory/)).toBeInTheDocument()
  })

  it('shows a generic error for other load failures', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, status: 500, json: () => Promise.resolve({}) })))
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    expect(await screen.findByText('Failed to load party inventory.')).toBeInTheDocument()
  })

  it('renders every member with their items and the party/members currency totals', async () => {
    stubFetchOk(partyResponse())
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    await screen.findByText('Aria', { selector: 'div' })

    expect(screen.getByText('Bram', { selector: 'div' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('Shield')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Axe')).toBeInTheDocument()
    expect(screen.getByText('Party Purse')).toBeInTheDocument()
    expect(screen.getByText("Members' Currency (total)")).toBeInTheDocument()
  })

  it('filters to a single member via the select and updates the currency label', async () => {
    const user = userEvent.setup()
    stubFetchOk(partyResponse())
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    await screen.findByText('Aria', { selector: 'div' })

    await user.selectOptions(screen.getByRole('combobox'), 'pc1')

    expect(screen.getByText("Aria's Currency")).toBeInTheDocument()
    expect(screen.getByDisplayValue('Shield')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('Axe')).not.toBeInTheDocument()
    expect(screen.queryByText('Bram', { selector: 'div' })).not.toBeInTheDocument()
  })

  it('adds and removes item rows for a specific member without affecting others', async () => {
    const user = userEvent.setup()
    stubFetchOk(partyResponse())
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    await screen.findByText('Aria', { selector: 'div' })

    const addButtons = screen.getAllByRole('button', { name: '+ Add Item' })
    await user.click(addButtons[0]) // Aria's

    const nameInputs = screen.getAllByPlaceholderText('Item name')
    expect(nameInputs).toHaveLength(3) // Aria: Shield + new blank, Bram: Axe

    await user.click(screen.getAllByRole('button', { name: 'Remove' })[0]) // remove Aria's Shield
    expect(screen.queryByDisplayValue('Shield')).not.toBeInTheDocument()
    expect(screen.getByDisplayValue('Axe')).toBeInTheDocument()
  })

  it('saves a specific member via PUT with only that member\'s items/currency', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}) })
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(partyResponse()) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    await screen.findByText('Aria', { selector: 'div' })

    await user.click(screen.getByRole('button', { name: 'Save Aria' }))

    await waitFor(() => {
      const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
      expect(putCall?.[0]).toBe('/api/rooms/room1/party-inventory/pcs/pc1')
      const body = JSON.parse((putCall![1] as RequestInit).body as string)
      expect(body).toEqual({ items: [{ name: 'Shield', quantity: 1 }], currency: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 } })
    })
  })

  it('shows a save error on a failed PUT', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve({ ok: false, status: 500, json: () => Promise.resolve({}) })
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(partyResponse()) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    await screen.findByText('Aria', { selector: 'div' })

    await user.click(screen.getByRole('button', { name: 'Save Aria' }))

    expect(await screen.findByText('Failed to save (500)')).toBeInTheDocument()
  })

  it('shows a no-members message when the party has no members', async () => {
    stubFetchOk(partyResponse({ members: [], items: [] }))
    render(<PartyInventoryPanel roomId="room1" onClose={vi.fn()} />)
    expect(await screen.findByText('No members in this party.')).toBeInTheDocument()
  })

  it('calls onClose on backdrop click, Escape, and the close button', async () => {
    const user = userEvent.setup()
    stubFetchOk(partyResponse())
    const onClose = vi.fn()
    const { container } = render(<PartyInventoryPanel roomId="room1" onClose={onClose} />)
    await screen.findByText('Aria', { selector: 'div' })

    await user.click(screen.getByText('Party Inventory'))
    expect(onClose).not.toHaveBeenCalled()

    fireEvent.keyDown(container.firstChild as Element, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: '✕' }))
    expect(onClose).toHaveBeenCalledTimes(2)

    await user.click(container.firstChild as Element)
    expect(onClose).toHaveBeenCalledTimes(3)
  })
})
