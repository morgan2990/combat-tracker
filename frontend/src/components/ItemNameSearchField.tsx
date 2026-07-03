import type { CSSProperties, KeyboardEvent } from 'react'
import type { Item, ItemSearchHit } from '../types'
import { useItemCatalogSearch } from '../hooks/useItemCatalogSearch'
import { color, radius } from '../theme'

interface ItemNameSearchFieldProps {
  item: Item
  edition: string
  style?: CSSProperties
  onUpdate: (patch: Partial<Item>) => void
  // Called with the id of a selected catalog result before the reference_url
  // follow-up fetch resolves. The caller checks its own item list (scoped
  // however it needs — a single PC's list, or one party member's) for an
  // existing row already carrying that catalog_item_id; if found, it
  // increments that row's quantity and returns true, in which case this
  // field's own row is left untouched. Returns false if no match was found,
  // in which case this field fills its own row once the follow-up resolves.
  onSelectExisting: (catalogItemId: string) => boolean
}

// ItemNameSearchField is the item-name field shared by InventoryPanel and
// PartyInventoryPanel: a freely-editable text input that also offers a
// debounced catalog search dropdown, and shows a link icon once the row is
// backed by a catalog selection. See design.md Decisions 4-5.
export function ItemNameSearchField({ item, edition, style, onUpdate, onSelectExisting }: ItemNameSearchFieldProps) {
  const { results, showDropdown, resolve } = useItemCatalogSearch(item.name, edition)

  function handleNameChange(name: string) {
    onUpdate({ name, catalog_item_id: undefined, reference_url: undefined })
  }

  async function handleSelect(hit: ItemSearchHit) {
    if (onSelectExisting(hit.id)) {
      onUpdate({ name: '' })
      return
    }
    const resolved = await resolve(hit)
    onUpdate({
      name: resolved.name,
      catalog_item_id: resolved.catalog_item_id,
      reference_url: resolved.reference_url || undefined,
    })
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && showDropdown && results[0]) {
      e.preventDefault()
      handleSelect(results[0])
    }
  }

  return (
    <div style={{ position: 'relative', flex: 1, ...style }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <input
          value={item.name}
          onChange={e => handleNameChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Item name"
          style={{ ...inputStyle, flex: 1 }}
        />
        {item.reference_url && (
          <a
            href={item.reference_url}
            target="_blank"
            rel="noopener noreferrer"
            title="Open 5etools reference"
            style={linkIconStyle}
          >
            🔗
          </a>
        )}
      </div>
      {showDropdown && (
        <div style={dropdownStyle}>
          {results.map(hit => (
            <div key={hit.id} onClick={() => handleSelect(hit)} style={dropdownItemStyle}>
              <span style={{ color: color.textPrimary }}>{hit.name}</span>
              <span style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <span style={{ fontSize: 11, color: color.textSecondary }}>{hit.category}</span>
                {hit.rarity && <span style={{ fontSize: 10, color: color.info }}>{hit.rarity}</span>}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

const inputStyle: CSSProperties = {
  padding: '6px 8px', fontSize: 13, background: color.bgPanel, border: `1px solid ${color.border}`, borderRadius: radius.sm, color: color.textPrimary,
}

const linkIconStyle: CSSProperties = {
  fontSize: 13, textDecoration: 'none', flexShrink: 0,
}

const dropdownStyle: CSSProperties = {
  position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 10,
  background: '#1a1a38', border: `1px solid ${color.border}`, borderRadius: radius.sm,
  marginTop: 2, maxHeight: 180, overflowY: 'auto',
}

const dropdownItemStyle: CSSProperties = {
  padding: '6px 10px', cursor: 'pointer', display: 'flex',
  justifyContent: 'space-between', alignItems: 'center', fontSize: 13,
  borderBottom: `1px solid ${color.border}`,
}
