import { useState, useEffect, useRef } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import type { EncounterMonster, MonsterSearchHit, CustomMonster } from '../types'
import { useLayoutTier } from '../hooks/useLayoutTier'
import { CustomMonsterList } from './CustomMonsterList'
import { CustomMonsterPillList } from './CustomMonsterPillList'

const SEARCH_MIN_CHARS = 3
const SEARCH_DEBOUNCE_MS = 175

export function EncounterForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id?: string }>()
  const editing = Boolean(id)
  const tier = useLayoutTier()

  const [name, setName] = useState('')
  const [edition, setEdition] = useState<'5e' | '5.5e'>('5e')
  const [monsters, setMonsters] = useState<EncounterMonster[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(editing)
  const [submitting, setSubmitting] = useState(false)

  const [query, setQuery] = useState('')
  const [results, setResults] = useState<MonsterSearchHit[]>([])
  const [showDropdown, setShowDropdown] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const [myCreatures, setMyCreatures] = useState<CustomMonster[]>([])

  useEffect(() => {
    fetch(`/api/custom-monsters?edition=${encodeURIComponent(edition)}`)
      .then(res => res.ok ? res.json() : [])
      .then((data: CustomMonster[]) => setMyCreatures(data))
      .catch(() => setMyCreatures([]))
  }, [edition])

  // In edit mode, load the existing encounter.
  useEffect(() => {
    if (!id) return
    fetch(`/api/encounters/${encodeURIComponent(id)}`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (!data) return
        setName(data.name)
        setEdition(data.edition)
        setMonsters(data.monsters ?? [])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)

    if (query.trim().length < SEARCH_MIN_CHARS) {
      setResults([])
      setShowDropdown(false)
      return
    }

    debounceRef.current = setTimeout(async () => {
      try {
        const res = await fetch(`/api/search/monsters?q=${encodeURIComponent(query.trim())}&edition=${encodeURIComponent(edition)}`)
        if (!res.ok) return
        const hits = await res.json() as MonsterSearchHit[]
        setResults(hits)
        setShowDropdown(true)
      } catch {
        setResults([])
        setShowDropdown(false)
      }
    }, SEARCH_DEBOUNCE_MS)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [query, edition])

  function addMonster(hit: MonsterSearchHit) {
    setMonsters(prev => [...prev, {
      name: hit.name,
      monster_id: hit.is_custom ? hit.id : undefined,
      is_custom: hit.is_custom,
      quantity: 1,
      display_name: '',
    }])
    setQuery('')
    setResults([])
    setShowDropdown(false)
  }

  function addCustomMonster(m: CustomMonster) {
    setMonsters(prev => [...prev, {
      name: m.name,
      monster_id: m.id,
      is_custom: true,
      quantity: 1,
      display_name: '',
    }])
  }

  function handleSearchKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && showDropdown && results[0]) {
      e.preventDefault()
      addMonster(results[0])
    }
  }

  function updateGroup(i: number, patch: Partial<EncounterMonster>) {
    setMonsters(prev => prev.map((m, idx) => idx === i ? { ...m, ...patch } : m))
  }

  function removeGroup(i: number) {
    setMonsters(prev => prev.filter((_, idx) => idx !== i))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const body = { name: name.trim(), edition, monsters }
      const res = await fetch(editing ? `/api/encounters/${encodeURIComponent(id!)}` : '/api/encounters', {
        method: editing ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) { setError(`Error: ${await res.text()}`); return }
      navigate('/')
    } catch {
      setError('Network error. Is the server running?')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return null

  return (
    <div style={{ maxWidth: 560, margin: '40px auto', padding: 24 }}>
      <h2 style={{ marginTop: 0 }}>{editing ? 'Edit Encounter' : 'New Encounter'}</h2>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>

        <label style={labelStyle}>
          <span style={labelText}>Encounter Name</span>
          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Goblin Ambush"
            required
            style={fieldStyle}
          />
        </label>

        <div style={labelStyle}>
          <span style={labelText}>Edition</span>
          <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
            {(['5e', '5.5e'] as const).map(ed => (
              <button
                key={ed}
                type="button"
                onClick={() => setEdition(ed)}
                style={{
                  padding: '6px 14px', fontSize: 13, cursor: 'pointer', borderRadius: 4,
                  border: '1px solid',
                  borderColor: edition === ed ? '#3498db' : '#2e2e48',
                  background: edition === ed ? '#0d1f38' : '#1a1a2c',
                  color: edition === ed ? '#3498db' : '#8888aa',
                }}
              >
                {ed}
              </button>
            ))}
          </div>
        </div>

        {tier === 'phone' ? (
          myCreatures.length > 0 && (
            <div>
              <span style={labelText}>My Creatures</span>
              <CustomMonsterPillList monsters={myCreatures} onSelect={addCustomMonster} />
            </div>
          )
        ) : (
          <CustomMonsterList monsters={myCreatures} onSelect={addCustomMonster} />
        )}

        <div style={{ position: 'relative' }}>
          <label style={labelStyle}>
            <span style={labelText}>Add Monster</span>
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              onKeyDown={handleSearchKeyDown}
              placeholder="Type 3+ characters…"
              style={fieldStyle}
            />
          </label>
          {showDropdown && results.length > 0 && (
            <div style={{
              position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 10,
              background: '#1a1a38', border: '1px solid #2e2e48', borderRadius: 4,
              marginTop: 2, maxHeight: 220, overflowY: 'auto',
            }}>
              {results.map(hit => (
                <div
                  key={hit.id}
                  onClick={() => addMonster(hit)}
                  style={{
                    padding: '8px 10px', cursor: 'pointer', display: 'flex',
                    justifyContent: 'space-between', alignItems: 'center', fontSize: 13,
                    borderBottom: '1px solid #2e2e48',
                  }}
                >
                  <span style={{ color: '#d4d4e8' }}>
                    {hit.name}
                    {hit.is_custom && (
                      <span style={{ marginLeft: 6, fontSize: 11, color: '#e67e22' }}>
                        by {hit.owner_display_name ?? 'unknown'}
                      </span>
                    )}
                  </span>
                  <span style={{ color: '#7878a0' }}>{hit.max_hp} HP</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={labelStyle}>
          <span style={labelText}>Monster Groups</span>
          {monsters.length === 0 && (
            <div style={{ fontSize: 13, color: '#5a5a78' }}>No monsters added yet.</div>
          )}
          {monsters.map((m, i) => (
            <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #2e2e48' }}>
              <span style={{ flex: 1, fontSize: 13, color: '#d4d4e8' }}>{m.name}</span>
              <input
                type="number"
                min={1}
                value={m.quantity}
                onChange={e => updateGroup(i, { quantity: Math.max(1, parseInt(e.target.value, 10) || 1) })}
                style={{ ...fieldStyle, width: 52 }}
              />
              <input
                value={m.display_name ?? ''}
                onChange={e => updateGroup(i, { display_name: e.target.value })}
                placeholder="Alias (optional)"
                style={{ ...fieldStyle, width: 140 }}
              />
              <button type="button" onClick={() => removeGroup(i)} style={btnStyle('#2e2e48')}>Remove</button>
            </div>
          ))}
        </div>

        {error && <div style={{ color: '#e74c3c', fontSize: 13 }}>{error}</div>}

        <button
          type="submit"
          disabled={submitting}
          style={btnStyle('#e67e22', submitting)}
        >
          {submitting ? 'Saving…' : editing ? 'Save Changes' : 'Save Encounter'}
        </button>
      </form>
    </div>
  )
}

const labelStyle: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 4 }
const labelText: React.CSSProperties = { fontSize: 12, color: '#7878a0' }
const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: '100%', boxSizing: 'border-box' }
function btnStyle(bg: string, disabled = false): React.CSSProperties {
  return { padding: '10px 20px', fontSize: 14, background: disabled ? '#444' : bg, color: '#fff', border: 'none', borderRadius: 4, cursor: disabled ? 'default' : 'pointer' }
}
