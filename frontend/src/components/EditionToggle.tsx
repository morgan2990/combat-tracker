import { color, radius } from '../theme'
interface EditionToggleProps {
  edition: '5e' | '5.5e'
  onChange: (edition: '5e' | '5.5e') => void
}

export function EditionToggle({ edition, onChange }: EditionToggleProps) {
  return (
    <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
      {(['5e', '5.5e'] as const).map(ed => (
        <button
          key={ed}
          type="button"
          onClick={() => onChange(ed)}
          style={{
            padding: '6px 14px', fontSize: 13, cursor: 'pointer', borderRadius: radius.sm,
            border: '1px solid',
            borderColor: edition === ed ? color.info : color.border,
            background: edition === ed ? '#0d1f38' : color.bgPanel,
            color: edition === ed ? color.info : '#8888aa',
          }}
        >
          {ed}
        </button>
      ))}
    </div>
  )
}
