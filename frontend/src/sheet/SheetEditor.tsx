import type { SheetRecord } from '../types'
import { color, radius } from '../theme'
import { blankRecord } from './sheetSchema'
import type { FieldDef } from './sheetSchema'

interface FieldsEditorProps {
  fields: FieldDef[]
  value: SheetRecord
  onChange: (next: SheetRecord) => void
}

/** Form for a record. Controlled: every edit calls onChange with a new record. */
export function FieldsEditor({ fields, value, onChange }: FieldsEditorProps) {
  return (
    <div>
      {fields.map(f => (
        <FieldEditor
          key={f.key}
          field={f}
          value={value[f.key]}
          onChange={v => onChange({ ...value, [f.key]: v })}
        />
      ))}
    </div>
  )
}

function FieldEditor({ field, value, onChange }: { field: FieldDef; value: unknown; onChange: (v: unknown) => void }) {
  switch (field.kind) {
    case 'text':
      return (
        <label style={labelStyle}>
          <span>{field.label}</span>
          <input value={(value as string | undefined) ?? ''} onChange={e => onChange(e.target.value)} style={inputStyle} />
        </label>
      )
    case 'textarea':
      return (
        <label style={labelStyle}>
          <span>{field.label}</span>
          <textarea value={(value as string | undefined) ?? ''} onChange={e => onChange(e.target.value)} rows={3} style={inputStyle} />
        </label>
      )
    case 'number':
      return (
        <label style={labelStyle}>
          <span>{field.label}</span>
          <input
            type="number"
            value={(value as number | null | undefined) ?? ''}
            onChange={e => onChange(e.target.value === '' ? null : Number(e.target.value))}
            style={inputStyle}
          />
        </label>
      )
    case 'bool':
      return (
        <label style={{ ...labelStyle, flexDirection: 'row', alignItems: 'center', gap: 6 }}>
          <input type="checkbox" checked={Boolean(value)} onChange={e => onChange(e.target.checked)} />
          <span>{field.label}</span>
        </label>
      )
    case 'select':
      return (
        <label style={labelStyle}>
          <span>{field.label}</span>
          <select value={(value as string | undefined) ?? ''} onChange={e => onChange(e.target.value)} style={inputStyle}>
            {field.options.map(o => <option key={o} value={o}>{o}</option>)}
          </select>
        </label>
      )
    case 'strings':
      return (
        <label style={labelStyle}>
          <span>{field.label}</span>
          <textarea
            value={((value as string[] | undefined) ?? []).join('\n')}
            onChange={e => onChange(e.target.value.split('\n'))}
            rows={3}
            style={inputStyle}
          />
        </label>
      )
    case 'numbers':
      return (
        <fieldset style={groupStyle}>
          <legend style={legendStyle}>{field.label}</legend>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {Array.from({ length: field.count }, (_, i) => {
              const current = (value as (number | null)[] | undefined) ?? []
              return (
                <label key={i} style={{ ...labelStyle, width: 90 }}>
                  <span>{field.itemLabel(i)}</span>
                  <input
                    type="number"
                    min={0}
                    value={current[i] ?? ''}
                    onChange={e => {
                      const next = Array.from({ length: field.count }, (_, j) => current[j] ?? null)
                      next[i] = e.target.value === '' ? null : Number(e.target.value)
                      onChange(next)
                    }}
                    style={inputStyle}
                  />
                </label>
              )
            })}
          </div>
        </fieldset>
      )
    case 'group':
      return (
        <fieldset style={groupStyle}>
          <legend style={legendStyle}>{field.label}</legend>
          <FieldsEditor fields={field.fields} value={(value as SheetRecord | undefined) ?? {}} onChange={onChange} />
        </fieldset>
      )
    case 'list':
      return <ListEditor field={field} items={(value as SheetRecord[] | undefined) ?? []} onChange={onChange} />
  }
}

function ListEditor({ field, items, onChange }: {
  field: Extract<FieldDef, { kind: 'list' }>
  items: SheetRecord[]
  onChange: (v: unknown) => void
}) {
  const noun = field.addLabel ?? field.label.toLowerCase()
  return (
    <fieldset style={groupStyle}>
      {!field.hideLabel && <legend style={legendStyle}>{field.label}</legend>}
      {items.map((item, i) => (
        <div key={i} role="group" aria-label={`${noun} ${i + 1}`} style={itemStyle}>
          <FieldsEditor
            fields={field.fields}
            value={item}
            onChange={next => onChange(items.map((it, idx) => (idx === i ? next : it)))}
          />
          <button
            type="button"
            aria-label={`Remove ${noun} ${i + 1}`}
            onClick={() => onChange(items.filter((_, idx) => idx !== i))}
            style={removeBtnStyle}
          >
            Remove
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={() => onChange([...items, blankRecord(field.fields)])}
        style={addBtnStyle}
      >
        + Add {noun}
      </button>
    </fieldset>
  )
}

const labelStyle: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: 3, marginBottom: 8,
  fontSize: 12, fontWeight: 600, color: color.textSecondary,
}
const inputStyle: React.CSSProperties = {
  padding: '6px 8px', fontSize: 14, width: '100%', boxSizing: 'border-box', fontWeight: 'normal',
}
const groupStyle: React.CSSProperties = {
  border: `1px solid ${color.border}`, borderRadius: radius.md, padding: '8px 10px', margin: '0 0 10px', minWidth: 0,
}
const legendStyle: React.CSSProperties = { fontSize: 12, fontWeight: 600, color: color.textSecondary, padding: '0 4px' }
const itemStyle: React.CSSProperties = {
  background: color.bgCard, border: `1px solid ${color.border}`, borderRadius: radius.md, padding: 10, marginBottom: 8,
}
const addBtnStyle: React.CSSProperties = {
  padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: color.border, color: color.textPrimary,
  border: 'none', borderRadius: radius.sm,
}
const removeBtnStyle: React.CSSProperties = {
  padding: '4px 10px', fontSize: 12, cursor: 'pointer', background: 'none', color: color.danger,
  border: `1px solid ${color.border}`, borderRadius: radius.sm,
}
