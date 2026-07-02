import { useState, useEffect } from 'react'

// Width-based layout tiers. Deliberately viewport-width only — no
// navigator/UA or touch-capability checks — since device class doesn't
// reliably predict available pixels (e.g. a high-res tablet should get the
// desktop layout; a narrowed desktop window should degrade like a phone).
export type LayoutTier = 'phone' | 'tablet' | 'desktop'

const TABLET_QUERY = '(min-width: 768px)'
const DESKTOP_QUERY = '(min-width: 1300px)'

function computeLayoutTier(): LayoutTier {
  if (typeof window === 'undefined') return 'phone'
  if (window.matchMedia(DESKTOP_QUERY).matches) return 'desktop'
  if (window.matchMedia(TABLET_QUERY).matches) return 'tablet'
  return 'phone'
}

export function useLayoutTier(): LayoutTier {
  const [tier, setTier] = useState<LayoutTier>(computeLayoutTier)

  useEffect(() => {
    const tabletQuery = window.matchMedia(TABLET_QUERY)
    const desktopQuery = window.matchMedia(DESKTOP_QUERY)
    const update = () => setTier(desktopQuery.matches ? 'desktop' : tabletQuery.matches ? 'tablet' : 'phone')
    tabletQuery.addEventListener('change', update)
    desktopQuery.addEventListener('change', update)
    return () => {
      tabletQuery.removeEventListener('change', update)
      desktopQuery.removeEventListener('change', update)
    }
  }, [])

  return tier
}
