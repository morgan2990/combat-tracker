import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DMView } from './DMView'
import type { Entity, RoomState } from '../types'
import { useLayoutTier } from '../hooks/useLayoutTier'
import type { LayoutTier } from '../hooks/useLayoutTier'

vi.mock('../hooks/useLayoutTier')

afterEach(() => {
  vi.unstubAllGlobals()
  vi.mocked(useLayoutTier).mockReset()
})

function setTier(tier: LayoutTier) {
  vi.mocked(useLayoutTier).mockReturnValue(tier)
}

function entity(overrides: Partial<Entity> = {}): Entity {
  return {
    id: 'e1', name: 'Aria', type: 'pc', max_hp: 30, current_hp: 30, temp_hp: 0,
    initiative: 10, shares_initiative: false, conditions: [], dead: false, is_hidden: false,
    ...overrides,
  }
}

function room(overrides: Partial<RoomState> = {}): RoomState {
  return {
    room_id: 'room1', edition: '5e', is_started: false, round: 1, active_index: 0,
    entities: [entity()],
    ...overrides,
  }
}

function stubPartyFetch(members: { pc_id: string; name: string; currency: { pp: number; gp: number; ep: number; sp: number; cp: number } }[] = []) {
  const partyBody = { party_linked: true, items: [], members, party_currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 }, members_currency_total: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }
  // Routes by URL: DMView itself, InventoryPanel, and PartyInventoryPanel all
  // hit .../party-inventory; DMNavColumn (rendered at tablet/desktop) also
  // hits /api/encounters and /api/custom-monsters expecting plain arrays.
  const fetchMock = vi.fn((url: string) => {
    if (url.includes('/party-inventory')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(partyBody) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('DMView', () => {
  it('phone tier: shows the tracker, encounter templates control, and hides the nav/statblock columns', async () => {
    setTier('phone')
    stubPartyFetch()
    render(<DMView roomState={room()} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)

    expect(screen.getByText('⚔ DM Panel')).toBeInTheDocument()
    expect(screen.getByText('Room Code:')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Encounter Templates/ })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('Aria')).toBeInTheDocument())
  })

  it('tablet tier: shows the DM nav column but not the statblock column or encounter templates control', async () => {
    setTier('tablet')
    stubPartyFetch()
    render(<DMView roomState={room()} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)

    expect(screen.getByText('📋 Encounters')).toBeInTheDocument() // DMNavColumn's own heading
    expect(screen.queryByRole('button', { name: /Encounter Templates/ })).not.toBeInTheDocument()
  })

  it('desktop tier: shows the statblock column placeholder', async () => {
    setTier('desktop')
    stubPartyFetch()
    render(<DMView roomState={room()} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)

    expect(await screen.findByText(/Select a creature's/)).toBeInTheDocument()
  })

  it('calls onBackToDashboard', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    const onBack = vi.fn()
    render(<DMView roomState={room()} sendMessage={vi.fn()} onBackToDashboard={onBack} />)

    await user.click(screen.getByRole('button', { name: '← Dashboard' }))
    expect(onBack).toHaveBeenCalled()
  })

  it('disables Start Combat while any pc/companion has no initiative, and lists who is pending', () => {
    setTier('phone')
    stubPartyFetch()
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria', initiative: null })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: '▶ Start Combat' })).toBeDisabled()
    expect(screen.getByText(/Waiting on initiative: Aria/)).toBeInTheDocument()
  })

  it('sends start_combat when everyone has initiative', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<DMView roomState={room({ entities: [entity({ initiative: 10 })] })} sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '▶ Start Combat' }))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'start_combat' })
  })

  it('shows Next Turn/End Combat once started, with a confirm step for ending', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<DMView roomState={room({ is_started: true, round: 2 })} sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)

    expect(screen.getByText('Round 2')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Next Turn →' }))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'next_turn' })

    await user.click(screen.getByRole('button', { name: '⚠ End Combat' }))
    expect(screen.getByText('End this encounter? Creatures will be removed.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText('End this encounter? Creatures will be removed.')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '⚠ End Combat' }))
    await user.click(screen.getByRole('button', { name: 'Yes, End Combat' }))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'end_combat' })
    expect(screen.queryByText('End this encounter? Creatures will be removed.')).not.toBeInTheDocument()
  })

  it('sends add_lair_action', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<DMView roomState={room()} sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '+ Add Lair Action' }))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'add_lair_action' })
  })

  it('shows Remove All Dead Creatures only when a creature is dead, and sends remove_dead_creatures', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    const { rerender } = render(<DMView roomState={room()} sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)
    expect(screen.queryByText(/Remove All Dead Creatures/)).not.toBeInTheDocument()

    rerender(<DMView roomState={room({ entities: [entity({ id: 'c1', type: 'creature', dead: true })] })} sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)
    await user.click(screen.getByText(/Remove All Dead Creatures/))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'remove_dead_creatures' })
  })

  it('shows a no-combatants message when there are no entities', () => {
    setTier('phone')
    stubPartyFetch()
    render(<DMView roomState={room({ entities: [] })} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)
    expect(screen.getByText(/No combatants yet/)).toBeInTheDocument()
  })

  it('opens the Party Inventory panel from the header button', async () => {
    setTier('phone')
    stubPartyFetch()
    const user = userEvent.setup()
    render(<DMView roomState={room()} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '🎒 Party Inventory' }))

    expect(await screen.findByText('Party Inventory')).toBeInTheDocument()
  })

  it("opens a linked party member's inventory from their EntityRow", async () => {
    setTier('phone')
    stubPartyFetch([{ pc_id: 'pc1', name: 'Aria', currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }])
    const user = userEvent.setup()
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria', pc_id: 'pc1' })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    const invBtn = await screen.findByTitle('View inventory')
    await user.click(invBtn)

    expect(await screen.findByText("Aria's Inventory")).toBeInTheDocument()
  })

  it("links a linked party member's sheet through the room, in a new tab", async () => {
    setTier('phone')
    stubPartyFetch([{ pc_id: 'pc1', name: 'Aria', currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } }])
    const state = room({ entities: [entity({ id: 'e1', name: 'Aria', pc_id: 'pc1' })] })
    render(<DMView roomState={state} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)

    const link = await screen.findByTitle('View character sheet')
    expect(link).toHaveAttribute('href', `/characters/pc1/sheet?room=${encodeURIComponent(state.room_id)}`)
    expect(link).toHaveAttribute('target', '_blank')
  })

  it("does not link a sheet for a pc not linked to the room's party", async () => {
    setTier('phone')
    stubPartyFetch([])
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria', pc_id: 'pc1' })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    await waitFor(() => expect(screen.getByText('Aria')).toBeInTheDocument())
    expect(screen.queryByTitle('View character sheet')).not.toBeInTheDocument()
  })

  it("does not offer inventory access for a pc not linked to the room's party", async () => {
    setTier('phone')
    stubPartyFetch([]) // no members linked
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria', pc_id: 'pc1' })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    await waitFor(() => expect(screen.getByText('Aria')).toBeInTheDocument())
    expect(screen.queryByTitle('View inventory')).not.toBeInTheDocument()
  })

  it('opens a statblock drawer for a sourced creature at the tablet tier', async () => {
    setTier('tablet')
    stubPartyFetch()
    const user = userEvent.setup()
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'c1', type: 'creature', name: 'Goblin', source_type: 'url', reference_url: 'https://example.com/goblin' })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    const statblockBtn = await screen.findByTitle('View statblock')
    await user.click(statblockBtn)

    expect(await screen.findByTitle('Goblin statblock')).toBeInTheDocument()
  })

  it('updates the desktop statblock column when a sourced creature is selected', async () => {
    setTier('desktop')
    stubPartyFetch()
    const user = userEvent.setup()
    render(
      <DMView
        roomState={room({ entities: [entity({ id: 'c1', type: 'creature', name: 'Goblin', source_type: 'url', reference_url: 'https://example.com/goblin' })] })}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    const statblockBtn = await screen.findByTitle('View statblock')
    await user.click(statblockBtn)

    expect(await screen.findByTitle('Goblin statblock')).toBeInTheDocument()
    expect(screen.queryByText(/Select a creature's/)).not.toBeInTheDocument()
  })
})
