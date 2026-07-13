import { afterEach, describe, expect, it, vi } from 'vitest'
import { deleteWithConfirm } from './deleteWithConfirm'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('deleteWithConfirm', () => {
  it('does nothing if the user cancels the confirmation', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const onSuccess = vi.fn()

    await deleteWithConfirm('Delete this?', '/api/thing/1', onSuccess)

    expect(fetchMock).not.toHaveBeenCalled()
    expect(onSuccess).not.toHaveBeenCalled()
  })

  it('DELETEs the url and calls onSuccess when confirmed and the request succeeds', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true }))
    vi.stubGlobal('fetch', fetchMock)
    const onSuccess = vi.fn()

    await deleteWithConfirm('Delete this?', '/api/thing/1', onSuccess)

    expect(fetchMock).toHaveBeenCalledWith('/api/thing/1', { method: 'DELETE' })
    expect(onSuccess).toHaveBeenCalled()
  })

  it('calls onError with a message when the request fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false })))
    const onSuccess = vi.fn()
    const onError = vi.fn()

    await deleteWithConfirm('Delete this?', '/api/thing/1', onSuccess, onError)

    expect(onSuccess).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledWith('Failed to delete')
  })

  it('does not throw when the request fails and no onError is given', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false })))

    await expect(deleteWithConfirm('Delete this?', '/api/thing/1', vi.fn())).resolves.toBeUndefined()
  })
})
