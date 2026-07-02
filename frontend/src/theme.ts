// Shared color and radius tokens for the small set of values reused across
// many components. Deliberately does not cover every hex literal in the
// codebase — several colors are single-purpose (e.g. entityVitals.ts's
// dead/unconscious/active row-color functions) or coincidental value
// matches with no shared meaning, and forcing those into shared tokens
// would create false coupling between unrelated design decisions.
export const color = {
  textPrimary: '#d4d4e8',
  textSecondary: '#7878a0',
  textMuted: '#5a5a78',
  border: '#2e2e48',
  bgPanel: '#1a1a2c',
  bgCard: '#161626',
  accent: '#e67e22',
  danger: '#e74c3c',
  info: '#3498db',
  success: '#27ae60',
} as const

export const radius = {
  sm: 4,
  md: 6,
  pill: 12,
} as const
