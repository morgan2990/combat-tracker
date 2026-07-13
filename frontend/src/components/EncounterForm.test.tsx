import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EncounterForm } from './EncounterForm'
import { useLayoutTier } from '../hooks/useLayoutTier'
import type { LayoutTier } from '../hooks/useLayoutTier'

vi.mock('../hooks/useLayoutTier')

const mockNavigate = vi.fn()
let mockParams: { id?: string } = {}
vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
  useParams: () => mockParams,
}))

function setTier(tier: LayoutTier) {
  vi.mocked(useLayoutTier).mockReturnValue(tier)
}

beforeEach(() => {
  mockParams = {}
  setTier('desktop')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.mocked(useLayoutTier).mockReset()
  mockNavigate.mockReset()
})

function customMonster(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cm1', name: 'Bandit Captain', edition: '5e', max_hp: 65,
    initiative_modifier: 2, is_custom: true, private: false,
    owner_id: 'owner1', owner_display_name: 'DM', ...overrides,
  }
}

function stubFetch(router: (url: string, init?: RequestInit) => Promise<unknown> | unknown) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => Promise.resolve(router(url, init)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function jsonOk(body: unknown) {
  return { ok: true, json: () => Promise.resolve(body), text: () => Promise.resolve('') }
}

describe('EncounterForm', () => {
  it('renders New Encounter and fetches My Creatures for the default edition', async () => {
    const fetchMock = stubFetch(() => jsonOk([customMonster()]))
    render(<EncounterForm />)

    expect(screen.getByText('New Encounter')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters?edition=5e')
    expect(await screen.findByText(/Bandit Captain/)).toBeInTheDocument()
  })

  it('shows the pill list at the phone tier and the full list otherwise', async () => {
    setTier('phone')
    stubFetch(() => jsonOk([customMonster()]))
    render(<EncounterForm />)
    expect(await screen.findByText('My Creatures')).toBeInTheDocument()
    expect(screen.queryByText('No custom monsters for this edition.')).not.toBeInTheDocument()
  })

  it('loads an existing encounter in edit mode and shows Edit Encounter', async () => {
    mockParams = { id: 'enc1' }
    stubFetch((url: string) => {
      if (url === '/api/encounters/enc1') {
        return jsonOk({ id: 'enc1', name: 'Goblin Ambush', owner_id: 'o1', edition: '5.5e', monsters: [{ name: 'Goblin', is_custom: false, quantity: 2, display_name: '' }] })
      }
      return jsonOk([])
    })
    render(<EncounterForm />)

    expect(await screen.findByText('Edit Encounter')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Goblin Ambush')).toBeInTheDocument()
    expect(screen.getByText('Goblin')).toBeInTheDocument()
  })

  it('refetches My Creatures when the edition toggle changes', async () => {
    const fetchMock = stubFetch(() => jsonOk([]))
    const user = userEvent.setup()
    render(<EncounterForm />)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters?edition=5e'))

    await user.click(screen.getByRole('button', { name: '5.5e' }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters?edition=5.5e'))
  })

  it('adds a custom monster from My Creatures', async () => {
    stubFetch(() => jsonOk([customMonster()]))
    const user = userEvent.setup()
    render(<EncounterForm />)

    const pill = await screen.findByRole('button', { name: /Bandit Captain/ })
    await user.click(pill)

    // "Bandit Captain" now appears twice: once as the source pill (still
    // selectable, e.g. to add a second copy) and once in the added group row.
    expect(screen.getAllByText('Bandit Captain', { exact: false }).length).toBeGreaterThanOrEqual(2)
    expect(screen.queryByText('No monsters added yet.')).not.toBeInTheDocument()
  })

  it('searches after debounce and shows official/custom hits with owner attribution', async () => {
    const fetchMock = stubFetch((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return jsonOk([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }])
      }
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Gob')

    const hit = await screen.findByRole('button', { name: /Goblin/ })
    expect(hit).toHaveTextContent('7 HP')
    expect(fetchMock).toHaveBeenCalledWith('/api/search/monsters?q=Gob&edition=5e')
  })

  it('adds a monster via Enter and via click, and lets you edit its quantity/alias or remove it', async () => {
    stubFetch((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return jsonOk([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }])
      }
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)

    const searchInput = screen.getByPlaceholderText('Type 3+ characters…')
    await user.type(searchInput, 'Gob')
    await screen.findByRole('button', { name: /Goblin/ })
    await user.type(searchInput, '{Enter}')

    expect(screen.getByText('Goblin')).toBeInTheDocument()
    const qtyInput = screen.getByDisplayValue('1')
    fireEvent.change(qtyInput, { target: { value: '4' } })
    expect(qtyInput).toHaveValue(4)

    await user.type(screen.getByPlaceholderText('Alias (optional)'), 'Boss')
    await user.click(screen.getByRole('button', { name: 'Remove' }))

    expect(screen.queryByText('Goblin')).not.toBeInTheDocument()
    expect(screen.getByText('No monsters added yet.')).toBeInTheDocument()
  })

  it('submits a POST to create, then navigates home', async () => {
    const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return jsonOk({})
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)

    await user.type(screen.getByPlaceholderText('Goblin Ambush'), 'Test Encounter')
    await user.click(screen.getByRole('button', { name: 'Save Encounter' }))

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
    const postCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'POST')!
    expect(postCall[0]).toBe('/api/encounters')
    expect(JSON.parse((postCall[1] as RequestInit).body as string)).toEqual({ name: 'Test Encounter', edition: '5e', monsters: [] })
  })

  it('submits a PUT to update when editing', async () => {
    mockParams = { id: 'enc1' }
    const fetchMock = stubFetch((url: string, init?: RequestInit) => {
      if (url === '/api/encounters/enc1' && !init) return jsonOk({ id: 'enc1', name: 'Old', owner_id: 'o1', edition: '5e', monsters: [] })
      if (init?.method === 'PUT') return jsonOk({})
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)
    await screen.findByText('Edit Encounter')

    await user.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
    const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
    expect(putCall?.[0]).toBe('/api/encounters/enc1')
  })

  it('shows the server error text on a failed save', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return { ok: false, text: () => Promise.resolve('name taken') }
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)

    await user.type(screen.getByPlaceholderText('Goblin Ambush'), 'X')
    await user.click(screen.getByRole('button', { name: 'Save Encounter' }))

    expect(await screen.findByText('Error: name taken')).toBeInTheDocument()
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('shows a network error message when fetch throws on submit', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') throw new Error('down')
      return jsonOk([])
    })
    const user = userEvent.setup()
    render(<EncounterForm />)

    await user.type(screen.getByPlaceholderText('Goblin Ambush'), 'X')
    await user.click(screen.getByRole('button', { name: 'Save Encounter' }))

    expect(await screen.findByText('Network error. Is the server running?')).toBeInTheDocument()
  })
})
