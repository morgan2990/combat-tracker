import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlayerView } from './PlayerView'
import type { Entity, RoomState } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
})

function entity(overrides: Partial<Entity> = {}): Entity {
  return {
    id: 'e1', name: 'Aria', type: 'pc', max_hp: 30, current_hp: 30, temp_hp: 0,
    initiative: 10, shares_initiative: false, conditions: [], dead: false, is_hidden: false,
    ...overrides,
  }
}

function room(overrides: Partial<RoomState> = {}): RoomState {
  return {
    room_id: 'room1', edition: '5e', is_started: true, round: 1, active_index: 0,
    entities: [entity()],
    ...overrides,
  }
}

describe('PlayerView', () => {
  it('calls onBackToDashboard from the back button', async () => {
    const user = userEvent.setup()
    const onBack = vi.fn()
    render(<PlayerView roomState={room()} myEntityId={null} needsInitiative={false} sendMessage={vi.fn()} onBackToDashboard={onBack} />)
    await user.click(screen.getByRole('button', { name: '← Dashboard' }))
    expect(onBack).toHaveBeenCalled()
  })

  it('shows the round number once combat has started', () => {
    render(<PlayerView roomState={room({ round: 3 })} myEntityId={null} needsInitiative={false} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)
    expect(screen.getByText('Round 3')).toBeInTheDocument()
  })

  it('shows the initiative form when needsInitiative, and submits a parsed value', async () => {
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(<PlayerView roomState={room()} myEntityId={null} needsInitiative sendMessage={sendMessage} onBackToDashboard={vi.fn()} />)

    expect(screen.getByText('Set your initiative')).toBeInTheDocument()
    await user.type(screen.getByPlaceholderText('e.g. 14'), '17')
    await user.click(screen.getByRole('button', { name: 'Set' }))

    expect(sendMessage).toHaveBeenCalledWith({ type: 'set_initiative', initiative: 17 })
  })

  it('shows a waiting message when combat has not started and initiative is not needed', () => {
    render(<PlayerView roomState={room({ is_started: false })} myEntityId={null} needsInitiative={false} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)
    expect(screen.getByText('Waiting for DM to start combat…')).toBeInTheDocument()
  })

  it('shows a no-combatants message when there are no entities', () => {
    render(<PlayerView roomState={room({ entities: [] })} myEntityId={null} needsInitiative={false} sendMessage={vi.fn()} onBackToDashboard={vi.fn()} />)
    expect(screen.getByText('No combatants yet.')).toBeInTheDocument()
  })

  it('shows a preparing message pre-combat when creatures exist but are hidden from the visible list', () => {
    render(
      <PlayerView
        roomState={room({ is_started: false, entities: [entity({ type: 'creature', id: 'c1' })] })}
        myEntityId={null}
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByText('The DM is preparing the encounter…')).toBeInTheDocument()
  })

  it('marks the active row and the current player, and shows HP for a pc', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria' })], active_index: 0 })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByText('you')).toBeInTheDocument()
    expect(screen.getByText('30/30 HP')).toBeInTheDocument()
  })

  it('shows a descriptive HP label instead of numbers for a creature', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'c1', type: 'creature', name: 'Goblin', current_hp: 6, max_hp: 20 })] })}
        myEntityId={null}
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByText('Injured')).toBeInTheDocument()
  })

  it('shows a dead badge and condition list on a row', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e2', name: 'Bram', dead: true, conditions: ['Prone', 'Stunned'] })] })}
        myEntityId={null}
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByText('💀 Dead')).toBeInTheDocument()
    expect(screen.getByText('Prone · Stunned')).toBeInTheDocument()
  })

  it("renders my entity's editor panel and a Your turn badge when it's my turn", () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1', name: 'Aria' })], active_index: 0 })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getByText('Your turn!')).toBeInTheDocument()
  })

  it('sends refresh_from_profile when the refresh button is clicked', async () => {
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={sendMessage}
        onBackToDashboard={vi.fn()}
      />,
    )
    await user.click(screen.getByRole('button', { name: '↻ Refresh profile' }))
    expect(sendMessage).toHaveBeenCalledWith({ type: 'refresh_from_profile' })
  })

  it('shows the Inventory button only when my entity has a pc_id, and opens the panel on click', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve({ pc: { name: 'Aria', max_hp: 30, items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 } } }),
    })))
    const user = userEvent.setup()
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1', pc_id: 'pc1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    const invBtn = screen.getByRole('button', { name: /Inventory/ })
    await user.click(invBtn)

    expect(await screen.findByText("Aria's Inventory")).toBeInTheDocument()
  })

  it('links my character sheet in a new tab so the room stays open', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1', pc_id: 'pc1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )

    const link = screen.getByRole('link', { name: /Sheet/ })
    expect(link).toHaveAttribute('href', '/characters/pc1/sheet')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'))
  })

  it('does not link a sheet when my entity has no pc_id', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.queryByRole('link', { name: /Sheet/ })).not.toBeInTheDocument()
  })

  it('does not show the Inventory button when my entity has no pc_id', () => {
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.queryByRole('button', { name: /Inventory/ })).not.toBeInTheDocument()
  })

  it('opens and closes the companion form, wiring onSubmit directly to sendMessage', async () => {
    const user = userEvent.setup()
    const sendMessage = vi.fn()
    render(
      <PlayerView
        roomState={room({ entities: [entity({ id: 'e1' })] })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={sendMessage}
        onBackToDashboard={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '+ Add Summon / Pet' }))
    expect(screen.getByText('Add Summon / Pet')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText('Add Summon / Pet')).not.toBeInTheDocument()
  })

  it('renders owned companions with their own editor and vitals badges', () => {
    render(
      <PlayerView
        roomState={room({
          entities: [
            entity({ id: 'e1' }),
            entity({ id: 'comp1', type: 'companion', name: 'Wolf', owner_id: 'e1', dead: true }),
          ],
        })}
        myEntityId="e1"
        needsInitiative={false}
        sendMessage={vi.fn()}
        onBackToDashboard={vi.fn()}
      />,
    )
    expect(screen.getAllByText('Wolf', { exact: false }).length).toBeGreaterThan(0)
    expect(screen.getByText('companion')).toBeInTheDocument()
  })

  describe('EntityEditor (via myEntity panel)', () => {
    function renderWithEntity(sendMessage = vi.fn(), overrides: Partial<Entity> = {}) {
      render(
        <PlayerView
          roomState={room({ entities: [entity({ id: 'e1', current_hp: 20, max_hp: 30, temp_hp: 5, ...overrides })] })}
          myEntityId="e1"
          needsInitiative={false}
          sendMessage={sendMessage}
          onBackToDashboard={vi.fn()}
        />,
      )
      return sendMessage
    }

    it('applies a negative delta, clamped to 0', async () => {
      const user = userEvent.setup()
      const sendMessage = renderWithEntity(vi.fn(), { current_hp: 3 })
      await user.click(screen.getByRole('button', { name: '-5' }))
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ type: 'update_entity', current_hp: 0 }))
    })

    it('applies a positive delta, clamped to max_hp', async () => {
      const user = userEvent.setup()
      const sendMessage = renderWithEntity(vi.fn(), { current_hp: 28, max_hp: 30 })
      await user.click(screen.getByRole('button', { name: '+5' }))
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ current_hp: 30 }))
    })

    it('shows temp HP when present', () => {
      renderWithEntity(vi.fn(), { temp_hp: 5 })
      expect(screen.getByText('+5 tmp')).toBeInTheDocument()
    })

    it('directly sets HP by typing and pressing Enter', async () => {
      const user = userEvent.setup()
      const sendMessage = renderWithEntity(vi.fn(), { current_hp: 20, max_hp: 30 })

      await user.click(screen.getByRole('button', { name: '20/30' }))
      const input = screen.getByDisplayValue('20')
      await user.clear(input)
      await user.type(input, '25{Enter}')

      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ current_hp: 25 }))
    })

    it('toggles a condition on and off', async () => {
      const user = userEvent.setup()
      const sendMessage = renderWithEntity(vi.fn(), { conditions: [] })

      await user.click(screen.getByRole('button', { name: 'Prone' }))
      expect(sendMessage).toHaveBeenLastCalledWith(expect.objectContaining({ conditions: ['Prone'] }))
    })
  })
})
