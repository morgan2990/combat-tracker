import type { ReactNode } from 'react'
import type { CharacterSheet, SheetBundle, SheetRecord } from '../types'

// The sheet UI is driven by these schemas rather than one hand-written view
// and form per section: a schema says which fields a section has, and the
// shared SheetView / SheetEditor render and edit any of them. Anything that
// needs more than a field list (spell markers, a magic item's uses) hangs off
// the schema as a `badge` or `extra` hook. Field keys match the API exactly.

export interface Badge {
  text: string
}

export interface SheetContext {
  sheet: CharacterSheet
}

interface FieldBase {
  key: string
  label: string
  hideInView?: boolean
  hideLabel?: boolean
}

export type FieldDef =
  | (FieldBase & { kind: 'text' | 'textarea' | 'number' | 'bool' | 'strings' })
  | (FieldBase & { kind: 'select'; options: string[] })
  | (FieldBase & { kind: 'numbers'; count: number; itemLabel: (i: number) => string })
  | (FieldBase & { kind: 'group'; fields: FieldDef[] })
  | (FieldBase & {
      kind: 'list'
      fields: FieldDef[]
      // The field shown as each item's heading (and left out of its body).
      titleKey?: string
      addLabel?: string
      badge?: (item: SheetRecord) => Badge | null
      extra?: (item: SheetRecord, ctx: SheetContext) => ReactNode
    })

export interface SectionSpec {
  // The API section name, which is also the key on the sheet.
  key: string
  title: string
  // An array section is a bare list on the wire; the UI edits it as {items}.
  array?: boolean
  // Object section: its fields. Array section: the fields of each item.
  fields: FieldDef[]
  titleKey?: string
  addLabel?: string
  badge?: (item: SheetRecord) => Badge | null
  extra?: (item: SheetRecord, ctx: SheetContext) => ReactNode
  // Shown while editing.
  note?: string
}

const NO_RECALC_NOTE = 'Values are stored exactly as entered — nothing is recalculated.'

const text = (key: string, label: string): FieldDef => ({ kind: 'text', key, label })
const area = (key: string, label: string): FieldDef => ({ kind: 'textarea', key, label })
const num = (key: string, label: string): FieldDef => ({ kind: 'number', key, label })
const bool = (key: string, label: string): FieldDef => ({ kind: 'bool', key, label })

const ABILITIES: [string, string][] = [
  ['str', 'Strength'], ['dex', 'Dexterity'], ['con', 'Constitution'],
  ['int', 'Intelligence'], ['wis', 'Wisdom'], ['cha', 'Charisma'],
]

const abilityGroups: FieldDef[] = ABILITIES.map(([key, label]) => ({
  kind: 'group', key, label,
  fields: [num('score', 'Score'), num('mod', 'Modifier'), num('save_bonus', 'Save bonus'), bool('save_proficient', 'Save proficient')],
}))

const skillFields: FieldDef[] = [
  text('key', 'Skill'), text('ability', 'Ability'), num('bonus', 'Bonus'),
  bool('proficient', 'Proficient'), bool('expertise', 'Expertise'),
]

const attackFields: FieldDef[] = [
  text('name', 'Name'), text('ability', 'Ability'), text('range', 'Range'), text('to_hit', 'To hit'),
  text('damage', 'Damage'), text('damage_type', 'Damage type'), area('notes', 'Description / properties'),
]

const actionFields: FieldDef[] = [text('name', 'Name'), text('source', 'Source'), bool('is_default', 'Default action')]

const proficiencyFields: FieldDef[] = [text('name', 'Name'), bool('pending', 'Choice pending')]

// Shared by a companion's sheet and each wild shape form.
const creatureFields: FieldDef[] = [
  text('name', 'Name'), text('species', 'Species'), text('type', 'Type'), text('size', 'Size'),
  text('alignment', 'Alignment'), text('ac', 'AC'), text('hp', 'HP'), text('speed_text', 'Speed'),
  text('senses', 'Senses'), num('proficiency_bonus', 'Proficiency bonus'),
  { kind: 'group', key: 'abilities', label: 'Abilities', fields: abilityGroups },
  { kind: 'strings', key: 'saves', label: 'Saves (one per line)' },
  { kind: 'list', key: 'skills', label: 'Skills', titleKey: 'key', fields: skillFields },
  { kind: 'list', key: 'attacks', label: 'Attacks', fields: attackFields },
  area('traits', 'Traits'),
  { kind: 'list', key: 'gear', label: 'Gear', fields: [text('name', 'Name'), text('amount', 'Amount')] },
  area('notes', 'Notes'),
]

