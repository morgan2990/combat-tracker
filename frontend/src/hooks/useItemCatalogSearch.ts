import { useEffect, useRef, useState } from 'react'
import type { CatalogItem, ItemSearchHit } from '../types'
import { fetchJSON } from '../fetchJSON'

const SEARCH_MIN_CHARS = 3
const SEARCH_DEBOUNCE_MS = 175

export interface ResolvedCatalogItem {
  catalog_item_id: string
  name: string
  reference_url: string
}

// mergeIntoExistingRow finds a sibling row (excluding rowIndex, the one being
// edited) already carrying catalogItemId and increments its quantity,
// returning the updated array — or null if no such row exists, in which case
// the caller should fill rowIndex's own row instead. Shared by InventoryPanel
// and PartyInventoryPanel so the merge rule can't drift between them; each
// caller supplies its own scoped array (a PC's items, or one party member's).
export function mergeIntoExistingRow<T extends { catalog_item_id?: string; quantity: number }>(
  items: T[],
  rowIndex: number,
  catalogItemId: string,
): T[] | null {
  const existingIdx = items.findIndex((it, idx) => idx !== rowIndex && it.catalog_item_id === catalogItemId)
  if (existingIdx === -1) return null
  return items.map((it, idx) => idx === existingIdx ? { ...it, quantity: it.quantity + 1 } : it)
}

// useItemCatalogSearch drives a single item-name field's debounced search
// dropdown against GET /api/search/items, and resolves a selected hit's full
// reference_url via a follow-up GET /api/items/{id} call — shared by
// InventoryPanel and PartyInventoryPanel so both stay in sync (see
// design.md's risk about code drift between the two consumers).
export function useItemCatalogSearch(query: string, edition: string) {
  const [results, setResults] = useState<ItemSearchHit[]>([])
  const [dismissed, setDismissed] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  // Tracks the name that shouldn't trigger a fresh search: seeded with the
  // field's initial value (so a row loaded with an existing name — free-text
  // or catalog-linked — doesn't immediately search/open a dropdown against
  // itself on mount), and updated after a selection resolves (so the
  // resulting item.name update, query changing to that same value, isn't
  // treated as fresh typing either). Only an actual change away from either
  // of those triggers a search.
  const lastSelectedNameRef = useRef<string | null>(query)

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)

    if (query === lastSelectedNameRef.current || query.trim().length < SEARCH_MIN_CHARS) {
      setResults([])
      return
    }

    setDismissed(false)
    debounceRef.current = setTimeout(async () => {
      const ed = edition || '5e'
      const hits = await fetchJSON<ItemSearchHit[]>(
        `/api/search/items?q=${encodeURIComponent(query.trim())}&edition=${encodeURIComponent(ed)}`,
        [],
      )
      setResults(hits)
    }, SEARCH_DEBOUNCE_MS)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [query, edition])

  async function resolve(hit: ItemSearchHit): Promise<ResolvedCatalogItem> {
    setDismissed(true)
    setResults([])
    lastSelectedNameRef.current = hit.name
    const full = await fetchJSON<CatalogItem | null>(`/api/items/${encodeURIComponent(hit.id)}`, null)
    const resolvedName = full?.name ?? hit.name
    lastSelectedNameRef.current = resolvedName
    return {
      catalog_item_id: hit.id,
      name: resolvedName,
      reference_url: full?.reference_url ?? '',
    }
  }

  return {
    results,
    showDropdown: !dismissed && results.length > 0,
    resolve,
  }
}
