import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EditionToggle } from './EditionToggle'

describe('EditionToggle', () => {
  it('renders a button for each edition', () => {
    render(<EditionToggle edition="5e" onChange={vi.fn()} />)
    expect(screen.getByRole('button', { name: '5e' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '5.5e' })).toBeInTheDocument()
  })

  it('calls onChange with the clicked edition', async () => {
    const onChange = vi.fn()
    render(<EditionToggle edition="5e" onChange={onChange} />)

    await userEvent.click(screen.getByRole('button', { name: '5.5e' }))

    expect(onChange).toHaveBeenCalledWith('5.5e')
  })

  it('clicking the already-active edition still reports it', async () => {
    const onChange = vi.fn()
    render(<EditionToggle edition="5e" onChange={onChange} />)

    await userEvent.click(screen.getByRole('button', { name: '5e' }))

    expect(onChange).toHaveBeenCalledWith('5e')
  })
})
