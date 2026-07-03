import { useState, useEffect } from 'react'
import type { Item, Currency, PartyInventoryResponse } from '../types'
import { CurrencyEditor } from './CurrencyEditor'
import { ItemNameSearchField } from './ItemNameSearchField'
import { mergeIntoExistingRow } from '../hooks/useItemCatalogSearch'
import { color, radius } from '../theme'

const emptyCurrency: Currency = { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 }

interface ParsedInventory {
  pcName: string
  maxHP: number
  items: Item[]
  currency: Currency
}

// GET /api/pcs/:id response, the owner's own view.
function parseOwnerResponse(data: { pc: { name: string; max_hp: number; items?: Item[]; currency?: Currency } }): ParsedInventory {
  return {
    pcName: data.pc.name,
    maxHP: data.pc.max_hp,
    items: data.pc.items ?? [],
    currency: data.pc.currency ?? emptyCurrency,
  }
}

// GET .../party-inventory response, sliced down to one member — the DM's view.
function parsePartyResponse(resp: PartyInventoryResponse, pcId: string): ParsedInventory {
  const member = resp.members.find(m => m.pc_id === pcId)
  return {
    pcName: member?.name ?? '',
    maxHP: 0,
    items: resp.items.filter(it => it.owner_pc_id === pcId).map(it => ({
      name: it.name,
      quantity: it.quantity,
      catalog_item_id: it.catalog_item_id,
      reference_url: it.reference_url,
    })),
    currency: member?.currency ?? emptyCurrency,
  }
}

interface InventoryPanelProps {
  pcId: string
  // When set, the DM is viewing this PC via the room's linked-party inventory
  // rather than the PC's own owner — fetches/saves go through the room-scoped
  // endpoints instead of /api/pcs/:id.
  roomId?: string
  edition?: string
  onClose: () => void
}

