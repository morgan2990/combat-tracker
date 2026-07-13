import { afterEach, describe, expect, it, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ItemNameSearchField } from './ItemNameSearchField'
import type { Item } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
})

function item(overrides: Partial<Item> = {}): Item {
  return { name: '', quantity: 1, ...overrides }
}

// ItemNameSearchField is a controlled component: its search query is derived
// from item.name, so exercising the debounced search requires actually
// feeding onUpdate's patches back in, like a real parent (InventoryPanel/
// PartyInventoryPanel) would — a fixed item prop would leave query frozen.
function ControlledField({ initial, onUpdate, onSelectExisting = () => false }: {
  initial: Item
  onUpdate: (patch: Partial<Item>) => void
  onSelectExisting?: (catalogItemId: string) => boolean
}) {
  const [current, setCurrent] = useState(initial)
  return (
    <ItemNameSearchField
      item={current}
      edition="5e"
      onUpdate={patch => { setCurrent(prev => ({ ...prev, ...patch })); onUpdate(patch) }}
      onSelectExisting={onSelectExisting}
    />
  )
}

describe('ItemNameSearchField', () => {
  it('clears catalog linkage fields when the name is edited by hand', async () => {
    const user = userEvent.setup()
    const onUpdate = vi.fn()
    render(
      <ItemNameSearchField
        item={item({ name: 'Sword' })}
        edition="5e"
        onUpdate={onUpdate}
        onSelectExisting={() => false}
      />,
    )

    await user.type(screen.getByPlaceholderText('Item name'), 'X')

    expect(onUpdate).toHaveBeenCalledWith({ name: 'SwordX', catalog_item_id: undefined, reference_url: undefined })
  })

  it('shows a link icon once the row is catalog-backed', () => {
    render(
      <ItemNameSearchField
        item={item({ name: 'Longsword', reference_url: 'https://example.com/longsword' })}
        edition="5e"
        onUpdate={vi.fn()}
        onSelectExisting={() => false}
      />,
    )
    expect(screen.getByTitle('Open 5etools reference')).toHaveAttribute('href', 'https://example.com/longsword')
  })

  it('searches and shows a dropdown, then resolves the selection into onUpdate', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/items')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'i1', name: 'Longsword', category: 'Weapon', rarity: 'Common' }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 'i1', name: 'Longsword', edition: '5e', category: 'Weapon', reference_url: 'https://example.com/longsword' }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const onUpdate = vi.fn()
    render(<ControlledField initial={item()} onUpdate={onUpdate} />)

    await user.type(screen.getByPlaceholderText('Item name'), 'Long')

    const hit = await screen.findByRole('button', { name: /Longsword/ })
    await user.click(hit)

    await waitFor(() => expect(onUpdate).toHaveBeenLastCalledWith({
      name: 'Longsword', catalog_item_id: 'i1', reference_url: 'https://example.com/longsword',
    }))
  })

  it('defers to onSelectExisting and clears the field when it claims the selection', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'i1', name: 'Longsword', category: 'Weapon' }]) }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const onUpdate = vi.fn()
    render(<ControlledField initial={item()} onUpdate={onUpdate} onSelectExisting={() => true} />)

    await user.type(screen.getByPlaceholderText('Item name'), 'Long')
    const hit = await screen.findByRole('button', { name: /Longsword/ })
    await user.click(hit)

    expect(onUpdate).toHaveBeenLastCalledWith({ name: '' })
    // The follow-up /api/items/{id} resolve fetch should never fire.
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining('/api/items/'))
  })

  it('selects the first result on Enter when the dropdown is open', async () => {
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/search/items')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'i1', name: 'Longsword', category: 'Weapon' }]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 'i1', name: 'Longsword', edition: '5e', category: 'Weapon', reference_url: '' }) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const onUpdate = vi.fn()
    render(<ControlledField initial={item()} onUpdate={onUpdate} />)

    const input = screen.getByPlaceholderText('Item name')
    await user.type(input, 'Long')
    await screen.findByRole('button', { name: /Longsword/ })
    await user.type(input, '{Enter}')

    await waitFor(() => expect(onUpdate).toHaveBeenLastCalledWith({
      name: 'Longsword', catalog_item_id: 'i1', reference_url: undefined,
    }))
  })
})
