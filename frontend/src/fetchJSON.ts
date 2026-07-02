// Fetches url and parses the JSON body, returning fallback if the response
// is not ok or the request fails. Callers that need to distinguish those two
// failure modes (e.g. to decide whether to retry) should not use this —
// wrap fetch directly instead.
export async function fetchJSON<T>(url: string, fallback: T): Promise<T> {
  try {
    const res = await fetch(url)
    if (!res.ok) return fallback
    return await res.json() as T
  } catch {
    return fallback
  }
}
