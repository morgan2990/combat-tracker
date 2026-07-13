import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchJSON } from './fetchJSON'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('fetchJSON', () => {
  it('returns the parsed JSON body on success', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve({ hello: 'world' }) })))

    const result = await fetchJSON('/api/thing', null)

    expect(result).toEqual({ hello: 'world' })
  })

  it('returns the fallback when the response is not ok', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, json: () => Promise.resolve({}) })))

    const result = await fetchJSON('/api/thing', 'fallback')

    expect(result).toBe('fallback')
  })

  it('returns the fallback when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('network down'))))

    const result = await fetchJSON('/api/thing', [])

    expect(result).toEqual([])
  })
})
