import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { InventoryPanel } from './InventoryPanel'
import type { PartyInventoryResponse } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
})

function ownerResponse(overrides: Partial<{ name: string; max_hp: number; items: unknown[]; currency: unknown }> = {}) {
  return {
    pc: {
      name: 'Aria', max_hp: 30, items: [{ name: 'Longsword', quantity: 1 }], currency: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 },
      ...overrides,
    },
  }
}

function partyResponse(): PartyInventoryResponse {
  return {
    party_linked: true,
    items: [
      { owner_pc_id: 'pc1', owner_name: 'Aria', name: 'Shield', quantity: 1 },
      { owner_pc_id: 'pc2', owner_name: 'Bram', name: 'Axe', quantity: 1 },
    ],
    members: [
      { pc_id: 'pc1', name: 'Aria', currency: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 } },
      { pc_id: 'pc2', name: 'Bram', currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } },
    ],
    party_currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
    members_currency_total: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 },
  }
}

function stubFetchOk(body: unknown) {
  const fetchMock = vi.fn(() => Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('InventoryPanel', () => {
  it('shows a loading state, then the owner PC name and item once loaded', async () => {
    stubFetchOk(ownerResponse())
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)

    expect(screen.getByText('Loading…')).toBeInTheDocument()

    expect(await screen.findByText("Aria's Inventory")).toBeInTheDocument()
    expect(screen.getByDisplayValue('Longsword')).toBeInTheDocument()
  })

  it('fetches the owner endpoint when no roomId is given', async () => {
    const fetchMock = stubFetchOk(ownerResponse())
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)
    await screen.findByText("Aria's Inventory")
    expect(fetchMock).toHaveBeenCalledWith('/api/pcs/pc1')
  })

  it('fetches the room-scoped party-inventory endpoint and filters items to this pcId when roomId is given', async () => {
    const fetchMock = stubFetchOk(partyResponse())
    render(<InventoryPanel pcId="pc1" roomId="room1" onClose={vi.fn()} />)

    expect(await screen.findByText("Aria's Inventory")).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/rooms/room1/party-inventory')
    expect(screen.getByDisplayValue('Shield')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('Axe')).not.toBeInTheDocument()
  })

  it('shows a permission error for a 403/404 load failure', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, status: 403, json: () => Promise.resolve({}) })))
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)
    expect(await screen.findByText("You don't have permission to view this character's inventory.")).toBeInTheDocument()
  })

  it('shows a generic error for other load failures', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, status: 500, json: () => Promise.resolve({}) })))
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)
    expect(await screen.findByText('Failed to load inventory.')).toBeInTheDocument()
  })

  it('shows a no-items message and adds a blank row on + Add Item', async () => {
    const user = userEvent.setup()
    stubFetchOk(ownerResponse({ items: [] }))
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)
    expect(await screen.findByText('No items yet.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '+ Add Item' }))

    expect(screen.getByPlaceholderText('Item name')).toBeInTheDocument()
    expect(screen.queryByText('No items yet.')).not.toBeInTheDocument()
  })

  it('removes a row on Remove click', async () => {
    const user = userEvent.setup()
    stubFetchOk(ownerResponse())
    render(<InventoryPanel pcId="pc1" onClose={vi.fn()} />)
    await screen.findByDisplayValue('Longsword')

    await user.click(screen.getByRole('button', { name: 'Remove' }))

    expect(screen.queryByDisplayValue('Longsword')).not.toBeInTheDocument()
    expect(screen.getByText('No items yet.')).toBeInTheDocument()
  })

  it('saves via PUT to the owner endpoint with name/max_hp/items/currency and closes on success', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}) })
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(ownerResponse()) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<InventoryPanel pcId="pc1" onClose={onClose} />)
    await screen.findByDisplayValue('Longsword')

    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')!
    expect(putCall[0]).toBe('/api/pcs/pc1')
    const body = JSON.parse((putCall[1] as RequestInit).body as string)
    expect(body).toEqual({ name: 'Aria', max_hp: 30, items: [{ name: 'Longsword', quantity: 1 }], currency: { pp: 0, gp: 5, ep: 0, sp: 0, cp: 0 } })
  })

  it('saves via PUT to the room-scoped endpoint with just items/currency when roomId is given', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}) })
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(partyResponse()) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<InventoryPanel pcId="pc1" roomId="room1" onClose={vi.fn()} />)
    await screen.findByDisplayValue('Shield')

    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
      expect(putCall?.[0]).toBe('/api/rooms/room1/party-inventory/pcs/pc1')
    })
  })

  it('shows a save error and does not close when the PUT fails', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve({ ok: false, status: 500, json: () => Promise.resolve({}) })
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(ownerResponse()) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<InventoryPanel pcId="pc1" onClose={onClose} />)
    await screen.findByDisplayValue('Longsword')

    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Failed to save (500)')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('calls onClose on backdrop click but not on clicks inside the panel', async () => {
    const user = userEvent.setup()
    stubFetchOk(ownerResponse())
    const onClose = vi.fn()
    const { container } = render(<InventoryPanel pcId="pc1" onClose={onClose} />)
    await screen.findByText("Aria's Inventory")

    await user.click(screen.getByText("Aria's Inventory"))
    expect(onClose).not.toHaveBeenCalled()

    await user.click(container.firstChild as Element)
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onClose on Escape from the backdrop', async () => {
    stubFetchOk(ownerResponse())
    const onClose = vi.fn()
    const { container } = render(<InventoryPanel pcId="pc1" onClose={onClose} />)
    await screen.findByText("Aria's Inventory")

    fireEvent.keyDown(container.firstChild as Element, { key: 'Escape' })

    expect(onClose).toHaveBeenCalled()
  })

  it('calls onClose on the close (✕) button', async () => {
    const user = userEvent.setup()
    stubFetchOk(ownerResponse())
    const onClose = vi.fn()
    render(<InventoryPanel pcId="pc1" onClose={onClose} />)
    await screen.findByText("Aria's Inventory")

    await user.click(screen.getByRole('button', { name: '✕' }))

    expect(onClose).toHaveBeenCalled()
  })
})
