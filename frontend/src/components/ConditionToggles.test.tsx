import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConditionToggles } from './ConditionToggles'
import { CONDITIONS } from '../entityVitals'

describe('ConditionToggles', () => {
  it('renders one button per known condition', () => {
    render(<ConditionToggles conditions={[]} onToggle={vi.fn()} />)
    for (const cond of CONDITIONS) {
      expect(screen.getByRole('button', { name: cond })).toBeInTheDocument()
    }
  })

  it('calls onToggle with the clicked condition', async () => {
    const onToggle = vi.fn()
    render(<ConditionToggles conditions={[]} onToggle={onToggle} />)

    await userEvent.click(screen.getByRole('button', { name: 'Prone' }))

    expect(onToggle).toHaveBeenCalledWith('Prone')
  })

  it('marks active conditions distinctly from inactive ones', () => {
    render(<ConditionToggles conditions={['Stunned']} onToggle={vi.fn()} />)

    const active = screen.getByRole('button', { name: 'Stunned' })
    const inactive = screen.getByRole('button', { name: 'Prone' })
    expect(active.style.color).not.toBe(inactive.style.color)
  })
})
