import { describe, expect, it } from 'vitest'
import { applyDrafts, clamp } from './CurrencyEditor'
import type { Currency } from '../types'

const baseCurrency: Currency = { pp: 0, gp: 10, ep: 0, sp: 5, cp: 100 }

describe('clamp', () => {
  it('leaves non-negative numbers unchanged', () => {
    expect(clamp(5)).toBe(5)
    expect(clamp(0)).toBe(0)
  })

  it('clamps negative numbers to 0', () => {
    expect(clamp(-5)).toBe(0)
  })
})

describe('applyDrafts', () => {
  it('applies a positive delta to a denomination', () => {
    const result = applyDrafts(baseCurrency, { gp: '+15' })
    expect(result.gp).toBe(25)
  })

  it('applies a negative delta to a denomination', () => {
    const result = applyDrafts(baseCurrency, { sp: '-3' })
    expect(result.sp).toBe(2)
  })

  it('treats a plain signed-less number as a delta too (not absolute)', () => {
    const result = applyDrafts(baseCurrency, { cp: '10' })
    expect(result.cp).toBe(110)
  })

  it('clamps a denomination at 0 rather than going negative', () => {
    const result = applyDrafts(baseCurrency, { sp: '-100' })
    expect(result.sp).toBe(0)
  })

  it('leaves denominations with no draft untouched', () => {
    const result = applyDrafts(baseCurrency, { gp: '+5' })
    expect(result.pp).toBe(baseCurrency.pp)
    expect(result.ep).toBe(baseCurrency.ep)
    expect(result.cp).toBe(baseCurrency.cp)
  })

  it('leaves a denomination untouched when its draft is blank or whitespace', () => {
    const result = applyDrafts(baseCurrency, { gp: '   ' })
    expect(result.gp).toBe(baseCurrency.gp)
  })

  it('leaves a denomination untouched when its draft is not a valid signed integer', () => {
    const result = applyDrafts(baseCurrency, { gp: 'abc', sp: '1.5' })
    expect(result.gp).toBe(baseCurrency.gp)
    expect(result.sp).toBe(baseCurrency.sp)
  })

  it('applies deltas to multiple denominations in one call', () => {
    const result = applyDrafts(baseCurrency, { gp: '+15', sp: '-3', cp: '+1' })
    expect(result).toEqual({ pp: 0, gp: 25, ep: 0, sp: 2, cp: 101 })
  })

  it('does not mutate the input currency', () => {
    const original = { ...baseCurrency }
    applyDrafts(baseCurrency, { gp: '+15' })
    expect(baseCurrency).toEqual(original)
  })
})
