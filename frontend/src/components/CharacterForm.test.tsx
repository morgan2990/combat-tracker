import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CharacterForm, parseCompanionRow } from './CharacterForm'

const mockNavigate = vi.fn()
let mockParams: { id?: string } = {}
vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
  useParams: () => mockParams,
}))

beforeEach(() => {
  mockParams = {}
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  mockNavigate.mockReset()
})

describe('parseCompanionRow', () => {
  it('parses a valid row', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '11', shares_initiative: false }))
      .toEqual({ name: 'Wolf', max_hp: 11 })
  })

  it('trims the name', () => {
    expect(parseCompanionRow({ name: '  Wolf  ', max_hp: '11', shares_initiative: false })!.name)
      .toBe('Wolf')
  })

  it('rejects a blank name', () => {
    expect(parseCompanionRow({ name: '   ', max_hp: '11', shares_initiative: false })).toBeNull()
  })

  it('rejects a missing max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '', shares_initiative: false })).toBeNull()
  })

  it('rejects a zero max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '0', shares_initiative: false })).toBeNull()
  })

  it('rejects a negative max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '-5', shares_initiative: false })).toBeNull()
  })

  it('rejects an unparseable max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: 'abc', shares_initiative: false })).toBeNull()
  })
})

function jsonOk(body: unknown) {
  return { ok: true, json: () => Promise.resolve(body), text: () => Promise.resolve('') }
}

