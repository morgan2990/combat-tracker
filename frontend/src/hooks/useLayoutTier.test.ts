import { afterEach, describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useLayoutTier } from './useLayoutTier'

const TABLET_QUERY = '(min-width: 768px)'
const DESKTOP_QUERY = '(min-width: 1300px)'

interface FakeMediaQueryList {
  matches: boolean
  listeners: Set<(ev: { matches: boolean }) => void>
}

let mediaQueries: Map<string, FakeMediaQueryList>

function installMatchMedia(initial: { tablet: boolean; desktop: boolean }) {
  mediaQueries = new Map([
    [TABLET_QUERY, { matches: initial.tablet, listeners: new Set() }],
    [DESKTOP_QUERY, { matches: initial.desktop, listeners: new Set() }],
  ])

  window.matchMedia = ((query: string) => {
    const mql = mediaQueries.get(query)
    if (!mql) throw new Error(`unexpected media query: ${query}`)
    return {
      get matches() { return mql.matches },
      media: query,
      addEventListener: (_: string, listener: (ev: { matches: boolean }) => void) => { mql.listeners.add(listener) },
      removeEventListener: (_: string, listener: (ev: { matches: boolean }) => void) => { mql.listeners.delete(listener) },
    } as unknown as MediaQueryList
  }) as typeof window.matchMedia
}

function setMatches(query: string, matches: boolean) {
  const mql = mediaQueries.get(query)!
  mql.matches = matches
  for (const listener of mql.listeners) listener({ matches })
}

afterEach(() => {
  // @ts-expect-error -- test-only cleanup of the installed stub
  delete window.matchMedia
})

describe('useLayoutTier', () => {
  it('reports phone when neither breakpoint matches', () => {
    installMatchMedia({ tablet: false, desktop: false })
    const { result } = renderHook(() => useLayoutTier())
    expect(result.current).toBe('phone')
  })

  it('reports tablet when only the tablet breakpoint matches', () => {
    installMatchMedia({ tablet: true, desktop: false })
    const { result } = renderHook(() => useLayoutTier())
    expect(result.current).toBe('tablet')
  })

  it('reports desktop when the desktop breakpoint matches', () => {
    installMatchMedia({ tablet: true, desktop: true })
    const { result } = renderHook(() => useLayoutTier())
    expect(result.current).toBe('desktop')
  })

  it('updates when a media query change fires', () => {
    installMatchMedia({ tablet: false, desktop: false })
    const { result } = renderHook(() => useLayoutTier())
    expect(result.current).toBe('phone')

    act(() => setMatches(TABLET_QUERY, true))
    expect(result.current).toBe('tablet')

    act(() => setMatches(DESKTOP_QUERY, true))
    expect(result.current).toBe('desktop')
  })

  it('stops listening after unmount', () => {
    installMatchMedia({ tablet: false, desktop: false })
    const { result, unmount } = renderHook(() => useLayoutTier())
    unmount()

    expect(() => act(() => setMatches(TABLET_QUERY, true))).not.toThrow()
    expect(result.current).toBe('phone')
  })
})
