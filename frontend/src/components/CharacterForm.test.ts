import { describe, expect, it } from 'vitest'
import { parseCompanionRow } from './CharacterForm'

describe('parseCompanionRow', () => {
  it('parses a valid row', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '11', shares_initiative: false }))
      .toEqual({ name: 'Wolf', max_hp: 11 })
  })

  it('trims the name', () => {
    expect(parseCompanionRow({ name: '  Wolf  ', max_hp: '11', shares_initiative: false })!.name)
      .toBe('Wolf')
  })

  it('rejects a blank name', () => {
    expect(parseCompanionRow({ name: '   ', max_hp: '11', shares_initiative: false })).toBeNull()
  })

  it('rejects a missing max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '', shares_initiative: false })).toBeNull()
  })

  it('rejects a zero max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '0', shares_initiative: false })).toBeNull()
  })

  it('rejects a negative max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: '-5', shares_initiative: false })).toBeNull()
  })

  it('rejects an unparseable max_hp', () => {
    expect(parseCompanionRow({ name: 'Wolf', max_hp: 'abc', shares_initiative: false })).toBeNull()
  })
})