const PREPARATIONS = ['known', 'prepared', 'always', 'unprepared', 'at_will', 'limited', 'other']

export function spellBadge(spell: SheetRecord): Badge | null {
  switch (spell.preparation) {
    case 'always': return { text: '★ Always prepared' }
    case 'prepared': return { text: '● Prepared' }
    case 'unprepared': return { text: '○ Not prepared' }
    case 'at_will': return { text: 'At will' }
    case 'limited': return { text: 'Limited use' }
    case 'other': return spell.mark ? { text: `Mark: ${String(spell.mark)}` } : null
    default: return null
  }
}

const spellFields: FieldDef[] = [
  text('name', 'Name'), num('level', 'Level'),
  { kind: 'select', key: 'preparation', label: 'Preparation', options: PREPARATIONS, hideInView: true },
  text('mark', 'Original mark'), text('save', 'Save'), text('school', 'School'),
  text('casting_time', 'Casting time'), text('range', 'Range'), text('components', 'Components'),
  text('duration', 'Duration'), text('book', 'Book'), text('page', 'Page'), area('description', 'Description'),
]

// Limited uses that name this item as their source, shown with the item.
function itemUses(item: SheetRecord, ctx: SheetContext): ReactNode {
  const uses = (ctx.sheet.limited_uses ?? []).filter(u => u.source && u.source === item.name)
  if (uses.length === 0) return null
  return (
    <div>
      {uses.map((u, i) => (
        <div key={i}>
          Uses: {String(u.name)}
          {u.max ? ` — ${String(u.max)}` : ''}
          {u.recovery ? ` (${String(u.recovery)})` : ''}
        </div>
      ))}
    </div>
  )
}

