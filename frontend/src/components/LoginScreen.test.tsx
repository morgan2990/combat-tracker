import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginScreen } from './LoginScreen'

afterEach(() => {
  vi.unstubAllGlobals()
})

function stubFetch(status: number, ok = status === 200) {
  const fetchMock = vi.fn(() => Promise.resolve({ ok, status, json: () => Promise.resolve({}) }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function submitButton() {
  return document.querySelector('form button[type="submit"]') as HTMLButtonElement
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByPlaceholderText('e.g. aria'), 'aria')
  await user.type(screen.getByPlaceholderText('••••••••'), 'secret123')
  await user.click(submitButton())
}

describe('LoginScreen', () => {
  it('disables submit until both fields are filled', async () => {
    const user = userEvent.setup()
    render(<LoginScreen onAuthed={vi.fn()} />)

    const submit = submitButton()
    expect(submit).toBeDisabled()

    await user.type(screen.getByPlaceholderText('e.g. aria'), 'aria')
    expect(submit).toBeDisabled()

    await user.type(screen.getByPlaceholderText('••••••••'), 'secret123')
    expect(submit).toBeEnabled()
  })

  it('posts to /api/login and calls onAuthed on success', async () => {
    const fetchMock = stubFetch(200)
    const onAuthed = vi.fn()
    render(<LoginScreen onAuthed={onAuthed} />)

    await fillAndSubmit(userEvent.setup())

    await waitFor(() => expect(onAuthed).toHaveBeenCalled())
    expect(fetchMock).toHaveBeenCalledWith('/api/login', expect.objectContaining({ method: 'POST' }))
  })

  it('switches to signup mode and posts to /api/signup', async () => {
    const fetchMock = stubFetch(200)
    const user = userEvent.setup()
    render(<LoginScreen onAuthed={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'Create Account' }))
    await user.type(screen.getByPlaceholderText('e.g. aria'), 'aria')
    await user.type(screen.getByPlaceholderText('••••••••'), 'secret123')
    await user.click(screen.getAllByRole('button', { name: 'Create Account' })[1])

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/signup', expect.anything()))
  })

  it('shows a specific error for 401', async () => {
    stubFetch(401)
    render(<LoginScreen onAuthed={vi.fn()} />)
    await fillAndSubmit(userEvent.setup())
    expect(await screen.findByText('Wrong username or passphrase.')).toBeInTheDocument()
  })

  it('shows a specific error for 409', async () => {
    stubFetch(409)
    render(<LoginScreen onAuthed={vi.fn()} />)
    await fillAndSubmit(userEvent.setup())
    expect(await screen.findByText('That username is already taken.')).toBeInTheDocument()
  })

  it('shows a generic error for other failure statuses', async () => {
    stubFetch(500)
    render(<LoginScreen onAuthed={vi.fn()} />)
    await fillAndSubmit(userEvent.setup())
    expect(await screen.findByText('Something went wrong. Please try again.')).toBeInTheDocument()
  })

  it('shows a service-unavailable error when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('network down'))))
    render(<LoginScreen onAuthed={vi.fn()} />)
    await fillAndSubmit(userEvent.setup())
    expect(await screen.findByText('Service unavailable. Please try again.')).toBeInTheDocument()
  })

  it('clears the error when switching modes', async () => {
    stubFetch(401)
    const user = userEvent.setup()
    render(<LoginScreen onAuthed={vi.fn()} />)
    await fillAndSubmit(user)
    expect(await screen.findByText('Wrong username or passphrase.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Create Account' }))

    expect(screen.queryByText('Wrong username or passphrase.')).not.toBeInTheDocument()
  })
})
