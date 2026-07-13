import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StatblockColumn } from './StatblockColumn'
import type { Entity } from '../types'

function entity(overrides: Partial<Entity> = {}): Entity {
  return {
    id: 'e1', name: 'Goblin', type: 'creature', max_hp: 7, current_hp: 7, temp_hp: 0,
    initiative: null, shares_initiative: false, conditions: [], dead: false, is_hidden: false,
    ...overrides,
  }
}

describe('StatblockColumn', () => {
  it('shows a placeholder when no entity is selected', () => {
    render(<StatblockColumn entity={null} onClose={vi.fn()} />)
    expect(screen.getByText(/Select a creature/)).toBeInTheDocument()
  })

  it('renders the entity name and an iframe for a url-sourced statblock', () => {
    render(<StatblockColumn entity={entity({ source_type: 'url', reference_url: 'https://example.com/goblin' })} onClose={vi.fn()} />)
    expect(screen.getByText('Goblin')).toBeInTheDocument()
    expect(screen.getByTitle('Goblin statblock')).toHaveAttribute('src', 'https://example.com/goblin')
  })

  it('renders an embed for a pdf-sourced statblock', () => {
    const { container } = render(<StatblockColumn entity={entity({ name: 'Orc', source_type: 'pdf' })} onClose={vi.fn()} />)
    const embed = container.querySelector('embed')
    expect(embed).toHaveAttribute('src', '/api/monsters/Orc/pdf')
    expect(embed).toHaveAttribute('type', 'application/pdf')
  })

  it('calls onClose when the close button is clicked', async () => {
    const onClose = vi.fn()
    render(<StatblockColumn entity={entity()} onClose={onClose} />)

    await userEvent.click(screen.getByRole('button'))

    expect(onClose).toHaveBeenCalled()
  })
})
