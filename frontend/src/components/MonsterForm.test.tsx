import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MonsterForm } from './MonsterForm'

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
  mockNavigate.mockReset()
})

function jsonOk(body: unknown) {
  return { ok: true, status: 200, json: () => Promise.resolve(body), text: () => Promise.resolve('') }
}

function stubFetch(router: (url: string, init?: RequestInit) => unknown) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => Promise.resolve(router(url, init)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('MonsterForm', () => {
  it('renders Register Monster in create mode', () => {
    stubFetch(() => jsonOk(null))
    render(<MonsterForm />)
    expect(screen.getByText('Register Monster')).toBeInTheDocument()
  })

  it('loads an existing custom monster in edit mode', async () => {
    mockParams = { id: 'cm1' }
    stubFetch(() => jsonOk({
      id: 'cm1', name: 'Bandit Captain', edition: '5.5e', max_hp: 65,
      initiative_modifier: 2, is_custom: true, private: true,
      owner_id: 'o1', owner_display_name: 'DM', source_type: 'url', reference_url: 'https://example.com/bandit',
    }))
    render(<MonsterForm />)

    expect(await screen.findByText('Edit Monster')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Bandit Captain')).toBeInTheDocument()
    expect(screen.getByDisplayValue(65)).toBeInTheDocument()
    expect(screen.getByDisplayValue(2)).toBeInTheDocument()
    expect(screen.getByRole('checkbox')).toBeChecked()
    expect(screen.getByDisplayValue('https://example.com/bandit')).toBeInTheDocument()
  })

  it('does not offer the PDF source option when editing', async () => {
    mockParams = { id: 'cm1' }
    stubFetch(() => jsonOk({ id: 'cm1', name: 'X', edition: '5e', max_hp: 1, initiative_modifier: null, is_custom: true, private: false, owner_id: 'o1', owner_display_name: 'DM' }))
    render(<MonsterForm />)
    await screen.findByText('Edit Monster')

    expect(screen.queryByRole('button', { name: 'PDF' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'URL' })).toBeInTheDocument()
  })

  it('shows the URL field only when the url source is selected', async () => {
    const user = userEvent.setup()
    stubFetch(() => jsonOk(null))
    render(<MonsterForm />)

    expect(screen.queryByPlaceholderText('https://...')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'URL' }))
    expect(screen.getByPlaceholderText('https://...')).toBeInTheDocument()
  })

  it('shows the file input only when the pdf source is selected (create mode)', async () => {
    const user = userEvent.setup()
    stubFetch(() => jsonOk(null))
    render(<MonsterForm />)

    await user.click(screen.getByRole('button', { name: 'PDF' }))
    expect(screen.getByText('PDF File (max 20 MB)')).toBeInTheDocument()
  })

  it('submits a JSON POST for a non-pdf creation and shows the saved screen', async () => {
    const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return jsonOk({})
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    expect(await screen.findByText('Monster saved!')).toBeInTheDocument()
    const postCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'POST')!
    expect(postCall[0]).toBe('/api/monsters')
    const body = JSON.parse((postCall[1] as RequestInit).body as string)
    expect(body).toEqual({ name: 'Owlbear', max_hp: 59, edition: '5e', private: false })
  })

  it('includes source_type/reference_url and initiative_modifier when set', async () => {
    const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return jsonOk({})
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.type(screen.getByPlaceholderText('e.g. 2 or -1'), '3')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'URL' }))
    await user.type(screen.getByPlaceholderText('https://...'), 'https://example.com/owlbear')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    await screen.findByText('Monster saved!')
    const postCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'POST')!
    const body = JSON.parse((postCall[1] as RequestInit).body as string)
    expect(body).toEqual({
      name: 'Owlbear', max_hp: 59, edition: '5e', private: true,
      source_type: 'url', reference_url: 'https://example.com/owlbear', initiative_modifier: 3,
    })
  })

  it('submits multipart form data for a pdf creation', async () => {
    const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return jsonOk({})
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'PDF' }))
    const file = new File(['%PDF-1.4'], 'owlbear.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    await user.upload(fileInput, file)
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    await screen.findByText('Monster saved!')
    const postCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'POST')!
    expect(postCall[0]).toBe('/api/monsters')
    const form = postCall[1]!.body as FormData
    expect(form.get('name')).toBe('Owlbear')
    expect(form.get('source_type')).toBe('pdf')
    expect((form.get('pdf') as File).name).toBe('owlbear.pdf')
  })

  it('submits a PUT when editing, then navigates home', async () => {
    mockParams = { id: 'cm1' }
    const fetchMock = stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return jsonOk({})
      return jsonOk({ id: 'cm1', name: 'X', edition: '5e', max_hp: 1, initiative_modifier: null, is_custom: true, private: false, owner_id: 'o1', owner_display_name: 'DM' })
    })
    const user = userEvent.setup()
    render(<MonsterForm />)
    await screen.findByText('Edit Monster')

    await user.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
    const putCall = fetchMock.mock.calls.find(c => (c[1] as RequestInit | undefined)?.method === 'PUT')
    expect(putCall?.[0]).toBe('/api/custom-monsters/cm1')
  })

  it('shows a specific error for a 413 (PDF too large)', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return { ok: false, status: 413, text: () => Promise.resolve('') }
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    expect(await screen.findByText('PDF file is too large (max 20 MB).')).toBeInTheDocument()
  })

  it('shows the server error text on other failures', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return { ok: false, status: 500, text: () => Promise.resolve('boom') }
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    expect(await screen.findByText('Error: boom')).toBeInTheDocument()
  })

  it('shows a network error message when fetch throws', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') throw new Error('down')
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))

    expect(await screen.findByText('Network error. Is the server running?')).toBeInTheDocument()
  })

  it('Add Another resets the form back to create mode; Back to Dashboard navigates', async () => {
    stubFetch((_url: string, init?: RequestInit) => {
      if (init?.method === 'POST') return jsonOk({})
      return jsonOk(null)
    })
    const user = userEvent.setup()
    render(<MonsterForm />)

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))
    await screen.findByText('Monster saved!')

    await user.click(screen.getByRole('button', { name: 'Add Another' }))
    expect(screen.getByText('Register Monster')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Goblin')).toHaveValue('')

    await user.type(screen.getByPlaceholderText('Goblin'), 'Owlbear')
    await user.type(screen.getByPlaceholderText('7'), '59')
    await user.click(screen.getByRole('button', { name: 'Save Monster' }))
    await screen.findByText('Monster saved!')

    await user.click(screen.getByRole('button', { name: 'Back to Dashboard' }))
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })
})
