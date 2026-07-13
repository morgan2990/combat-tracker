import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CustomMonsterList } from './CustomMonsterList'
import type { CustomMonster } from '../types'

function monster(overrides: Partial<CustomMonster> = {}): CustomMonster {
  return {
    id: 'cm1', name: 'Goblin Boss', edition: '5e', max_hp: 21,
    initiative_modifier: null, is_custom: true, private: false,
    owner_id: 'owner1', owner_display_name: 'DM', ...overrides,
  }
}

describe('CustomMonsterList', () => {
  it('shows an empty message when there are no monsters', () => {
    render(<CustomMonsterList monsters={[]} onSelect={vi.fn()} />)
    expect(screen.getByText('No custom monsters for this edition.')).toBeInTheDocument()
  })

  it('renders a button per monster with its name and HP', () => {
    render(<CustomMonsterList monsters={[monster()]} onSelect={vi.fn()} />)
    expect(screen.queryByText('No custom monsters for this edition.')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Goblin Boss/ })).toHaveTextContent('21 HP')
  })

  it('calls onSelect with the clicked monster', async () => {
    const onSelect = vi.fn()
    const m = monster()
    render(<CustomMonsterList monsters={[m]} onSelect={onSelect} />)

    await userEvent.click(screen.getByRole('button', { name: /Goblin Boss/ }))

    expect(onSelect).toHaveBeenCalledWith(m)
  })
})
