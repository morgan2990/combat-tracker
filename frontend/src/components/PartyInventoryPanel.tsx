import { useState, useEffect } from 'react'
import type { Item, Currency, PartyInventoryResponse } from '../types'
import { CurrencyEditor, CURRENCY_DENOMS } from './CurrencyEditor'
import { ItemNameSearchField } from './ItemNameSearchField'
import { mergeIntoExistingRow } from '../hooks/useItemCatalogSearch'
import { color, radius } from '../theme'

const emptyCurrency: Currency = { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 }

interface PartyInventoryPanelProps {
  roomId: string
  edition?: string
  onClose: () => void
}

interface WorkingMember {
  pcId: string
  name: string
  items: Item[]
  currency: Currency
}

function CurrencyReadout({ currency }: { currency: Currency }) {
  return (
    <div style={{ display: 'flex', gap: 12, fontSize: 13 }}>
      {CURRENCY_DENOMS.map(denom => (
        <span key={denom}>{currency[denom]} <span style={{ color: color.textMuted, textTransform: 'uppercase' }}>{denom}</span></span>
      ))}
    </div>
  )
}

export function PartyInventoryPanel({ roomId, edition, onClose }: PartyInventoryPanelProps) {
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [partyLinked, setPartyLinked] = useState(false)
  const [partyCurrency, setPartyCurrency] = useState<Currency>(emptyCurrency)
  const [membersCurrencyTotal, setMembersCurrencyTotal] = useState<Currency>(emptyCurrency)
  const [members, setMembers] = useState<WorkingMember[]>([])
  const [filterPcId, setFilterPcId] = useState('')
  const [savingPcId, setSavingPcId] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    setLoadError(null)
    fetch(`/api/rooms/${encodeURIComponent(roomId)}/party-inventory`)
      .then(res => {
        if (!res.ok) {
          throw new Error(res.status === 403
            ? "You don't have permission to view this room's party inventory."
            : 'Failed to load party inventory.')
        }
        return res.json() as Promise<PartyInventoryResponse>
      })
      .then(data => {
        setPartyLinked(data.party_linked)
        setPartyCurrency(data.party_currency)
        setMembersCurrencyTotal(data.members_currency_total)
        const itemsByOwner: Record<string, Item[]> = {}
        for (const row of data.items) {
          (itemsByOwner[row.owner_pc_id] ??= []).push({
            name: row.name,
            quantity: row.quantity,
            catalog_item_id: row.catalog_item_id,
            reference_url: row.reference_url,
          })
        }
        setMembers(data.members.map(m => ({
          pcId: m.pc_id,
          name: m.name,
          items: itemsByOwner[m.pc_id] ?? [],
          currency: m.currency,
        })))
      })
      .catch(err => setLoadError(err instanceof Error ? err.message : 'Failed to load party inventory.'))
      .finally(() => setLoading(false))
  }, [roomId])

  function updateMemberItems(pcId: string, next: Item[]) {
    setMembers(prev => prev.map(m => m.pcId === pcId ? { ...m, items: next } : m))
  }

  function updateItem(pcId: string, index: number, patch: Partial<Item>) {
    const member = members.find(m => m.pcId === pcId)
    if (!member) return
    updateMemberItems(pcId, member.items.map((it, idx) => idx === index ? { ...it, ...patch } : it))
  }

  // Merges a catalog selection into a sibling row within the *same* member's
  // item list only — a selection made while editing one member never affects
  // another member's rows (see design.md Decision 5 / the dm-party-inventory
  // spec's per-member merge scenarios).
  function trySelectExisting(pcId: string, rowIndex: number, catalogItemId: string): boolean {
    const member = members.find(m => m.pcId === pcId)
    if (!member) return false
    const merged = mergeIntoExistingRow(member.items, rowIndex, catalogItemId)
    if (!merged) return false
    updateMemberItems(pcId, merged)
    return true
  }

  async function saveMember(pcId: string) {
    const member = members.find(m => m.pcId === pcId)
    if (!member) return
    setSavingPcId(pcId)
    setSaveError(null)
    try {
      const res = await fetch(`/api/rooms/${encodeURIComponent(roomId)}/party-inventory/pcs/${encodeURIComponent(pcId)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          items: member.items.filter(it => it.name.trim()),
          currency: member.currency,
        }),
      })
      if (!res.ok) throw new Error(`Failed to save (${res.status})`)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save.')
    } finally {
      setSavingPcId(null)
    }
  }

  const filteredMember = filterPcId ? members.find(m => m.pcId === filterPcId) : undefined
  const visibleMembers = filteredMember ? [filteredMember] : members
  const displayedMembersCurrency = filteredMember ? filteredMember.currency : membersCurrencyTotal

  return (
    <div
      style={overlayStyle}
      onClick={onClose}
      onKeyDown={e => { if (e.key === 'Escape') onClose() }}
    >
      <div style={panelStyle} onClick={e => e.stopPropagation()}>
        <div style={headerStyle}>
          <h3 style={{ margin: 0, fontSize: 18 }}>Party Inventory</h3>
          <button onClick={onClose} style={closeBtn}>✕</button>
        </div>

        {loading && <div style={{ padding: '20px 0', color: color.textSecondary, fontSize: 13 }}>Loading…</div>}

        {!loading && loadError && <div style={errorStyle}>{loadError}</div>}

        {!loading && !loadError && !partyLinked && (
          <div style={{ fontSize: 13, color: color.textSecondary }}>
            No party is linked to this room yet. A party links automatically the first time one of its members joins.
          </div>
        )}

        {!loading && !loadError && partyLinked && (
          <>
            <div style={{ marginBottom: 16 }}>
              <select value={filterPcId} onChange={e => setFilterPcId(e.target.value)} style={selectStyle}>
                <option value="">All members</option>
                {members.map(m => <option key={m.pcId} value={m.pcId}>{m.name}</option>)}
              </select>
            </div>

            <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap', marginBottom: 20 }}>
              <div>
                <div style={sectionLabel}>Party Purse</div>
                <CurrencyReadout currency={partyCurrency} />
              </div>
              <div>
                <div style={sectionLabel}>{filteredMember ? `${filteredMember.name}'s Currency` : "Members' Currency (total)"}</div>
                <CurrencyReadout currency={displayedMembersCurrency} />
              </div>
            </div>

            {visibleMembers.length === 0 && (
              <div style={{ fontSize: 13, color: color.textMuted }}>No members in this party.</div>
            )}

            {visibleMembers.map(member => (
              <div key={member.pcId} style={memberSectionStyle}>
                <div style={{ fontWeight: 600, marginBottom: 8 }}>{member.name}</div>

                {member.items.length === 0 && (
                  <div style={{ fontSize: 13, color: color.textMuted, padding: '4px 0' }}>No items.</div>
                )}
                {member.items.map((item, i) => (
                  <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 6 }}>
                    <ItemNameSearchField
                      item={item}
                      edition={edition ?? '5e'}
                      onUpdate={patch => updateItem(member.pcId, i, patch)}
                      onSelectExisting={catalogItemId => trySelectExisting(member.pcId, i, catalogItemId)}
                    />
                    <input
                      type="number"
                      min={1}
                      value={item.quantity}
                      onChange={e => updateItem(member.pcId, i, { quantity: Math.max(1, Number.parseInt(e.target.value, 10) || 1) })}
                      style={{ ...inputStyle, width: 60 }}
                    />
                    <button type="button" onClick={() => updateMemberItems(member.pcId, member.items.filter((_, idx) => idx !== i))} style={removeBtn}>Remove</button>
                  </div>
                ))}
                <button type="button" onClick={() => updateMemberItems(member.pcId, [...member.items, { name: '', quantity: 1 }])} style={addBtn}>
                  + Add Item
                </button>

                <div style={{ marginTop: 12, marginBottom: 10 }}>
                  <div style={sectionLabel}>Currency</div>
                  <CurrencyEditor
                    currency={member.currency}
                    onChange={c => setMembers(prev => prev.map(m => m.pcId === member.pcId ? { ...m, currency: c } : m))}
                  />
                </div>

                <button onClick={() => saveMember(member.pcId)} disabled={savingPcId === member.pcId} style={saveBtn(savingPcId === member.pcId)}>
                  {savingPcId === member.pcId ? 'Saving…' : `Save ${member.name}`}
                </button>
              </div>
            ))}

            {saveError && <div style={errorStyle}>{saveError}</div>}
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
  width: 520, maxWidth: '90vw', maxHeight: '85vh', overflowY: 'auto',
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

const selectStyle: React.CSSProperties = {
  padding: '8px 10px', fontSize: 14, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const memberSectionStyle: React.CSSProperties = {
  padding: '12px 0', borderTop: `1px solid ${color.border}`, marginTop: 4,
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
    marginTop: 10, padding: '8px 16px', fontSize: 13, fontWeight: 600,
    background: color.success, color: '#fff', border: 'none', borderRadius: radius.sm,
    cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.6 : 1,
  }
}
