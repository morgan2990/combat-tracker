import { describe, expect, it } from 'vitest'
import { entityVitalState, vitalRowBg, vitalTextColor } from './entityVitals'
import { color } from './theme'

describe('entityVitalState', () => {
  it('returns dead when dead is true, regardless of HP', () => {
    expect(entityVitalState(true, 10)).toBe('dead')
    expect(entityVitalState(true, 0)).toBe('dead')
  })

  it('returns unconscious when HP is exactly 0 and not dead', () => {
    expect(entityVitalState(false, 0)).toBe('unconscious')
  })

  it('returns alive when HP is above 0 and not dead', () => {
    expect(entityVitalState(false, 1)).toBe('alive')
    expect(entityVitalState(false, 50)).toBe('alive')
  })
})

describe('vitalRowBg', () => {
  it('prioritizes dead over active/me', () => {
    expect(vitalRowBg('dead', true, true)).toBe('#141414')
  })

  it('prioritizes unconscious over active/me', () => {
    expect(vitalRowBg('unconscious', true, true)).toBe('#1a1608')
  })

  it('uses the active color when alive and active', () => {
    expect(vitalRowBg('alive', true, false)).toBe('#1f1508')
  })

  it('uses the "me" color when alive, not active, and isMe', () => {
    expect(vitalRowBg('alive', false, true)).toBe('#0f1a2c')
  })

  it('defaults isMe to false, falling through to the panel background', () => {
    expect(vitalRowBg('alive', false)).toBe(color.bgPanel)
  })
})

describe('vitalTextColor', () => {
  it('returns a distinct color per state', () => {
    const dead = vitalTextColor('dead')
    const unconscious = vitalTextColor('unconscious')
    const alive = vitalTextColor('alive')
    expect(new Set([dead, unconscious, alive]).size).toBe(3)
  })
})
