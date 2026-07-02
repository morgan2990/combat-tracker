import type { Entity } from '../types'
import placeholderSrc from '../assets/statblock-placeholder.svg'

interface StatblockColumnProps {
  entity: Entity | null
  onClose: () => void
}

// Desktop-tier sibling to StatblockDrawer: renders the open creature's
// statblock as this column's content instead of a viewport overlay. Phone
// and tablet tiers keep StatblockDrawer entirely unchanged.
export function StatblockColumn({ entity, onClose }: StatblockColumnProps) {
  return (
    <div style={{
      height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column',
      background: '#12121e', border: '1px solid #2e2e48', borderRadius: 6, overflow: 'hidden',
    }}>
      {entity ? (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #2e2e48', flexShrink: 0 }}>
            <span style={{ fontWeight: 600, color: '#d4d4e8', fontSize: 15 }}>{entity.name}</span>
            <button
              onClick={onClose}
              style={{ background: 'none', border: 'none', color: '#7878a0', fontSize: 20, cursor: 'pointer', lineHeight: 1 }}
            >
              ✕
            </button>
          </div>
          <div style={{ flex: 1, overflow: 'hidden' }}>
            {entity.source_type === 'url' && entity.reference_url && (
              <iframe
                src={entity.reference_url}
                title={`${entity.name} statblock`}
                style={{ width: '100%', height: '100%', border: 'none' }}
              />
            )}
            {entity.source_type === 'pdf' && (
              <embed
                src={`/api/monsters/${encodeURIComponent(entity.name)}/pdf`}
                type="application/pdf"
                style={{ width: '100%', height: '100%' }}
              />
            )}
          </div>
        </>
      ) : (
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 12, padding: 24, textAlign: 'center' }}>
          <img src={placeholderSrc} alt="" style={{ width: 96, height: 96, opacity: 0.5 }} />
          <span style={{ fontSize: 13, color: '#454568' }}>Select a creature's 📋 icon to view its statblock here</span>
        </div>
      )}
    </div>
  )
}
