import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EntityRow, parseHP } from './EntityRow'
import type { Entity } from '../types'

describe('parseHP', () => {
  it('applies a positive delta to the current HP', () => {
    expect(parseHP('+5', 10, 20)).toBe(15)
  })

  it('applies a negative delta to the current HP', () => {
    expect(parseHP('-4', 10, 20)).toBe(6)
  })

  it('treats a plain number as an absolute value, not a delta', () => {
    expect(parseHP('7', 10, 20)).toBe(7)
  })

  it('clamps the result to the max HP', () => {
    expect(parseHP('+100', 10, 20)).toBe(20)
  })

  it('clamps the result to a minimum of 0', () => {
    expect(parseHP('-100', 10, 20)).toBe(0)
  })

  it('falls back to current HP when the delta is unparseable', () => {
    expect(parseHP('+abc', 10, 20)).toBe(10)
    expect(parseHP('-abc', 10, 20)).toBe(10)
  })

  it('falls back to current HP when the absolute value is unparseable', () => {
    expect(parseHP('abc', 10, 20)).toBe(10)
  })

  it('trims surrounding whitespace before parsing', () => {
    expect(parseHP('  +5  ', 10, 20)).toBe(15)
  })
})

function entity(overrides: Partial<Entity> = {}): Entity {
  return {
    id: 'e1', name: 'Goblin', type: 'creature', max_hp: 20, current_hp: 20, temp_hp: 0,
    initiative: 12, shares_initiative: false, conditions: [], dead: false, is_hidden: false,
    ...overrides,
  }
}

function renderRow(overrides: Partial<Entity> = {}, props: Partial<Parameters<typeof EntityRow>[0]> = {}) {
  const sendMessage = vi.fn()
  const onOpenInventory = vi.fn()
  render(
    <EntityRow
      entity={entity(overrides)}
      isActive={false}
      sendMessage={sendMessage}
      onOpenInventory={onOpenInventory}
      {...props}
    />,
  )
  return { sendMessage, onOpenInventory }
}