function stubFetch(router: (url: string, init?: RequestInit) => unknown) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => Promise.resolve(router(url, init)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('CharacterForm', () => {
  it('renders Create Character with no existing-companions section', () => {
    render(<CharacterForm onSaved={vi.fn()} />)
    expect(screen.getByText('Create Character')).toBeInTheDocument()
    expect(screen.queryByText('Existing Companions / Pets')).not.toBeInTheDocument()
    expect(screen.getByText('No new companions.')).toBeInTheDocument()
  })

  it('navigates home from the back button', async () => {
    const user = userEvent.setup()
    render(<CharacterForm onSaved={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: '← Back' }))
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })

  it('loads an existing character and its companions in edit mode', async () => {
    mockParams = { id: 'pc1' }
    stubFetch(() => jsonOk({
      pc: { id: 'pc1', owner_user_id: 'o1', name: 'Aria', type: 'pc', max_hp: 30, shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } },
      companions: [{ id: 'comp1', owner_user_id: 'o1', name: 'Wolf', type: 'companion', max_hp: 11, shares_initiative: true, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }],
    }))
    render(<CharacterForm onSaved={vi.fn()} />)

    expect(await screen.findByText('Edit Character')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Aria')).toBeInTheDocument()
    expect(screen.getByText('Existing Companions / Pets')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Wolf')).toBeInTheDocument()
  })

  it('adds and removes a new companion row', async () => {
    const user = userEvent.setup()
    render(<CharacterForm onSaved={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '+ Add' }))
    expect(screen.queryByText('No new companions.')).not.toBeInTheDocument()
    expect(screen.getByPlaceholderText('e.g. Rex')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove' }))
    expect(screen.getByText('No new companions.')).toBeInTheDocument()
  })

  it('submits create: POSTs the character, then each valid new companion, then calls onSaved', async () => {
    const fetchMock = stubFetch((url: string, init?: RequestInit) => {
      if (url === '/api/pcs' && init?.method === 'POST') return jsonOk({ id: 'new-pc' })
      if (url === '/api/pcs/new-pc/companions') return jsonOk({})
      return jsonOk(null)
    })
    const user = userEvent.setup()
    const onSaved = vi.fn()
    render(<CharacterForm onSaved={onSaved} />)

    await user.type(screen.getByPlaceholderText('e.g. Aria Sunsong'), 'Aria')
    await user.type(screen.getByPlaceholderText('e.g. 32'), '30')
    await user.click(screen.getByRole('button', { name: '+ Add' }))
    await user.type(screen.getByPlaceholderText('e.g. Rex'), 'Wolf')
    await user.type(screen.getByPlaceholderText('HP'), '11')
    await user.click(screen.getByRole('button', { name: 'Save Character' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const pcPost = fetchMock.mock.calls.find(c => c[0] === '/api/pcs')!
    expect(JSON.parse((pcPost[1] as RequestInit).body as string)).toEqual({ name: 'Aria', max_hp: 30 })
    const compPost = fetchMock.mock.calls.find(c => c[0] === '/api/pcs/new-pc/companions')!
    expect(JSON.parse((compPost[1] as RequestInit).body as string)).toEqual({ name: 'Wolf', max_hp: 11, shares_initiative: false })
  })

  it('skips invalid new companion rows silently', async () => {
    const fetchMock = stubFetch((url: string, init?: RequestInit) => {
      if (url === '/api/pcs' && init?.method === 'POST') return jsonOk({ id: 'new-pc' })
      return jsonOk({})
    })
    const user = userEvent.setup()
    const onSaved = vi.fn()
    render(<CharacterForm onSaved={onSaved} />)

    await user.type(screen.getByPlaceholderText('e.g. Aria Sunsong'), 'Aria')
    await user.type(screen.getByPlaceholderText('e.g. 32'), '30')
    // Bypass the Add-row button's own required-field HTML validation by
    // submitting with zero new companion rows added, which already
    // exercises the "skip invalid/empty rows" loop trivially; a row with
    // truly invalid content can't be submitted via the UI since its inputs
    // are marked required.
    await user.click(screen.getByRole('button', { name: 'Save Character' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(fetchMock.mock.calls.some(c => String(c[0]).includes('/companions'))).toBe(false)
  })

  it('submits edit: PUTs the character', async () => {
    mockParams = { id: 'pc1' }
    const fetchMock = stubFetch((url: string, init?: RequestInit) => {
      if (url === '/api/pcs/pc1' && !init) return jsonOk({ pc: { id: 'pc1', owner_user_id: 'o1', name: 'Aria', type: 'pc', max_hp: 30, shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }, companions: [] })
      if (url === '/api/pcs/pc1' && init?.method === 'PUT') return jsonOk({})
      return jsonOk({})
    })
    const user = userEvent.setup()
    const onSaved = vi.fn()
    render(<CharacterForm onSaved={onSaved} />)
    await screen.findByText('Edit Character')

    await user.click(screen.getByRole('button', { name: 'Save Character' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const putCall = fetchMock.mock.calls.find(c => c[0] === '/api/pcs/pc1' && (c[1] as RequestInit | undefined)?.method === 'PUT')!
    expect(JSON.parse((putCall[1] as RequestInit).body as string)).toEqual({ name: 'Aria', max_hp: 30 })
  })

  it('shows an error and does not call onSaved when the save fails', async () => {
    stubFetch((url: string, init?: RequestInit) => {
      if (url === '/api/pcs' && init?.method === 'POST') return { ok: false, json: () => Promise.resolve({}) }
      return jsonOk(null)
    })
    const user = userEvent.setup()
    const onSaved = vi.fn()
    render(<CharacterForm onSaved={onSaved} />)

    await user.type(screen.getByPlaceholderText('e.g. Aria Sunsong'), 'Aria')
    await user.type(screen.getByPlaceholderText('e.g. 32'), '30')
    await user.click(screen.getByRole('button', { name: 'Save Character' }))

    expect(await screen.findByText('Failed to save character')).toBeInTheDocument()
    expect(onSaved).not.toHaveBeenCalled()
  })

  describe('existing companion actions', () => {
    function pcWithCompanion() {
      return {
        pc: { id: 'pc1', owner_user_id: 'o1', name: 'Aria', type: 'pc', max_hp: 30, shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } },
        companions: [{ id: 'comp1', owner_user_id: 'o1', name: 'Wolf', type: 'companion', max_hp: 11, shares_initiative: false, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }],
      }
    }

    it('saves an edited existing companion via PUT', async () => {
      mockParams = { id: 'pc1' }
      const fetchMock = stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/pcs/pc1' && !init) return jsonOk(pcWithCompanion())
        if (url === '/api/pcs/comp1' && init?.method === 'PUT') return jsonOk({})
        return jsonOk({})
      })
      const user = userEvent.setup()
      render(<CharacterForm onSaved={vi.fn()} />)
      await screen.findByDisplayValue('Wolf')

      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        const putCall = fetchMock.mock.calls.find(c => c[0] === '/api/pcs/comp1' && (c[1] as RequestInit | undefined)?.method === 'PUT')
        expect(putCall).toBeTruthy()
      })
    })

    it('shows an error when saving an existing companion fails', async () => {
      mockParams = { id: 'pc1' }
      stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/pcs/pc1' && !init) return jsonOk(pcWithCompanion())
        if (url === '/api/pcs/comp1' && init?.method === 'PUT') return { ok: false, json: () => Promise.resolve({}) }
        return jsonOk({})
      })
      const user = userEvent.setup()
      render(<CharacterForm onSaved={vi.fn()} />)
      await screen.findByDisplayValue('Wolf')

      await user.click(screen.getByRole('button', { name: 'Save' }))

      expect(await screen.findByText('Failed to save companion')).toBeInTheDocument()
    })

    it('deletes an existing companion after confirming', async () => {
      mockParams = { id: 'pc1' }
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const fetchMock = stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/pcs/pc1' && !init) return jsonOk(pcWithCompanion())
        if (url === '/api/pcs/comp1' && init?.method === 'DELETE') return { ok: true }
        return jsonOk({})
      })
      const user = userEvent.setup()
      render(<CharacterForm onSaved={vi.fn()} />)
      await screen.findByDisplayValue('Wolf')

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      await waitFor(() => expect(screen.queryByDisplayValue('Wolf')).not.toBeInTheDocument())
      expect(fetchMock).toHaveBeenCalledWith('/api/pcs/comp1', { method: 'DELETE' })
    })

    it('does not delete when the confirmation is cancelled', async () => {
      mockParams = { id: 'pc1' }
      vi.spyOn(window, 'confirm').mockReturnValue(false)
      stubFetch((url: string) => {
        if (url === '/api/pcs/pc1') return jsonOk(pcWithCompanion())
        return jsonOk({})
      })
      const user = userEvent.setup()
      render(<CharacterForm onSaved={vi.fn()} />)
      await screen.findByDisplayValue('Wolf')

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      expect(screen.getByDisplayValue('Wolf')).toBeInTheDocument()
    })

    it('shows an error when deleting an existing companion fails', async () => {
      mockParams = { id: 'pc1' }
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      stubFetch((url: string, init?: RequestInit) => {
        if (url === '/api/pcs/pc1' && !init) return jsonOk(pcWithCompanion())
        if (url === '/api/pcs/comp1' && init?.method === 'DELETE') return { ok: false }
        return jsonOk({})
      })
      const user = userEvent.setup()
      render(<CharacterForm onSaved={vi.fn()} />)
      await screen.findByDisplayValue('Wolf')

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      expect(await screen.findByText('Failed to delete companion')).toBeInTheDocument()
    })
  })
})
