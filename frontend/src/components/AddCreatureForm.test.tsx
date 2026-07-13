import { createRef } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AddCreatureForm } from './AddCreatureForm'
import type { AddCreatureFormHandle } from './AddCreatureForm'
import type { CustomMonster } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
})

function customMonster(overrides: Partial<CustomMonster> = {}): CustomMonster {
  return {
    id: 'cm1', name: 'Bandit Captain', edition: '5e', max_hp: 65,
    initiative_modifier: 2, is_custom: true, private: false,
    owner_id: 'owner1', owner_display_name: 'DM', ...overrides,
  }
}

describe('AddCreatureForm', () => {
  it('fetches and shows My Creatures pills only when showMyCreaturesInline', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([customMonster()]) }))
    vi.stubGlobal('fetch', fetchMock)
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline />)

    expect(await screen.findByRole('button', { name: /Bandit Captain/ })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters?edition=5e')
  })

  it('does not fetch My Creatures when showMyCreaturesInline is false', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(screen.queryByText('My Creatures')).not.toBeInTheDocument()
  })

  it('selecting a My Creatures pill fills name, HP, and shows the statblock-ready badge', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([customMonster({ source_type: 'pdf' })]),
    })))
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline />)

    const pill = await screen.findByRole('button', { name: /Bandit Captain/ })
    await user.click(pill)

    expect(screen.getByPlaceholderText('Goblin')).toHaveValue('Bandit Captain')
    expect(screen.getByPlaceholderText('14')).toHaveValue(65)
    expect(screen.getByText('statblock ready')).toBeInTheDocument()
  })

  it('does not search below the minimum character threshold', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Go')
    await new Promise(r => setTimeout(r, 250))

    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('searches after the debounce and shows a dropdown with edition and HP', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }]),
    }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Gob')

    const hit = await screen.findByRole('button', { name: /Goblin/ })
    expect(hit).toHaveTextContent('7 HP')
    expect(fetchMock).toHaveBeenCalledWith('/api/search/monsters?q=Gob&edition=5e')
  })

  it("shows the custom monster's owner on a custom hit", async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([{ id: 'cm1', name: 'Bandit', max_hp: 11, initiative_modifier: null, is_custom: true, owner_display_name: 'Bob' }]),
    })))
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Ban')

    expect(await screen.findByText('by Bob')).toBeInTheDocument()
  })

  it('selecting an official hit fetches /api/monsters/{name} and fills the form', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ source_type: 'url', reference_url: 'https://example.com/goblin', pdf_object_key: '', initiative_modifier: 1 }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Gob')
    const hit = await screen.findByRole('button', { name: /Goblin/ })
    await user.click(hit)

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/monsters/Goblin'))
    expect(screen.getByPlaceholderText('Goblin')).toHaveValue('Goblin')
    expect(screen.getByPlaceholderText('Type 3+ characters…')).toHaveValue('')
  })

  it('selecting a custom hit fetches /api/custom-monsters/{id}', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'cm1', name: 'Bandit', max_hp: 11, initiative_modifier: null, is_custom: true }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ source_type: '', reference_url: '', pdf_object_key: '', initiative_modifier: null }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Ban')
    const hit = await screen.findByRole('button', { name: /Bandit/ })
    await user.click(hit)

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters/cm1'))
  })

  it('selects the first result on Enter', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ source_type: '', reference_url: '', pdf_object_key: '', initiative_modifier: null }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    const searchInput = screen.getByPlaceholderText('Type 3+ characters…')
    await user.type(searchInput, 'Gob')
    await screen.findByRole('button', { name: /Goblin/ })
    await user.type(searchInput, '{Enter}')

    await waitFor(() => expect(screen.getByPlaceholderText('Goblin')).toHaveValue('Goblin'))
  })

  it('clears the resolved monster reference when the name is hand-edited', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([customMonster({ source_type: 'pdf' })]),
    })))
    const user = userEvent.setup()
    render(<AddCreatureForm sendMessage={vi.fn()} edition="5e" showMyCreaturesInline />)

    const pill = await screen.findByRole('button', { name: /Bandit Captain/ })
    await user.click(pill)
    expect(screen.getByText('statblock ready')).toBeInTheDocument()

    await user.type(screen.getByPlaceholderText('Goblin'), 'X')
    expect(screen.queryByText('statblock ready')).not.toBeInTheDocument()
  })

  it('submits add_creature with defaults, and does not submit when invalid', async () => {
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<AddCreatureForm sendMessage={sendMessage} edition="5e" showMyCreaturesInline={false} />)

    await user.click(screen.getByRole('button', { name: 'Add Creature' }))
    expect(sendMessage).not.toHaveBeenCalled()

    await user.type(screen.getByPlaceholderText('Goblin'), 'Orc')
    await user.type(screen.getByPlaceholderText('14'), '15')
    await user.click(screen.getByRole('button', { name: 'Add Creature' }))

    expect(sendMessage).toHaveBeenCalledWith({
      type: 'add_creature', name: 'Orc', max_hp: 15, quantity: 1,
      source_type: '', reference_url: '', pdf_object_key: '', display_name: '',
    })
  })

  it('submits the resolved monster reference, quantity, and trimmed display name, then resets the form', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/monsters')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'm1', name: 'Goblin', max_hp: 7, initiative_modifier: null, is_custom: false }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ source_type: 'url', reference_url: 'https://example.com/goblin', pdf_object_key: '', initiative_modifier: 2 }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<AddCreatureForm sendMessage={sendMessage} edition="5e" showMyCreaturesInline={false} />)

    await user.type(screen.getByPlaceholderText('Type 3+ characters…'), 'Gob')
    const hit = await screen.findByRole('button', { name: /Goblin/ })
    await user.click(hit)
    await waitFor(() => expect(screen.getByText('statblock ready')).toBeInTheDocument())

    await user.clear(screen.getByLabelText('Qty'))
    await user.type(screen.getByLabelText('Qty'), '3')
    await user.type(screen.getByPlaceholderText('e.g. Guard Alpha'), '  Guard Alpha  ')
    await user.click(screen.getByRole('button', { name: 'Add Creature' }))

    expect(sendMessage).toHaveBeenCalledWith({
      type: 'add_creature', name: 'Goblin', max_hp: 7, quantity: 3,
      source_type: 'url', reference_url: 'https://example.com/goblin', pdf_object_key: '',
      display_name: 'Guard Alpha', initiative_modifier: 2,
    })

    expect(screen.getByPlaceholderText('Goblin')).toHaveValue('')
    expect(screen.getByPlaceholderText('14')).toHaveValue(null)
    expect(screen.getByLabelText('Qty')).toHaveValue(1)
  })

  it('exposes selectCustomMonster via ref', async () => {
    const ref = createRef<AddCreatureFormHandle>()
    render(<AddCreatureForm ref={ref} sendMessage={vi.fn()} edition="5e" showMyCreaturesInline={false} />)

    act(() => ref.current!.selectCustomMonster(customMonster({ name: 'Ogre', max_hp: 59 })))

    expect(await screen.findByPlaceholderText('Goblin')).toHaveValue('Ogre')
    expect(screen.getByPlaceholderText('14')).toHaveValue(59)
  })
})
