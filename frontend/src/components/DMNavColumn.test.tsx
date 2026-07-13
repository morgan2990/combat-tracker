import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DMNavColumn } from './DMNavColumn'

afterEach(() => {
  vi.unstubAllGlobals()
})

function stubFetch() {
  const fetchMock = vi.fn((url: string) => {
    if (url.startsWith('/api/encounters')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([{ id: 'enc1', name: 'Goblin Ambush' }]),
      })
    }
    if (url.startsWith('/api/custom-monsters')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([{
          id: 'cm1', name: 'Bandit Captain', edition: '5e', max_hp: 65,
          initiative_modifier: null, is_custom: true, private: false,
          owner_id: 'owner1', owner_display_name: 'DM',
        }]),
      })
    }
    return Promise.resolve({ ok: false, json: () => Promise.resolve(null) })
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('DMNavColumn', () => {
  it('fetches and renders encounters and custom monsters for the given edition', async () => {
    const fetchMock = stubFetch()
    render(<DMNavColumn sendMessage={vi.fn()} edition="5e" onSelectCustomMonster={vi.fn()} />)

    await waitFor(() => expect(screen.getByRole('button', { name: 'Goblin Ambush' })).toBeInTheDocument())
    expect(screen.getByRole('button', { name: /Bandit Captain/ })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/encounters?edition=5e')
    expect(fetchMock).toHaveBeenCalledWith('/api/custom-monsters?edition=5e')
  })

  it('shows the empty-state message when there are no encounters', async () => {
    stubFetch()
    render(<DMNavColumn sendMessage={vi.fn()} edition="5e" onSelectCustomMonster={vi.fn()} />)

    await waitFor(() => expect(screen.getByText('No saved encounters for this edition.')).toBeInTheDocument())
  })

  it('sends an inject_encounter message when an encounter is clicked', async () => {
    stubFetch()
    const sendMessage = vi.fn()
    render(<DMNavColumn sendMessage={sendMessage} edition="5e" onSelectCustomMonster={vi.fn()} />)

    const btn = await screen.findByRole('button', { name: 'Goblin Ambush' })
    await userEvent.click(btn)

    expect(sendMessage).toHaveBeenCalledWith({ type: 'inject_encounter', encounter_id: 'enc1' })
  })

  it('calls onSelectCustomMonster when a custom monster is clicked', async () => {
    stubFetch()
    const onSelectCustomMonster = vi.fn()
    render(<DMNavColumn sendMessage={vi.fn()} edition="5e" onSelectCustomMonster={onSelectCustomMonster} />)

    const btn = await screen.findByRole('button', { name: /Bandit Captain/ })
    await userEvent.click(btn)

    expect(onSelectCustomMonster).toHaveBeenCalledWith(expect.objectContaining({ id: 'cm1', name: 'Bandit Captain' }))
  })
})
