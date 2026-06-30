import type { Entity } from '../types'

interface StatblockDrawerProps {
  entity: Entity
  open: boolean
  onClose: () => void
}

export function StatblockDrawer({ entity, open, onClose }: StatblockDrawerProps) {
  return (
    <div style={{
      position: 'fixed', top: 0, right: 0, bottom: 0,
      width: 420, maxWidth: '95vw',
      background: '#12121e', borderLeft: '1px solid #2e2e48',
      display: 'flex', flexDirection: 'column',
      zIndex: 100, boxShadow: '-4px 0 24px rgba(0,0,0,0.6)',
      transform: open ? 'translateX(0)' : 'translateX(105%)',
      transition: 'transform 0.2s ease',
    }}>
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
        {entity.source_type === 'pdf' && open && (
          <embed
            src={`/api/monsters/${encodeURIComponent(entity.name)}/pdf`}
            type="application/pdf"
            style={{ width: '100%', height: '100%' }}
          />
        )}
      </div>
    </div>
  )
}
