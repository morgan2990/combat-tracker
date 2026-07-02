export const CONDITIONS = ['Prone', 'Stunned', 'Poisoned', 'Blinded', 'Frightened', 'Incapacitated', 'Restrained', 'Paralyzed']

export type VitalState = 'dead' | 'unconscious' | 'alive'

export function entityVitalState(dead: boolean, currentHP: number): VitalState {
  if (dead) return 'dead'
  if (currentHP === 0) return 'unconscious'
  return 'alive'
}

export function vitalRowBg(vitalState: VitalState, isActive: boolean, isMe = false): string {
  if (vitalState === 'dead') return '#141414'
  if (vitalState === 'unconscious') return '#1a1608'
  if (isActive) return '#1f1508'
  if (isMe) return '#0f1a2c'
  return '#1a1a2c'
}

export function vitalTextColor(vitalState: VitalState): string {
  if (vitalState === 'dead') return '#585858'
  if (vitalState === 'unconscious') return '#9090a0'
  return '#d4d4e8'
}
