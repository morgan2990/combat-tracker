import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useCharacterSheet } from '../hooks/useCharacterSheet'
import { SectionCard } from '../sheet/SectionCard'
import { CREATURE_SECTION, PC_SECTIONS } from '../sheet/sheetSchema'
import { color } from '../theme'

/**
 * A character's full sheet. The owner can edit each section; with ?room=<id>
 * it is the DM's read-only view of a party member.
 */
export function CharacterSheetPage() {
  const { id = '' } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const roomId = params.get('room') ?? undefined
  const readOnly = Boolean(roomId)
  const { data, loading, error, reload } = useCharacterSheet(id, roomId)

  if (loading && !data) return <div style={pageStyle}>Loading…</div>

  return (
    <div style={pageStyle}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <button
          onClick={() => navigate('/')}
          style={{ background: 'none', border: 'none', color: color.textSecondary, fontSize: 14, cursor: 'pointer', padding: 0 }}
        >
          ← Back
        </button>
        {data && (
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>
            {data.pc.name} <span style={{ fontSize: 13, color: color.textMuted, fontWeight: 400 }}>{data.pc.max_hp} HP</span>
          </h1>
        )}
        {readOnly && <span style={{ fontSize: 12, color: color.textSecondary }}>Read-only · DM view</span>}
      </div>

      {error && (
        <div role="alert" style={{ padding: '10px 12px', border: `1px solid ${color.danger}`, borderRadius: 6, color: color.danger, fontSize: 13 }}>
          {error}
        </div>
      )}

      {data && (
        <>
          {(data.pc.type === 'companion' ? [CREATURE_SECTION] : PC_SECTIONS).map(spec => (
            <SectionCard
              key={spec.key}
              spec={spec}
              bundle={data}
              idPrefix="sheet"
              editable={!readOnly}
              onChanged={reload}
            />
          ))}

          {data.companions.length > 0 && (
            <>
              <h2 style={{ fontSize: 16, margin: '28px 0 12px' }}>Companions</h2>
              {data.companions.map(c => (
                <SectionCard
                  key={c.pc.id}
                  spec={CREATURE_SECTION}
                  bundle={c}
                  title={c.pc.name}
                  idPrefix={`sheet.companion.${c.pc.id}`}
                  editable={!readOnly}
                  onChanged={reload}
                />
              ))}
            </>
          )}
        </>
      )}
    </div>
  )
}

const pageStyle: React.CSSProperties = { maxWidth: 720, margin: '32px auto', padding: '0 16px' }