export const PC_SECTIONS: SectionSpec[] = [
  {
    key: 'identity', title: 'Identity',
    fields: [
      text('name', 'Name'), text('player', 'Player'), text('classes_text', 'Class line'),
      num('character_level', 'Character level'),
      { kind: 'list', key: 'classes', label: 'Classes', titleKey: 'class', addLabel: 'class',
        fields: [text('class', 'Class'), text('subclass', 'Subclass'), num('level', 'Level')] },
      text('species', 'Species'), text('background', 'Background'), text('size', 'Size'),
      text('alignment', 'Alignment'), num('xp', 'XP'),
    ],
  },
  { key: 'abilities', title: 'Abilities & Saves', fields: abilityGroups, note: NO_RECALC_NOTE },
  {
    key: 'skills', title: 'Skills',
    fields: [
      { kind: 'list', key: 'entries', label: 'Skills', titleKey: 'key', addLabel: 'skill', fields: skillFields },
      num('passive_perception', 'Passive Perception'), num('passive_insight', 'Passive Insight'),
      num('passive_investigation', 'Passive Investigation'),
    ],
    note: NO_RECALC_NOTE,
  },
  {
    key: 'combat', title: 'Combat',
    fields: [
      num('max_hp', 'Max HP'), num('ac', 'AC'), num('initiative', 'Initiative'),
      num('proficiency_bonus', 'Proficiency bonus'), text('speed_text', 'Speed'),
      { kind: 'group', key: 'speeds', label: 'Speeds',
        fields: [num('walk', 'Walk'), num('climb', 'Climb'), num('swim', 'Swim'), num('fly', 'Fly'), num('burrow', 'Burrow')] },
      { kind: 'list', key: 'hit_dice', label: 'Hit dice', addLabel: 'hit dice',
        fields: [num('die', 'Die (sides)'), num('total', 'Total')] },
    ],
    note: NO_RECALC_NOTE,
  },
  {
    key: 'defenses', title: 'Defenses',
    fields: [text('vision', 'Vision'), { kind: 'strings', key: 'resistances', label: 'Resistances (one per line)' }, area('save_notes', 'Save notes')],
  },
  { key: 'attacks', title: 'Attacks', array: true, addLabel: 'attack', fields: attackFields },
  {
    key: 'actions', title: 'Actions',
    fields: [
      { kind: 'list', key: 'action', label: 'Actions', addLabel: 'action', fields: actionFields },
      { kind: 'list', key: 'bonus_action', label: 'Bonus actions', addLabel: 'bonus action', fields: actionFields },
      { kind: 'list', key: 'reaction', label: 'Reactions', addLabel: 'reaction', fields: actionFields },
    ],
  },
  {
    key: 'limited_uses', title: 'Limited Uses', array: true, addLabel: 'limited use',
    fields: [text('name', 'Name'), text('max', 'Max uses'), text('recovery', 'Recovery'), text('source', 'Source')],
  },
  {
    key: 'features', title: 'Features & Feats', array: true, addLabel: 'feature',
    fields: [
      { kind: 'select', key: 'kind', label: 'Kind', options: ['class', 'species', 'background', 'feat', 'other'] },
      text('name', 'Name'), area('text', 'Text'), text('source', 'Source'), num('level', 'Level'),
      text('book', 'Book'), text('page', 'Page'), text('uses_note', 'Uses note'),
    ],
  },
  {
    key: 'weapon_masteries', title: 'Weapon Masteries', array: true, titleKey: 'weapon', addLabel: 'weapon mastery',
    fields: [text('weapon', 'Weapon'), text('mastery', 'Mastery'), area('notes', 'Notes')],
  },
  {
    key: 'proficiencies', title: 'Proficiencies',
    fields: [
      { kind: 'list', key: 'armor', label: 'Armor', addLabel: 'armor', fields: proficiencyFields },
      { kind: 'list', key: 'weapons', label: 'Weapons', addLabel: 'weapon', fields: proficiencyFields },
      { kind: 'list', key: 'tools', label: 'Tools', addLabel: 'tool', fields: proficiencyFields },
      { kind: 'list', key: 'languages', label: 'Languages', addLabel: 'language', fields: proficiencyFields },
    ],
  },
  {
    key: 'magic_items', title: 'Magic Items', array: true, addLabel: 'magic item', extra: itemUses,
    fields: [
      text('name', 'Name'), text('stage', 'Stage / level'), bool('attunement_required', 'Requires attunement'),
      bool('attuned', 'Attuned'), area('text', 'Description'), text('source', 'Source'),
    ],
  },
  {
    key: 'spellcasting', title: 'Spellcasting',
    fields: [
      { kind: 'list', key: 'spell_lists', label: 'Spell lists', titleKey: 'name', addLabel: 'spell list',
        fields: [
          { kind: 'select', key: 'kind', label: 'Source', options: ['class', 'feat', 'item', 'other'] },
          text('name', 'Name'), text('ability', 'Ability'), text('save_dc', 'Spell save DC'), text('attack', 'Spell attack'),
          num('prepare_count', 'Spells to prepare'),
          { kind: 'list', key: 'spells', label: 'Spells', addLabel: 'spell', badge: spellBadge, fields: spellFields },
        ] },
      { kind: 'numbers', key: 'slots', label: 'Spell slots', count: 9, itemLabel: i => `Level ${i + 1} slots` },
      { kind: 'group', key: 'pact_magic', label: 'Pact Magic',
        fields: [num('slot_level', 'Slot level'), num('total', 'Slots')] },
    ],
    note: NO_RECALC_NOTE,
  },
  {
    key: 'wild_shapes', title: 'Wild Shapes',
    fields: [
      text('uses', 'Uses'), text('duration', 'Duration'), text('limitations', 'Limitations'),
      { kind: 'list', key: 'forms', label: 'Forms', addLabel: 'form', fields: creatureFields },
    ],
  },
  {
    key: 'notes', title: 'Notes',
    fields: [
      area('appearance', 'Appearance'), area('personality', 'Personality'), area('ideals', 'Ideals'),
      area('bonds', 'Bonds'), area('flaws', 'Flaws'), area('backstory', 'Backstory'), area('free_text', 'Notes'),
    ],
  },
]

export const CREATURE_SECTION: SectionSpec = { key: 'creature', title: 'Companion', fields: creatureFields }

// --- value helpers shared by the view and the editor ---

/** The fields the engine renders for a section: array sections become a single list. */
export function specFields(spec: SectionSpec): FieldDef[] {
  if (!spec.array) return spec.fields
  return [{
    kind: 'list', key: 'items', label: spec.title, hideLabel: true, fields: spec.fields,
    titleKey: spec.titleKey ?? 'name', addLabel: spec.addLabel, badge: spec.badge, extra: spec.extra,
  }]
}

