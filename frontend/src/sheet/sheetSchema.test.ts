import { describe, expect, it } from 'vitest'
import type { SheetBundle } from '../types'
import {
  CREATURE_SECTION, PC_SECTIONS, blankRecord, isEmptyValue, normalizeRecord,
  sectionCount, sectionValue, spellBadge, specFields, toRequestData,
} from './sheetSchema'
import type { SectionSpec } from './sheetSchema'

function spec(key: string): SectionSpec {
  const s = PC_SECTIONS.find(x => x.key === key)
  if (!s) throw new Error(`no section ${key}`)
  return s
}

const bundle: SheetBundle = {
  pc: {
    id: 'pc1', owner_user_id: 'u1', name: 'Aria', type: 'pc', max_hp: 16, shares_initiative: false,
    items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
  },
  sheet: {
    pc_id: 'pc1', schema_version: 1, revision: 3,
    identity: { species: 'Eladrin' }, attacks: [{ name: 'Club' }], combat: { ac: 13 },
  },
}

describe('section catalogue', () => {
  it('covers every PC section the API accepts, in order, plus the companion creature', () => {
    expect(PC_SECTIONS.map(s => s.key)).toEqual([
      'identity', 'abilities', 'skills', 'combat', 'defenses', 'attacks', 'actions', 'limited_uses',
      'features', 'weapon_masteries', 'proficiencies', 'magic_items', 'spellcasting', 'wild_shapes', 'notes',
    ])
    expect(CREATURE_SECTION.key).toBe('creature')
  })
})

describe('sectionValue', () => {
  it("takes the identity name and combat max HP from the PC, not the sheet", () => {
    expect(sectionValue(spec('identity'), bundle)).toEqual({ species: 'Eladrin', name: 'Aria' })
    expect(sectionValue(spec('combat'), bundle)).toEqual({ ac: 13, max_hp: 16 })
  })

  it('wraps an array section as {items}, empty when nothing is stored', () => {
    expect(sectionValue(spec('attacks'), bundle)).toEqual({ items: [{ name: 'Club' }] })
    expect(sectionValue(spec('features'), bundle)).toEqual({ items: [] })
  })

  it('gives an object section with nothing stored an empty record', () => {
    expect(sectionValue(spec('notes'), bundle)).toEqual({})
  })
})

describe('sectionCount', () => {
  it('counts items of an array section and filled fields of an object section', () => {
    expect(sectionCount(spec('attacks'), { items: [{}, {}] })).toBe(2)
    expect(sectionCount(spec('notes'), { personality: 'Curious', flaws: '' })).toBe(1)
    expect(sectionCount(spec('notes'), {})).toBe(0)
  })
})

describe('isEmptyValue', () => {
  const [name, level, attuned] = [
    { kind: 'text', key: 'a', label: 'A' }, { kind: 'number', key: 'b', label: 'B' }, { kind: 'bool', key: 'c', label: 'C' },
  ] as const

  it('treats blank text, false, null and empty lists as empty', () => {
    expect(isEmptyValue(name, '')).toBe(true)
    expect(isEmptyValue(name, undefined)).toBe(true)
    expect(isEmptyValue(attuned, false)).toBe(true)
    expect(isEmptyValue(level, null)).toBe(true)
  })

  it('does not treat a zero as empty — a score of 0 is a real value', () => {
    expect(isEmptyValue(level, 0)).toBe(false)
    expect(isEmptyValue(name, 'x')).toBe(false)
    expect(isEmptyValue(attuned, true)).toBe(false)
  })

  it('treats a group as empty only when every field in it is', () => {
    const group = specFields(spec('combat')).find(f => f.key === 'speeds')!
    expect(isEmptyValue(group, { walk: null })).toBe(true)
    expect(isEmptyValue(group, { walk: 30 })).toBe(false)
  })
})

