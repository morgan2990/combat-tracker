import type { SheetRecord } from '../types'
import { color, radius } from '../theme'
import { isEmptyValue } from './sheetSchema'
import type { FieldDef, SheetContext } from './sheetSchema'

interface FieldsViewProps {
  fields: FieldDef[]
  value: SheetRecord
  ctx: SheetContext
  // A list item's heading field is shown in the card title, not repeated here.
  skipKey?: string
}

/** Read-only rendering of a record: only non-empty fields appear. */
export function FieldsView({ fields, value, ctx, skipKey }: FieldsViewProps) {
  const shown = fields.filter(f => f.key !== skipKey && !f.hideInView && !isEmptyValue(f, value[f.key]))
  return (
    <div>
      {shown.map(f => <FieldView key={f.key} field={f} value={value[f.key]} ctx={ctx} />)}
    </div>
  )
}

function FieldView({ field, value, ctx }: { field: FieldDef; value: unknown; ctx: SheetContext }) {
  if (field.kind === 'group') {
    return (
      <div style={blockStyle}>
        <div style={blockLabel}>{field.label}</div>
        <div style={indentStyle}>
          <FieldsView fields={field.fields} value={value as SheetRecord} ctx={ctx} />
        </div>
      </div>
    )
  }

  if (field.kind === 'list') {
    const items = value as SheetRecord[]
    // A list's items are headed by their name when they have one.
    const titleKey = field.titleKey ?? (field.fields.some(f => f.key === 'name') ? 'name' : undefined)
    return (
      <div style={blockStyle}>
        {!field.hideLabel && <div style={blockLabel}>{field.label}</div>}
        {items.map((item, i) => (
          <div key={i} style={cardStyle}>
            {(titleKey || field.badge) && (
              <div style={{ display: 'flex', gap: 8, alignItems: 'baseline' }}>
                {titleKey && <strong>{String(item[titleKey] ?? '')}</strong>}
                {field.badge && <BadgeView item={item} badge={field.badge} />}
              </div>
            )}
            <FieldsView fields={field.fields} value={item} ctx={ctx} skipKey={titleKey} />
            {field.extra?.(item, ctx)}
          </div>
        ))}
      </div>
    )
  }

  return (
    <div style={rowStyle}>
      <span style={labelStyle}>{field.label}</span>
      <span style={{ whiteSpace: 'pre-wrap', minWidth: 0 }}>{displayText(field, value)}</span>
    </div>
  )
}

function BadgeView({ item, badge }: { item: SheetRecord; badge: (item: SheetRecord) => { text: string } | null }) {
  const b = badge(item)
  if (!b) return null
  return <span style={badgeStyle}>{b.text}</span>
}

function displayText(field: FieldDef, value: unknown): string {
  switch (field.kind) {
    case 'bool': return '✓'
    case 'strings': return (value as string[]).join(', ')
    case 'numbers': {
      const parts = (value as number[]).map((n, i) => (n ? `${field.itemLabel(i)}: ${n}` : null)).filter(Boolean)
      return parts.join(' · ')
    }
    default: return String(value)
  }
}

const rowStyle: React.CSSProperties = { display: 'flex', gap: 10, fontSize: 13, padding: '2px 0' }
const labelStyle: React.CSSProperties = { color: color.textSecondary, minWidth: 110, flexShrink: 0 }
const blockStyle: React.CSSProperties = { marginTop: 6 }
const blockLabel: React.CSSProperties = { fontSize: 12, fontWeight: 600, color: color.textSecondary, textTransform: 'uppercase', letterSpacing: '0.04em' }
const indentStyle: React.CSSProperties = { paddingLeft: 10 }
const cardStyle: React.CSSProperties = {
  background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: radius.md,
  padding: '8px 10px', marginTop: 6, fontSize: 13,
}
const badgeStyle: React.CSSProperties = {
  fontSize: 11, color: color.accent, border: `1px solid ${color.border}`, borderRadius: radius.pill, padding: '0 8px',
}