describe('EntityRow', () => {
  it('renders name, HP, initiative, and type', () => {
    renderRow({ name: 'Goblin', current_hp: 15, max_hp: 20, initiative: 12 })
    expect(screen.getByText('Goblin')).toBeInTheDocument()
    expect(screen.getByText('15/20 HP')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText('creature')).toBeInTheDocument()
  })

  it('shows "--" for null initiative', () => {
    renderRow({ initiative: null })
    expect(screen.getByText('--')).toBeInTheDocument()
  })

  it('renders the display_name alongside the underlying name', () => {
    renderRow({ name: 'Goblin', display_name: 'Sneaky Pete' })
    expect(screen.getByText('Sneaky Pete (Goblin)')).toBeInTheDocument()
  })

  it('shows the active marker when isActive', () => {
    renderRow({}, { isActive: true })
    expect(screen.getByText('▶')).toBeInTheDocument()
  })

  it('shows dead and unconscious badges', () => {
    const { unmount } = render(<EntityRow entity={entity({ dead: true })} isActive={false} sendMessage={vi.fn()} onOpenInventory={vi.fn()} />)
    expect(screen.getByText('💀 Dead')).toBeInTheDocument()
    unmount()

    render(<EntityRow entity={entity({ current_hp: 0, dead: false })} isActive={false} sendMessage={vi.fn()} onOpenInventory={vi.fn()} />)
    expect(screen.getByText('😵 Unconscious')).toBeInTheDocument()
  })

  it('lists active conditions', () => {
    renderRow({ conditions: ['Prone', 'Stunned'] })
    expect(screen.getByText('Prone · Stunned')).toBeInTheDocument()
  })

  it('shows temp HP as a bonus', () => {
    renderRow({ temp_hp: 5 })
    expect(screen.getByText('+5')).toBeInTheDocument()
  })

  it('hides the HP block for a lair action', () => {
    renderRow({ type: 'lair_action' })
    expect(screen.queryByText(/\/\d+ HP/)).not.toBeInTheDocument()
  })

  it('toggles the expanded panel on row click, and again to collapse', async () => {
    const user = userEvent.setup()
    renderRow()
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Goblin/ }))
    expect(screen.getByText('Initiative')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Goblin/ }))
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument()
  })

  it('toggles the expanded panel on Enter/Space keydown', async () => {
    const user = userEvent.setup()
    renderRow()
    const row = screen.getByRole('button', { name: /Goblin/ })
    row.focus()

    await user.keyboard('{Enter}')
    expect(screen.getByText('Initiative')).toBeInTheDocument()

    await user.keyboard(' ')
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument()
  })

  it('shows the inventory button only for a pc with canOpenInventory, and calls onOpenInventory without expanding the row', async () => {
    const user = userEvent.setup()
    const { onOpenInventory } = renderRow({ type: 'pc', pc_id: 'pc1' }, { canOpenInventory: true })

    const invBtn = screen.getByTitle('View inventory')
    await user.click(invBtn)

    expect(onOpenInventory).toHaveBeenCalledWith('pc1')
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument() // row did not expand
  })

  it('does not show the inventory button when canOpenInventory is false', () => {
    renderRow({ type: 'pc', pc_id: 'pc1' }, { canOpenInventory: false })
    expect(screen.queryByTitle('View inventory')).not.toBeInTheDocument()
  })

  it('shows the statblock button only for a sourced creature with onStatblock, and calls it without expanding', async () => {
    const user = userEvent.setup()
    const onStatblock = vi.fn()
    renderRow({ type: 'creature', source_type: 'pdf' }, { onStatblock })

    const btn = screen.getByTitle('View statblock')
    await user.click(btn)

    expect(onStatblock).toHaveBeenCalled()
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument()
  })

  it('toggles visibility for creatures without expanding the row', async () => {
    const user = userEvent.setup()
    const { sendMessage } = renderRow({ type: 'creature', is_hidden: false, id: 'c1' })

    await user.click(screen.getByTitle('Visible to players — click to hide'))

    expect(sendMessage).toHaveBeenCalledWith({ type: 'toggle_entity_visibility', entity_id: 'c1' })
    expect(screen.queryByText('Initiative')).not.toBeInTheDocument()
  })

  it('shows the tooltip when initiative_roll and initiative_modifier are set', () => {
    renderRow({ initiative: 15, initiative_roll: 12, initiative_modifier: 3 })
    expect(screen.getByText('15').closest('div')).toHaveAttribute('title', 'd20: 12 +3 = 15')
  })

  describe('expanded edit panel', () => {
    async function expand(user: ReturnType<typeof userEvent.setup>) {
      await user.click(screen.getByRole('button', { name: /Goblin/ }))
    }

    it('applies an HP change and clears the input', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ current_hp: 10, max_hp: 20 })
      await expand(user)

      const hpInput = screen.getByPlaceholderText('10')
      await user.type(hpInput, '+5')
      await user.click(screen.getByRole('button', { name: 'Apply' }))

      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ type: 'dm_update_entity', current_hp: 15 }))
      expect(hpInput).toHaveValue('')
    })

    it('does nothing when applying a blank HP input', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow()
      await expand(user)

      await user.click(screen.getByRole('button', { name: 'Apply' }))
      expect(sendMessage).not.toHaveBeenCalled()
    })

    it('sets initiative on Enter, ignoring an unchanged or unparseable value', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ initiative: 12 })
      await expand(user)

      const initInput = screen.getByDisplayValue('12')
      await user.clear(initInput)
      await user.type(initInput, '18{Enter}')
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ initiative: 18 }))

      sendMessage.mockClear()
      await user.clear(initInput)
      await user.type(initInput, '12{Enter}')
      expect(sendMessage).not.toHaveBeenCalled()
    })

    it('sets a creature name, ignoring blank input', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ type: 'creature', name: 'Goblin' })
      await expand(user)

      const nameInput = screen.getByDisplayValue('Goblin')
      await user.clear(nameInput)
      await user.type(nameInput, 'Goblin Boss')
      await user.click(screen.getAllByRole('button', { name: 'Set' })[1]) // Name's Set button

      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ name: 'Goblin Boss' }))
    })

    it('does not show name/alias fields for a pc', async () => {
      const user = userEvent.setup()
      renderRow({ type: 'pc' })
      await expand(user)
      expect(screen.queryByText('Name')).not.toBeInTheDocument()
      expect(screen.queryByText('Alias')).not.toBeInTheDocument()
    })

    it('sets an alias', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ type: 'creature' })
      await expand(user)

      const aliasInput = screen.getByPlaceholderText('none')
      await user.type(aliasInput, 'Boss')
      await user.click(screen.getAllByRole('button', { name: 'Set' })[2]) // Alias's Set button

      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ display_name: 'Boss' }))
    })

    it('toggles a condition from the panel', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ conditions: [] })
      await expand(user)

      await user.click(screen.getByRole('button', { name: 'Prone' }))
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ conditions: ['Prone'] }))
    })

    it('kills and revives via the action button', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ dead: false })
      await expand(user)

      await user.click(screen.getByRole('button', { name: 'Kill' }))
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ dead: true, current_hp: 0 }))
    })

    it('shows Revive for an already-dead entity', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ dead: true })
      await expand(user)

      await user.click(screen.getByRole('button', { name: 'Revive' }))
      expect(sendMessage).toHaveBeenCalledWith(expect.objectContaining({ dead: false }))
    })

    it('removes the entity', async () => {
      const user = userEvent.setup()
      const { sendMessage } = renderRow({ id: 'e9' })
      await expand(user)

      await user.click(screen.getByRole('button', { name: 'Remove' }))
      expect(sendMessage).toHaveBeenCalledWith({ type: 'remove_entity', entity_id: 'e9' })
    })

    it('hides the Kill/Revive button for a lair action', async () => {
      const user = userEvent.setup()
      renderRow({ type: 'lair_action' })
      await expand(user)
      expect(screen.queryByRole('button', { name: 'Kill' })).not.toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument()
    })
  })
})
