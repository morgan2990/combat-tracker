import { describe, expect, it } from 'vitest'
import { parseHP } from './EntityRow'

describe('parseHP', () => {
  it('applies a positive delta to the current HP', () => {
    expect(parseHP('+5', 10, 20)).toBe(15)
  })

  it('applies a negative delta to the current HP', () => {
    expect(parseHP('-4', 10, 20)).toBe(6)
  })

  it('treats a plain number as an absolute value, not a delta', () => {
    expect(parseHP('7', 10, 20)).toBe(7)
  })

  it('clamps the result to the max HP', () => {
    expect(parseHP('+100', 10, 20)).toBe(20)
  })

  it('clamps the result to a minimum of 0', () => {
    expect(parseHP('-100', 10, 20)).toBe(0)
  })

  it('falls back to current HP when the delta is unparseable', () => {
    expect(parseHP('+abc', 10, 20)).toBe(10)
    expect(parseHP('-abc', 10, 20)).toBe(10)
  })

  it('falls back to current HP when the absolute value is unparseable', () => {
    expect(parseHP('abc', 10, 20)).toBe(10)
  })

  it('trims surrounding whitespace before parsing', () => {
    expect(parseHP('  +5  ', 10, 20)).toBe(15)
  })
})
