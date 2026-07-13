import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { mergeIntoExistingRow, useItemCatalogSearch } from './useItemCatalogSearch'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('mergeIntoExistingRow', () => {
  const items = [
    { catalog_item_id: 'a', quantity: 1 },
    { catalog_item_id: 'b', quantity: 2 },
  ]

  it('increments the quantity of a sibling row with a matching catalog_item_id', () => {
    const result = mergeIntoExistingRow(items, 0, 'b')
    expect(result).toEqual([{ catalog_item_id: 'a', quantity: 1 }, { catalog_item_id: 'b', quantity: 3 }])
  })

  it('returns null when no sibling row matches', () => {
    expect(mergeIntoExistingRow(items, 0, 'c')).toBeNull()
  })

  it('ignores a match at rowIndex itself', () => {
    expect(mergeIntoExistingRow(items, 0, 'a')).toBeNull()
  })
})

describe('useItemCatalogSearch', () => {
  it('does not search on mount even if the initial query is long enough', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderHook(() => useItemCatalogSearch('Longsword', '5e'))

    await new Promise(r => setTimeout(r, 250))
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('does not search below the minimum character threshold', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const { result, rerender } = renderHook(({ query }) => useItemCatalogSearch(query, '5e'), { initialProps: { query: '' } })

    rerender({ query: 'Lo' })
    await new Promise(r => setTimeout(r, 250))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.showDropdown).toBe(false)
  })

  it('searches after the debounce once the query changes and is long enough', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({
      ok: true, json: () => Promise.resolve([{ id: 'i1', name: 'Longsword', category: 'Weapon' }]),
    }))
    vi.stubGlobal('fetch', fetchMock)
    const { result, rerender } = renderHook(({ query }) => useItemCatalogSearch(query, '5e'), { initialProps: { query: '' } })

    rerender({ query: 'Long' })

    await waitFor(() => expect(result.current.results).toHaveLength(1))
    expect(result.current.showDropdown).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/api/search/items?q=Long&edition=5e')
  })

  it('defaults edition to 5e when empty', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([]) }))
    vi.stubGlobal('fetch', fetchMock)
    const { rerender } = renderHook(({ query }) => useItemCatalogSearch(query, ''), { initialProps: { query: '' } })

    rerender({ query: 'Long' })

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('edition=5e'))
  })

  it('resolve() dismisses the dropdown, clears results, and fetches the full catalog item', async () => {
    const fetchMock = vi.fn(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ id: 'i1', name: 'Longsword +1', edition: '5e', category: 'Weapon', reference_url: 'https://example.com/longsword' }),
    }))
    vi.stubGlobal('fetch', fetchMock)
    const { result } = renderHook(() => useItemCatalogSearch('', '5e'))

    const resolved = await result.current.resolve({ id: 'i1', name: 'Longsword', category: 'Weapon' })

    expect(resolved).toEqual({ catalog_item_id: 'i1', name: 'Longsword +1', reference_url: 'https://example.com/longsword' })
    expect(fetchMock).toHaveBeenCalledWith('/api/items/i1')
  })

  it('resolve() falls back to the hit name and empty url when the follow-up fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, json: () => Promise.resolve(null) })))
    const { result } = renderHook(() => useItemCatalogSearch('', '5e'))

    const resolved = await result.current.resolve({ id: 'i1', name: 'Longsword', category: 'Weapon' })

    expect(resolved).toEqual({ catalog_item_id: 'i1', name: 'Longsword', reference_url: '' })
  })
})
