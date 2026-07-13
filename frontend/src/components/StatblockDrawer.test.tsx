import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StatblockDrawer } from './StatblockDrawer'
import type { Entity } from '../types'

function entity(overrides: Partial<Entity> = {}): Entity {
  return {
    id: 'e1', name: 'Goblin', type: 'creature', max_hp: 7, current_hp: 7, temp_hp: 0,
    initiative: null, shares_initiative: false, conditions: [], dead: false, is_hidden: false,
    ...overrides,
  }
}

describe('StatblockDrawer', () => {
  it('renders the entity name and an iframe for a url-sourced statblock', () => {
    render(<StatblockDrawer entity={entity({ source_type: 'url', reference_url: 'https://example.com/goblin' })} open onClose={vi.fn()} />)
    expect(screen.getByText('Goblin')).toBeInTheDocument()
    expect(screen.getByTitle('Goblin statblock')).toHaveAttribute('src', 'https://example.com/goblin')
  })

  it('renders an embed for a pdf-sourced statblock only while open', () => {
    const { container, rerender } = render(<StatblockDrawer entity={entity({ name: 'Orc', source_type: 'pdf' })} open={false} onClose={vi.fn()} />)
    expect(container.querySelector('embed')).not.toBeInTheDocument()

    rerender(<StatblockDrawer entity={entity({ name: 'Orc', source_type: 'pdf' })} open onClose={vi.fn()} />)
    const embed = container.querySelector('embed')
    expect(embed).toHaveAttribute('src', '/api/monsters/Orc/pdf')
  })

  it('calls onClose when the close button is clicked', async () => {
    const onClose = vi.fn()
    render(<StatblockDrawer entity={entity()} open onClose={onClose} />)

    await userEvent.click(screen.getByRole('button'))

    expect(onClose).toHaveBeenCalled()
  })
})
