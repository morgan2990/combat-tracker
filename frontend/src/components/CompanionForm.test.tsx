import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CompanionForm } from './CompanionForm'

describe('CompanionForm', () => {
  it('submits with the entered name, HP, and initiative, then closes', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    const onClose = vi.fn()
    render(<CompanionForm onSubmit={onSubmit} onClose={onClose} />)

    await user.type(screen.getByPlaceholderText('e.g. Wolf'), 'Wolf')
    await user.type(screen.getByPlaceholderText('e.g. 18'), '18')
    await user.type(screen.getByPlaceholderText('leave blank if sharing'), '12')
    await user.click(screen.getByRole('button', { name: 'Add to Tracker' }))

    expect(onSubmit).toHaveBeenCalledWith({
      type: 'add_companion', name: 'Wolf', max_hp: 18, shares_initiative: false, initiative: 12,
    })
    expect(onClose).toHaveBeenCalled()
  })

  it('submits null initiative when left blank', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<CompanionForm onSubmit={onSubmit} onClose={vi.fn()} />)

    await user.type(screen.getByPlaceholderText('e.g. Wolf'), 'Wolf')
    await user.type(screen.getByPlaceholderText('e.g. 18'), '18')
    await user.click(screen.getByRole('button', { name: 'Add to Tracker' }))

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ initiative: null }))
  })

  it('clears and disables initiative when sharing is checked', async () => {
    const user = userEvent.setup()
    render(<CompanionForm onSubmit={vi.fn()} onClose={vi.fn()} />)

    const initiativeInput = screen.getByPlaceholderText('leave blank if sharing')
    await user.type(initiativeInput, '12')
    await user.click(screen.getByRole('checkbox', { name: /Shares initiative/ }))

    expect(initiativeInput).toBeDisabled()
    expect(initiativeInput).toHaveValue(null)
  })

  it('submits shares_initiative true and null initiative when sharing', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<CompanionForm onSubmit={onSubmit} onClose={vi.fn()} />)

    await user.type(screen.getByPlaceholderText('e.g. Wolf'), 'Wolf')
    await user.type(screen.getByPlaceholderText('e.g. 18'), '18')
    await user.click(screen.getByRole('checkbox', { name: /Shares initiative/ }))
    await user.click(screen.getByRole('button', { name: 'Add to Tracker' }))

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ shares_initiative: true, initiative: null }))
  })

  it('does not submit when name is blank', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<CompanionForm onSubmit={onSubmit} onClose={vi.fn()} />)

    await user.type(screen.getByPlaceholderText('e.g. 18'), '18')
    await user.click(screen.getByRole('button', { name: 'Add to Tracker' }))

    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('calls onClose without submitting when Cancel is clicked', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    const onClose = vi.fn()
    render(<CompanionForm onSubmit={onSubmit} onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onClose).toHaveBeenCalled()
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
