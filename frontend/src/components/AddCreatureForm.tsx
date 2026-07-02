import { useState, useEffect, useRef, forwardRef, useImperativeHandle } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import type { MonsterSearchHit, CustomMonster } from '../types'
import { CustomMonsterPillList } from './CustomMonsterPillList'
import { fetchJSON } from '../fetchJSON'
import { labelStyle, labelText } from '../formFieldStyles'

const SEARCH_MIN_CHARS = 3
const SEARCH_DEBOUNCE_MS = 175

interface AddCreatureFormProps {
  sendMessage: (msg: object) => void
  edition: string
  showMyCreaturesInline: boolean
}

interface MonsterRef { source_type: string; reference_url: string; pdf_object_key: string; initiative_modifier: number | null }

export interface AddCreatureFormHandle {
  selectCustomMonster: (m: CustomMonster) => void
}

export const AddCreatureForm = forwardRef<AddCreatureFormHandle, AddCreatureFormProps>(function AddCreatureForm(
  { sendMessage, edition, showMyCreaturesInline }, ref
) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<MonsterSearchHit[]>([])
  const [showDropdown, setShowDropdown] = useState(false)

  const [name, setName] = useState('')
  const [maxHP, setMaxHP] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [displayName, setDisplayName] = useState('')
  const [monsterRef, setMonsterRef] = useState<MonsterRef | null>(null)

  const [myCreatures, setMyCreatures] = useState<CustomMonster[]>([])

  useEffect(() => {
    // Only fetched here when this tier renders the list inline; at
    // tablet/desktop tiers DMNavColumn fetches and displays it instead.
    if (!showMyCreaturesInline) { setMyCreatures([]); return }
    fetchJSON<CustomMonster[]>(`/api/custom-monsters?edition=${encodeURIComponent(edition)}`, []).then(setMyCreatures)
  }, [edition, showMyCreaturesInline])

  function selectCustomMonster(m: CustomMonster) {
    setName(m.name)
    setMaxHP(String(m.max_hp))
    setMonsterRef({
      source_type: m.source_type ?? '',
      reference_url: m.reference_url ?? '',
      pdf_object_key: m.pdf_object_key ?? '',
      initiative_modifier: m.initiative_modifier,
    })
  }

  useImperativeHandle(ref, () => ({ selectCustomMonster }))

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)

    if (query.trim().length < SEARCH_MIN_CHARS) {
      setResults([])
      setShowDropdown(false)
      return
    }

    debounceRef.current = setTimeout(async () => {
      try {
        const ed = edition || '5e'
        const res = await fetch(`/api/search/monsters?q=${encodeURIComponent(query.trim())}&edition=${encodeURIComponent(ed)}`)
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

  async function selectMonster(hit: MonsterSearchHit) {
    setQuery('')
    setResults([])
    setShowDropdown(false)
    setName(hit.name)
    setMaxHP(String(hit.max_hp))

    try {
      const url = hit.is_custom
        ? `/api/custom-monsters/${encodeURIComponent(hit.id)}`
        : `/api/monsters/${encodeURIComponent(hit.name)}`
      const res = await fetch(url)
      if (!res.ok) { setMonsterRef(null); return }
      const m = await res.json() as { source_type?: string; reference_url?: string; pdf_object_key?: string; initiative_modifier?: number | null }
      setMonsterRef({
        source_type: m.source_type ?? '',
        reference_url: m.reference_url ?? '',
        pdf_object_key: m.pdf_object_key ?? '',
        initiative_modifier: m.initiative_modifier ?? hit.initiative_modifier,
      })
    } catch {
      setMonsterRef(null)
    }
  }

  function handleSearchKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && showDropdown && results[0]) {
      e.preventDefault()
      selectMonster(results[0])
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const hp = parseInt(maxHP, 10)
    const qty = Math.max(1, parseInt(quantity, 10) || 1)
    if (!name.trim() || !hp || hp <= 0) return
    const msg: Record<string, unknown> = {
      type: 'add_creature',
      name: name.trim(),
      max_hp: hp,
      quantity: qty,
      source_type: monsterRef?.source_type ?? '',
      reference_url: monsterRef?.reference_url ?? '',
      pdf_object_key: monsterRef?.pdf_object_key ?? '',
      display_name: displayName.trim(),
    }
    if (monsterRef?.initiative_modifier != null) {
      msg.initiative_modifier = monsterRef.initiative_modifier
    }
    sendMessage(msg)
    setName('')
    setMaxHP('')
    setQuantity('1')
    setDisplayName('')
    setMonsterRef(null)
  }

  return (
    <div style={{ padding: '12px 0' }}>
      {showMyCreaturesInline && myCreatures.length > 0 && (
        <div style={{ marginBottom: 10 }}>
          <div style={labelText}>My Creatures</div>
          <CustomMonsterPillList monsters={myCreatures} onSelect={selectCustomMonster} />
        </div>
      )}
      <div style={{ position: 'relative', marginBottom: 8, maxWidth: 240 }}>
        <label style={labelStyle}>
          <span style={labelText}>Search monsters</span>
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
                onClick={() => selectMonster(hit)}
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
                <span style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <span style={{ fontSize: 10, color: '#3498db', border: '1px solid #3498db', borderRadius: 3, padding: '1px 5px' }}>
                    {edition || '5e'}
                  </span>
                  <span style={{ color: '#7878a0' }}>{hit.max_hp} HP</span>
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <label style={labelStyle}>
          <span style={labelText}>Name</span>
          <input
            value={name}
            onChange={e => { setName(e.target.value); setMonsterRef(null) }}
            placeholder="Goblin"
            required
            style={fieldStyle}
          />
        </label>
        <label style={labelStyle}>
          <span style={labelText}>Max HP</span>
          <input type="number" value={maxHP} onChange={e => setMaxHP(e.target.value)} placeholder="14" min={1} required style={{ ...fieldStyle, width: 70 }} />
        </label>
        <label style={labelStyle}>
          <span style={labelText}>Qty</span>
          <input type="number" value={quantity} onChange={e => setQuantity(e.target.value)} min={1} style={{ ...fieldStyle, width: 52 }} />
        </label>
        <label style={labelStyle}>
          <span style={labelText}>Custom Display Name / Alias (Optional)</span>
          <input
            value={displayName}
            onChange={e => setDisplayName(e.target.value)}
            placeholder="e.g. Guard Alpha"
            style={{ ...fieldStyle, width: 160 }}
          />
        </label>
        {monsterRef && monsterRef.source_type && (
          <span style={{ fontSize: 11, color: '#27ae60', alignSelf: 'center', paddingBottom: 2 }}>statblock ready</span>
        )}
        <button type="submit" style={{ padding: '8px 16px', fontSize: 14, alignSelf: 'flex-end', background: '#e67e22', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
          Add Creature
        </button>
      </form>
    </div>
  )
})

const fieldStyle: React.CSSProperties = { padding: '8px', fontSize: 14, width: 120 }