/** The editable/viewable value for a section: the PC's name and max HP come from the PC, not the sheet. */
export function sectionValue(spec: SectionSpec, bundle: SheetBundle): SheetRecord {
  const raw = bundle.sheet[spec.key as keyof CharacterSheet]
  if (spec.array) return { items: Array.isArray(raw) ? raw : [] }
  const value: SheetRecord = { ...(raw as SheetRecord | undefined) }
  if (spec.key === 'identity') value.name = bundle.pc.name
  if (spec.key === 'combat') value.max_hp = bundle.pc.max_hp
  return value
}

export function isEmptyValue(field: FieldDef, v: unknown): boolean {
  if (v === null || v === undefined) return true
  switch (field.kind) {
    case 'text': case 'textarea': case 'select': return v === ''
    case 'number': return false
    case 'bool': return v === false
    case 'strings': return !Array.isArray(v) || v.every(s => String(s).trim() === '')
    case 'numbers': return !Array.isArray(v) || v.every(n => !n)
    case 'group': return field.fields.every(f => isEmptyValue(f, (v as SheetRecord)[f.key]))
    case 'list': return !Array.isArray(v) || v.length === 0
  }
}

/** Number of non-empty top-level entries, shown as the section's count. */
export function sectionCount(spec: SectionSpec, value: SheetRecord): number {
  if (spec.array) return Array.isArray(value.items) ? value.items.length : 0
  return spec.fields.filter(f => !isEmptyValue(f, value[f.key])).length
}

function blankField(f: FieldDef): unknown {
  switch (f.kind) {
    case 'text': case 'textarea': return ''
    case 'number': return null
    case 'bool': return false
    case 'select': return f.options[0]
    case 'strings': return []
    case 'numbers': return Array.from({ length: f.count }, () => null)
    case 'group': return blankRecord(f.fields)
    case 'list': return []
  }
}

/** A new, empty record for a list item. */
export function blankRecord(fields: FieldDef[]): SheetRecord {
  return Object.fromEntries(fields.map(f => [f.key, blankField(f)]))
}

/**
 * Fills defaults a form control would otherwise show but not hold: a select
 * with no stored value displays its first option, so the value must be that
 * option or saving would send an empty (invalid) one.
 */
export function normalizeRecord(fields: FieldDef[], record: SheetRecord): SheetRecord {
  const out: SheetRecord = { ...record }
  for (const f of fields) {
    const v = out[f.key]
    if (f.kind === 'select' && !v) out[f.key] = f.options[0]
    else if (f.kind === 'group') out[f.key] = normalizeRecord(f.fields, (v as SheetRecord | undefined) ?? {})
    else if (f.kind === 'list' && Array.isArray(v)) out[f.key] = v.map(item => normalizeRecord(f.fields, item as SheetRecord))
  }
  return out
}

/** The request body's data for a section, from the editor's value. */
export function toRequestData(spec: SectionSpec, value: SheetRecord): unknown {
  const pruned = pruneRecord(specFields(spec), value)
  return spec.array ? (pruned.items ?? []) : pruned
}

function pruneRecord(fields: FieldDef[], record: SheetRecord): SheetRecord {
  const out: SheetRecord = { ...record }
  for (const f of fields) {
    const v = out[f.key]
    if (f.kind === 'strings' && Array.isArray(v)) {
      out[f.key] = v.map(s => String(s).trim()).filter(Boolean)
    } else if (f.kind === 'numbers' && Array.isArray(v)) {
      const totals = v.map(n => (typeof n === 'number' ? n : 0))
      while (totals.length > 0 && totals[totals.length - 1] === 0) totals.pop()
      out[f.key] = totals
    } else if (f.kind === 'group' && v) {
      const pruned = pruneRecord(f.fields, v as SheetRecord)
      // A group with nothing filled in is left out rather than saved as zeros.
      if (f.fields.every(sub => isEmptyValue(sub, pruned[sub.key]))) delete out[f.key]
      else out[f.key] = pruned
    } else if (f.kind === 'list' && Array.isArray(v)) {
      out[f.key] = v.map(item => pruneRecord(f.fields, item as SheetRecord))
    }
  }
  return out
}
