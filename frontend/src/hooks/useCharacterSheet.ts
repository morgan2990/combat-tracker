import { useCallback, useEffect, useState } from 'react'
import type { SheetResponse } from '../types'

interface CharacterSheetState {
  data: SheetResponse | null
  loading: boolean
  error: string | null
  // Re-fetches without clearing what's on screen, so open editors keep their input.
  reload: () => Promise<void>
}

/**
 * Loads a character's sheet bundle. With roomId it goes through the DM's
 * read-only route for a party member; without, through the owner's route.
 */
export function useCharacterSheet(pcId: string, roomId?: string): CharacterSheetState {
  const [data, setData] = useState<SheetResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    const url = roomId
      ? `/api/rooms/${encodeURIComponent(roomId)}/pcs/${encodeURIComponent(pcId)}/sheet`
      : `/api/pcs/${encodeURIComponent(pcId)}/sheet`
    try {
      const res = await fetch(url)
      if (!res.ok) {
        setError(res.status === 403 || res.status === 404 || res.status === 401
          ? "You don't have access to this character sheet."
          : 'Failed to load the character sheet.')
        return
      }
      setData(await res.json() as SheetResponse)
      setError(null)
    } catch {
      setError('Failed to load the character sheet.')
    }
  }, [pcId, roomId])

  useEffect(() => {
    setLoading(true)
    load().finally(() => setLoading(false))
  }, [load])

  return { data, loading, error, reload: load }
}
