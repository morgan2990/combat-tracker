import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CustomMonsterPillList } from './CustomMonsterPillList'
import type { CustomMonster } from '../types'

function monster(overrides: Partial<CustomMonster> = {}): CustomMonster {
  return {
    id: 'cm1', name: 'Goblin Boss', edition: '5e', max_hp: 21,
    initiative_modifier: null, is_custom: true, private: false,
    owner_id: 'owner1', owner_display_name: 'DM', ...overrides,
  }
}

describe('CustomMonsterPillList', () => {
  it('renders no pills for an empty list', () => {
    render(<CustomMonsterPillList monsters={[]} onSelect={vi.fn()} />)
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('renders a pill per monster with its name and HP', () => {
    render(<CustomMonsterPillList monsters={[monster(), monster({ id: 'cm2', name: 'Orc', max_hp: 15 })]} onSelect={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Goblin Boss/ })).toHaveTextContent('21 HP')
    expect(screen.getByRole('button', { name: /Orc/ })).toHaveTextContent('15 HP')
  })

  it('calls onSelect with the clicked monster', async () => {
    const onSelect = vi.fn()
    const m = monster()
    render(<CustomMonsterPillList monsters={[m]} onSelect={onSelect} />)

    await userEvent.click(screen.getByRole('button', { name: /Goblin Boss/ }))

    expect(onSelect).toHaveBeenCalledWith(m)
  })
})
