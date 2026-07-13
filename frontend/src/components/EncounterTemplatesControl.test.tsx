import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EncounterTemplatesControl } from './EncounterTemplatesControl'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('EncounterTemplatesControl', () => {
  it('does not fetch until opened', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<EncounterTemplatesControl sendMessage={vi.fn()} edition="5e" />)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('fetches and lists encounters for the edition when opened', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([{ id: 'enc1', name: 'Goblin Ambush', owner_id: 'o1', edition: '5e', monsters: [] }]),
    }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<EncounterTemplatesControl sendMessage={vi.fn()} edition="5e" />)

    await user.click(screen.getByRole('button', { name: /Encounter Templates/ }))

    expect(await screen.findByRole('button', { name: 'Goblin Ambush' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/encounters?edition=5e')
  })

  it('shows the empty-state message when the fetch resolves with no encounters', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([]) })))
    const user = userEvent.setup()
    render(<EncounterTemplatesControl sendMessage={vi.fn()} edition="5e" />)

    await user.click(screen.getByRole('button', { name: /Encounter Templates/ }))

    expect(await screen.findByText('No saved encounters for this edition.')).toBeInTheDocument()
  })

  it('treats a failed fetch as an empty list', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, json: () => Promise.resolve(null) })))
    const user = userEvent.setup()
    render(<EncounterTemplatesControl sendMessage={vi.fn()} edition="5e" />)

    await user.click(screen.getByRole('button', { name: /Encounter Templates/ }))

    expect(await screen.findByText('No saved encounters for this edition.')).toBeInTheDocument()
  })

  it('does not refetch on a second open', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([]) }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<EncounterTemplatesControl sendMessage={vi.fn()} edition="5e" />)

    const toggle = screen.getByRole('button', { name: /Encounter Templates/ })
    await user.click(toggle)
    await screen.findByText('No saved encounters for this edition.')
    await user.click(toggle) // close
    await user.click(toggle) // reopen

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('injects the encounter and closes the dropdown on click', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([{ id: 'enc1', name: 'Goblin Ambush', owner_id: 'o1', edition: '5e', monsters: [] }]),
    })))
    const sendMessage = vi.fn()
    const user = userEvent.setup()
    render(<EncounterTemplatesControl sendMessage={sendMessage} edition="5e" />)

    await user.click(screen.getByRole('button', { name: /Encounter Templates/ }))
    const encounterBtn = await screen.findByRole('button', { name: 'Goblin Ambush' })
    await user.click(encounterBtn)

    expect(sendMessage).toHaveBeenCalledWith({ type: 'inject_encounter', encounter_id: 'enc1' })
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Goblin Ambush' })).not.toBeInTheDocument())
  })
})
