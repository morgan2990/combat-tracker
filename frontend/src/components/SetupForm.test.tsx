import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SetupForm } from './SetupForm'

describe('SetupForm', () => {
  it('greets the player by name', () => {
    render(<SetupForm myName="Aria" onSubmit={vi.fn()} />)
    expect(screen.getByText('Welcome, Aria!')).toBeInTheDocument()
  })

  it('disables submit until Max HP has a value', async () => {
    const user = userEvent.setup()
    render(<SetupForm myName="Aria" onSubmit={vi.fn()} />)

    const submit = screen.getByRole('button', { name: 'Enter Combat' })
    expect(submit).toBeDisabled()

    await user.type(screen.getByLabelText('Max HP'), '30')
    expect(submit).toBeEnabled()
  })

  it('submits with the parsed HP and initiative, then locks the form', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<SetupForm myName="Aria" onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText('Max HP'), '30')
    await user.type(screen.getByLabelText('Initiative'), '14')
    await user.click(screen.getByRole('button', { name: 'Enter Combat' }))

    expect(onSubmit).toHaveBeenCalledWith({ type: 'setup_character', max_hp: 30, initiative: 14 })
    expect(screen.getByRole('button', { name: 'Joining tracker…' })).toBeDisabled()
    expect(screen.getByLabelText('Max HP')).toBeDisabled()
  })

  it('defaults initiative to 0 when left blank', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<SetupForm myName="Aria" onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText('Max HP'), '30')
    await user.click(screen.getByRole('button', { name: 'Enter Combat' }))

    expect(onSubmit).toHaveBeenCalledWith({ type: 'setup_character', max_hp: 30, initiative: 0 })
  })

  it('does not submit when Max HP is zero', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<SetupForm myName="Aria" onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText('Max HP'), '0')
    await user.click(screen.getByRole('button', { name: 'Enter Combat' }))

    expect(onSubmit).not.toHaveBeenCalled()
  })
})
