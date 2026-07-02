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
  )
}