describe('normalizeRecord / blankRecord', () => {
  const spells = specFields(spec('spellcasting')).find(f => f.key === 'spell_lists')!

  it('starts a new list item with a valid first option in every select', () => {
    if (spells.kind !== 'list') throw new Error('expected a list')
    expect(blankRecord(spells.fields)).toMatchObject({ kind: 'class', name: '', spells: [] })
  })

  it('fills a missing select value so the form and the saved value agree', () => {
    const out = normalizeRecord(specFields(spec('spellcasting')), {
      spell_lists: [{ name: 'Druid', spells: [{ name: 'Shield' }] }],
    })
    const list = (out.spell_lists as Record<string, unknown>[])[0]
    expect(list.kind).toBe('class')
    expect((list.spells as Record<string, unknown>[])[0].preparation).toBe('known')
  })

  it('keeps a stored select value', () => {
    const out = normalizeRecord(specFields(spec('features')), { items: [{ kind: 'feat', name: 'Alert' }] })
    expect((out.items as Record<string, unknown>[])[0].kind).toBe('feat')
  })
})

describe('toRequestData', () => {
  it('sends an array section as a bare list', () => {
    expect(toRequestData(spec('attacks'), { items: [{ name: 'Club' }] })).toEqual([{ name: 'Club' }])
    expect(toRequestData(spec('attacks'), {})).toEqual([])
  })

  it('leaves out a group nothing was entered in, rather than saving zeros', () => {
    const data = toRequestData(spec('spellcasting'), { pact_magic: { slot_level: null, total: null } }) as Record<string, unknown>
    expect(data).not.toHaveProperty('pact_magic')
  })

  it('keeps a group with a value in it', () => {
    const data = toRequestData(spec('spellcasting'), { pact_magic: { slot_level: 1, total: 1 } }) as Record<string, unknown>
    expect(data.pact_magic).toEqual({ slot_level: 1, total: 1 })
  })

  it('turns blank slot totals into zeros and drops trailing zeros', () => {
    const data = toRequestData(spec('spellcasting'), { slots: [3, null, 2, null, null, null, null, null, null] }) as Record<string, unknown>
    expect(data.slots).toEqual([3, 0, 2])
  })

  it('splits multi-line text fields into trimmed, non-empty entries', () => {
    const data = toRequestData(spec('defenses'), { resistances: ['Fire ', '', ' Cold'] }) as Record<string, unknown>
    expect(data.resistances).toEqual(['Fire', 'Cold'])
  })

  it('prunes inside list items too', () => {
    const data = toRequestData(spec('wild_shapes'), {
      forms: [{ name: 'Rat', abilities: { str: { score: null, mod: null, save_bonus: null, save_proficient: false } }, saves: ['Dex +2', ' '] }],
    }) as { forms: Record<string, unknown>[] }
    expect(data.forms[0]).not.toHaveProperty('abilities')
    expect(data.forms[0].saves).toEqual(['Dex +2'])
  })

  it('keeps free-text values exactly as typed', () => {
    const data = toRequestData(spec('attacks'), { items: [{ name: 'Sting', to_hit: 'DC 12', damage: '2d4+2' }] }) as Record<string, unknown>[]
    expect(data[0]).toMatchObject({ to_hit: 'DC 12', damage: '2d4+2' })
  })
})

describe('spellBadge', () => {
  it('gives always-prepared and prepared spells different markers', () => {
    const always = spellBadge({ preparation: 'always' })
    const prepared = spellBadge({ preparation: 'prepared' })
    expect(always?.text).toBeTruthy()
    expect(prepared?.text).toBeTruthy()
    expect(always?.text).not.toBe(prepared?.text)
  })

  it('has no marker for a plain known spell and shows the original mark for "other"', () => {
    expect(spellBadge({ preparation: 'known' })).toBeNull()
    expect(spellBadge({ preparation: 'other', mark: 'circle' })?.text).toContain('circle')
    expect(spellBadge({ preparation: 'other' })).toBeNull()
  })
})