export function InventoryPanel({ pcId, roomId, edition, onClose }: InventoryPanelProps) {
  const [pcName, setPcName] = useState('')
  const [maxHP, setMaxHP] = useState(0)
  const [items, setItems] = useState<Item[]>([])
  const [currency, setCurrency] = useState<Currency>(emptyCurrency)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setLoading(true)
    setLoadError(null)
    const url = roomId
      ? `/api/rooms/${encodeURIComponent(roomId)}/party-inventory`
      : `/api/pcs/${encodeURIComponent(pcId)}`
    fetch(url)
      .then(res => {
        if (!res.ok) {
          throw new Error(res.status === 403 || res.status === 404
            ? "You don't have permission to view this character's inventory."
            : 'Failed to load inventory.')
        }
        return res.json()
      })
      .then(data => {
        const parsed = roomId ? parsePartyResponse(data as PartyInventoryResponse, pcId) : parseOwnerResponse(data)
        setPcName(parsed.pcName)
        setMaxHP(parsed.maxHP)
        setItems(parsed.items)
        setCurrency(parsed.currency)
      })
      .catch(err => setLoadError(err instanceof Error ? err.message : 'Failed to load inventory.'))
      .finally(() => setLoading(false))
  }, [pcId, roomId])

  function addItem() {
    setItems(prev => [...prev, { name: '', quantity: 1 }])
  }

  function updateItem(i: number, patch: Partial<Item>) {
    setItems(prev => prev.map((it, idx) => idx === i ? { ...it, ...patch } : it))
  }

  function removeItem(i: number) {
    setItems(prev => prev.filter((_, idx) => idx !== i))
  }

  function trySelectExisting(rowIndex: number, catalogItemId: string): boolean {
    const merged = mergeIntoExistingRow(items, rowIndex, catalogItemId)
    if (!merged) return false
    setItems(merged)
    return true
  }

  async function handleSave() {
    setSaving(true)
    setSaveError(null)
    try {
      const url = roomId
        ? `/api/rooms/${encodeURIComponent(roomId)}/party-inventory/pcs/${encodeURIComponent(pcId)}`
        : `/api/pcs/${encodeURIComponent(pcId)}`
      const body = roomId
        ? { items: items.filter(it => it.name.trim()), currency }
        : { name: pcName, max_hp: maxHP, items: items.filter(it => it.name.trim()), currency }
      const res = await fetch(url, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error(`Failed to save (${res.status})`)
      onClose()
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save inventory.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={panelStyle} onClick={e => e.stopPropagation()}>
        <div style={headerStyle}>
          <h3 style={{ margin: 0, fontSize: 18 }}>{pcName ? `${pcName}'s Inventory` : 'Inventory'}</h3>
          <button onClick={onClose} style={closeBtn}>✕</button>
        </div>

        {loading && <div style={{ padding: '20px 0', color: color.textSecondary, fontSize: 13 }}>Loading…</div>}

        {!loading && loadError && <div style={errorStyle}>{loadError}</div>}

        {!loading && !loadError && (
          <>
            <div style={{ marginBottom: 16 }}>
              <div style={sectionLabel}>Items</div>
              {items.length === 0 && (
                <div style={{ fontSize: 13, color: color.textMuted, padding: '6px 0' }}>No items yet.</div>
              )}
              {items.map((item, i) => (
                <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 6 }}>
                  <ItemNameSearchField
                    item={item}
                    edition={edition ?? '5e'}
                    onUpdate={patch => updateItem(i, patch)}
                    onSelectExisting={catalogItemId => trySelectExisting(i, catalogItemId)}
                  />
                  <input
                    type="number"
                    min={1}
                    value={item.quantity}
                    onChange={e => updateItem(i, { quantity: Math.max(1, parseInt(e.target.value, 10) || 1) })}
                    style={{ ...inputStyle, width: 60 }}
                  />
                  <button type="button" onClick={() => removeItem(i)} style={removeBtn}>Remove</button>
                </div>
              ))}
              <button type="button" onClick={addItem} style={addBtn}>+ Add Item</button>
            </div>

            <div style={{ marginBottom: 20 }}>
              <div style={sectionLabel}>Currency</div>
              <CurrencyEditor currency={currency} onChange={setCurrency} />
            </div>

            {saveError && <div style={errorStyle}>{saveError}</div>}

            <button onClick={handleSave} disabled={saving} style={saveBtn(saving)}>
              {saving ? 'Saving…' : 'Save'}
            </button>
          </>
        )}
      </div>
    </div>
  )
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
  display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100,
}

const panelStyle: React.CSSProperties = {
  width: 420, maxWidth: '90vw', maxHeight: '85vh', overflowY: 'auto',
  background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: 8, padding: 20,
}

const headerStyle: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16,
}

const closeBtn: React.CSSProperties = {
  background: 'none', border: 'none', color: color.textSecondary, fontSize: 16, cursor: 'pointer', padding: 4,
}

const sectionLabel: React.CSSProperties = {
  fontSize: 12, color: color.textSecondary, textTransform: 'uppercase', letterSpacing: '0.05em',
  marginBottom: 8, fontWeight: 600,
}

const inputStyle: React.CSSProperties = {
  padding: '6px 8px', fontSize: 13, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const removeBtn: React.CSSProperties = {
  fontSize: 12, cursor: 'pointer', padding: '4px 8px', background: 'none', color: color.danger, border: '1px solid #3a1010', borderRadius: radius.sm,
}

const addBtn: React.CSSProperties = {
  marginTop: 4, fontSize: 13, cursor: 'pointer', padding: '6px 12px', background: color.border, color: color.textPrimary, border: 'none', borderRadius: radius.sm,
}

const errorStyle: React.CSSProperties = {
  marginBottom: 12, fontSize: 13, color: color.danger, padding: '8px 10px', background: '#1a0808', border: `1px solid ${color.danger}`, borderRadius: radius.md,
}

function saveBtn(disabled: boolean): React.CSSProperties {
  return {
    width: '100%', padding: '10px 0', fontSize: 14, fontWeight: 600,
    background: color.success, color: '#fff', border: 'none', borderRadius: radius.sm,
    cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.6 : 1,
  }
}
