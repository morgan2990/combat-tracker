import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { CharacterSheetPage } from './CharacterSheetPage'
import type { CharacterSheet, PC, SheetResponse } from '../types'

const aria: PC = {
  id: 'pc1', owner_user_id: 'u1', name: 'Aria', type: 'pc', max_hp: 16, shares_initiative: false,
  items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
}
const rex: PC = { ...aria, id: 'c1', name: 'Rex', type: 'companion', parent_pc_id: 'pc1', max_hp: 8 }

function sheetOf(pcId: string, over: Partial<CharacterSheet> = {}): CharacterSheet {
  return { pc_id: pcId, schema_version: 1, revision: 0, ...over }
}

function response(over: Partial<SheetResponse> = {}): SheetResponse {
  return { pc: aria, sheet: sheetOf('pc1'), companions: [], ...over }
}

interface Reply { status?: number; body?: unknown; text?: string }

// Routes each request by "METHOD url" and records every call.
function stubFetch(routes: Record<string, Reply | (() => Reply)>) {
  const calls: { method: string; url: string; body?: unknown }[] = []
  const mock = vi.fn((url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ method, url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    const entry = routes[`${method} ${url}`]
    if (!entry) return Promise.resolve({ ok: false, status: 404, json: () => Promise.resolve({}), text: () => Promise.resolve('') })
    const r = typeof entry === 'function' ? entry() : entry
    const status = r.status ?? 200
    return Promise.resolve({
      ok: status >= 200 && status < 300, status,
      json: () => Promise.resolve(r.body ?? {}), text: () => Promise.resolve(r.text ?? ''),
    })
  })
  vi.stubGlobal('fetch', mock)
  return calls
}

function renderPage(path = '/characters/pc1/sheet') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/characters/:id/sheet" element={<CharacterSheetPage />} />
        <Route path="/" element={<div>dashboard</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const putsTo = (calls: ReturnType<typeof stubFetch>, url: string) => calls.filter(c => c.method === 'PUT' && c.url === url)

describe('CharacterSheetPage — owner', () => {
  it('loads the owner route and shows the name, HP and an Edit action on every section', async () => {
    const calls = stubFetch({ 'GET /api/pcs/pc1/sheet': { body: response() } })
    renderPage()

    expect(await screen.findByRole('heading', { name: /Aria/ })).toBeInTheDocument()
    expect(screen.getByText('16 HP')).toBeInTheDocument()
    expect(calls[0]).toMatchObject({ method: 'GET', url: '/api/pcs/pc1/sheet' })
    for (const title of [
      'Identity', 'Abilities & Saves', 'Skills', 'Combat', 'Defenses', 'Attacks', 'Actions', 'Limited Uses',
      'Features & Feats', 'Weapon Masteries', 'Proficiencies', 'Magic Items', 'Spellcasting', 'Wild Shapes', 'Notes',
    ]) {
      expect(screen.getByRole('button', { name: `Edit ${title}` })).toBeInTheDocument()
    }
  })

  it('renders an empty sheet with every section empty and editable', async () => {
    stubFetch({ 'GET /api/pcs/pc1/sheet': { body: response() } })
    renderPage()

    await screen.findByRole('heading', { name: /Aria/ })
    // Identity and Combat always carry the PC's name and max HP, so the other 13 are empty.
    expect(screen.getAllByText('Nothing recorded yet.')).toHaveLength(13)
    expect(screen.queryByText(/Read-only/)).not.toBeInTheDocument()
  })

  it('shows an access message when the sheet cannot be loaded', async () => {
    stubFetch({ 'GET /api/pcs/pc1/sheet': { status: 404 } })
    renderPage()

    expect(await screen.findByRole('alert')).toHaveTextContent("You don't have access")
    expect(screen.queryByRole('button', { name: /^Edit / })).not.toBeInTheDocument()
  })

  it('goes back to the dashboard', async () => {
    stubFetch({ 'GET /api/pcs/pc1/sheet': { body: response() } })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: '← Back' }))

    expect(screen.getByText('dashboard')).toBeInTheDocument()
  })
})

describe('CharacterSheetPage — read-only DM view', () => {
  it('loads through the room route and offers no editing', async () => {
    const calls = stubFetch({
      'GET /api/rooms/R1/pcs/pc1/sheet': { body: response({ sheet: sheetOf('pc1', { notes: { personality: 'Curious' } }) }) },
    })
    renderPage('/characters/pc1/sheet?room=R1')

    expect(await screen.findByText('Curious')).toBeInTheDocument()
    expect(calls[0].url).toBe('/api/rooms/R1/pcs/pc1/sheet')
    expect(screen.getByText(/Read-only/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Edit / })).not.toBeInTheDocument()
  })

  it('shows an access message when the PC is not in the room\'s party', async () => {
    stubFetch({ 'GET /api/rooms/R1/pcs/pc1/sheet': { status: 403 } })
    renderPage('/characters/pc1/sheet?room=R1')

    expect(await screen.findByRole('alert')).toHaveTextContent("You don't have access")
  })
})

describe('CharacterSheetPage — viewing sections', () => {
  it('shows stored values and hides empty fields', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({
          sheet: sheetOf('pc1', {
            identity: { species: 'Eladrin', classes: [{ class: 'Warlock', subclass: 'Archfey', level: 1 }, { class: 'Druid', subclass: 'Dreams', level: 2 }] },
            abilities: { str: { score: 8, mod: -1, save_proficient: false }, wis: { score: 18, mod: 4, save_bonus: 6, save_proficient: true } },
            limited_uses: [{ name: 'Balm', max: '2d6', recovery: 'long rest' }],
            proficiencies: { languages: [{ name: 'Sylvan', pending: false }, { name: 'from Human', pending: true }] },
          }),
        }),
      },
    })
    renderPage()

    expect(await screen.findByText('Eladrin')).toBeInTheDocument()
    expect(screen.getByText('Warlock')).toBeInTheDocument()
    expect(screen.getByText('Druid')).toBeInTheDocument()
    expect(screen.getByText('Archfey')).toBeInTheDocument()
    expect(screen.getByText('Balm')).toBeInTheDocument()
    expect(screen.getByText('2d6')).toBeInTheDocument()
    expect(screen.getByText('from Human')).toBeInTheDocument()
    expect(screen.queryByText('Dexterity')).not.toBeInTheDocument()
    expect(screen.getByText('Wisdom')).toBeInTheDocument()
  })

  it('marks always-prepared and prepared spells differently, and names the list source', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({
          sheet: sheetOf('pc1', {
            spellcasting: {
              spell_lists: [
                { kind: 'class', name: 'Druid', save_dc: '14', attack: '+6', spells: [
                  { name: 'Find Familiar', level: 1, preparation: 'always' },
                  { name: 'Healing Word', level: 1, preparation: 'prepared' },
                  { name: 'Absorb Elements', level: 1, preparation: 'unprepared' },
                  { name: 'Hex', level: 1, preparation: 'known' },
                ] },
                { kind: 'item', name: 'Ring of Jaedear', spells: [{ name: 'Shield', level: 1, preparation: 'limited' }] },
              ],
              slots: [3],
              pact_magic: { slot_level: 1, total: 1 },
            },
          }),
        }),
      },
    })
    renderPage()

    await screen.findByText('Find Familiar')
    expect(screen.getByText('★ Always prepared')).toBeInTheDocument()
    expect(screen.getByText('● Prepared')).toBeInTheDocument()
    expect(screen.getByText('○ Not prepared')).toBeInTheDocument()
    expect(screen.getByText('Limited use')).toBeInTheDocument()
    // a plain known spell carries no marker, and the raw preparation value is not repeated as a field
    expect(screen.queryByText('known')).not.toBeInTheDocument()
    expect(screen.getByText('Ring of Jaedear')).toBeInTheDocument()
    // regular slots and Pact Magic are shown separately
    expect(screen.getByText('Level 1 slots: 3')).toBeInTheDocument()
    expect(screen.getByText('Pact Magic')).toBeInTheDocument()
  })

  it('shows a magic item together with the limited uses that name it as their source', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({
          sheet: sheetOf('pc1', {
            magic_items: [{ name: 'Ring of Jaedear', stage: 'Level 1', attuned: true }],
            limited_uses: [
              { name: 'Shield', max: '1', recovery: 'dawn', source: 'Ring of Jaedear' },
              { name: 'Fey Step', max: '2', recovery: 'long rest' },
            ],
          }),
        }),
      },
    })
    renderPage()

    expect(await screen.findByText('Uses: Shield — 1 (dawn)')).toBeInTheDocument()
    expect(screen.queryByText(/Uses: Fey Step/)).not.toBeInTheDocument()
    expect(screen.getByText('Level 1')).toBeInTheDocument()
  })

  it('shows each companion\'s creature sheet under its own name', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({
          companions: [{ pc: rex, sheet: sheetOf('c1', { revision: 2, creature: { ac: '13+Prof', speed_text: '40 ft, climb 30 ft, burrow 15 ft' } }) }],
        }),
      },
    })
    renderPage()

    expect(await screen.findByRole('heading', { name: 'Companions' })).toBeInTheDocument()
    expect(screen.getByText('13+Prof')).toBeInTheDocument()
    expect(screen.getByText('40 ft, climb 30 ft, burrow 15 ft')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit Rex' })).toBeInTheDocument()
  })

  it('shows only the creature section when opened on a companion', async () => {
    stubFetch({
      'GET /api/pcs/c1/sheet': { body: response({ pc: rex, sheet: sheetOf('c1', { creature: { ac: '12' } }) }) },
    })
    renderPage('/characters/c1/sheet')

    expect(await screen.findByText('12')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit Companion' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit Spellcasting' })).not.toBeInTheDocument()
  })
})

describe('CharacterSheetPage — editing', () => {
  it('saves a section with the sheet revision and then reloads', async () => {
    let revision = 0
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': () => ({ body: response({ sheet: sheetOf('pc1', { revision }) }) }),
      'PUT /api/pcs/pc1/sheet/notes': () => { revision = 1; return { body: { revision: 1 } } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Notes' }))
    await userEvent.type(screen.getByLabelText('Personality'), 'Curious')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/notes')).toHaveLength(1))
    const put = putsTo(calls, '/api/pcs/pc1/sheet/notes')[0]
    expect(put.body).toMatchObject({ revision: 0, data: { personality: 'Curious' } })
    // closes the editor and reloads the sheet
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument())
    expect(calls.filter(c => c.method === 'GET')).toHaveLength(2)
  })

  it('cancels without saving', async () => {
    const calls = stubFetch({ 'GET /api/pcs/pc1/sheet': { body: response() } })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Notes' }))
    await userEvent.type(screen.getByLabelText('Personality'), 'Curious')
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByLabelText('Personality')).not.toBeInTheDocument()
    expect(calls.some(c => c.method === 'PUT')).toBe(false)
  })

  it('edits identity with the PC\'s current name and sends the new one', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/identity': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Identity' }))
    const name = screen.getByLabelText('Name')
    expect(name).toHaveValue('Aria')
    await userEvent.clear(name)
    await userEvent.type(name, 'Aria Sunsong')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/identity')).toHaveLength(1))
    expect(putsTo(calls, '/api/pcs/pc1/sheet/identity')[0].body).toMatchObject({ data: { name: 'Aria Sunsong' } })
  })

  it('edits combat with the PC\'s current max HP, as typed — nothing is recalculated', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response({ sheet: sheetOf('pc1', { combat: { ac: 13 } }) }) },
      'PUT /api/pcs/pc1/sheet/combat': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Combat' }))
    expect(screen.getByText(/nothing is recalculated/)).toBeInTheDocument()
    const hp = screen.getByLabelText('Max HP')
    expect(hp).toHaveValue(16)
    await userEvent.clear(hp)
    await userEvent.type(hp, '21')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/combat')).toHaveLength(1))
    expect(putsTo(calls, '/api/pcs/pc1/sheet/combat')[0].body).toMatchObject({ data: { max_hp: 21, ac: 13 } })
  })

  it('adds, fills and removes rows in a list section, sending a bare list', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/attacks': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Attacks' }))
    await userEvent.click(screen.getByRole('button', { name: '+ Add attack' }))
    await userEvent.click(screen.getByRole('button', { name: '+ Add attack' }))
    const first = within(screen.getByRole('group', { name: 'attack 1' }))
    await userEvent.type(first.getByLabelText('Name'), 'Sting')
    await userEvent.type(first.getByLabelText('To hit'), 'DC 12')
    await userEvent.click(screen.getByRole('button', { name: 'Remove attack 2' }))
    expect(screen.queryByRole('group', { name: 'attack 2' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/attacks')).toHaveLength(1))
    const data = (putsTo(calls, '/api/pcs/pc1/sheet/attacks')[0].body as { data: unknown[] }).data
    expect(data).toHaveLength(1)
    expect(data[0]).toMatchObject({ name: 'Sting', to_hit: 'DC 12' })
  })

  it('edits spellcasting: a spell\'s preparation, slot totals, and no empty Pact Magic', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/spellcasting': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Spellcasting' }))
    await userEvent.click(screen.getByRole('button', { name: '+ Add spell list' }))
    const list = within(screen.getByRole('group', { name: 'spell list 1' }))
    await userEvent.type(list.getAllByLabelText('Name')[0], 'Druid')
    await userEvent.click(list.getByRole('button', { name: '+ Add spell' }))
    const spell = within(screen.getByRole('group', { name: 'spell 1' }))
    await userEvent.type(spell.getByLabelText('Name'), 'Find Familiar')
    await userEvent.clear(spell.getByLabelText('Level'))
    await userEvent.type(spell.getByLabelText('Level'), '1')
    await userEvent.selectOptions(spell.getByLabelText('Preparation'), 'always')
    await userEvent.type(screen.getByLabelText('Level 1 slots'), '3')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/spellcasting')).toHaveLength(1))
    const data = (putsTo(calls, '/api/pcs/pc1/sheet/spellcasting')[0].body as { data: Record<string, unknown> }).data
    expect(data.slots).toEqual([3])
    expect(data).not.toHaveProperty('pact_magic')
    expect((data.spell_lists as Record<string, unknown>[])[0]).toMatchObject({
      kind: 'class', name: 'Druid',
      spells: [{ name: 'Find Familiar', level: 1, preparation: 'always' }],
    })
  })

  it('edits a proficiency\'s pending flag and keeps untouched fields of existing rows', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({ sheet: sheetOf('pc1', { magic_items: [{ name: 'Ring of Jaedear', catalog_item_id: 'cat-1', attuned: true }] }) }),
      },
      'PUT /api/pcs/pc1/sheet/magic_items': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByText('Ring of Jaedear')

    await userEvent.click(screen.getByRole('button', { name: 'Edit Magic Items' }))
    await userEvent.type(screen.getByLabelText('Stage / level'), 'Level 1')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/magic_items')).toHaveLength(1))
    const data = (putsTo(calls, '/api/pcs/pc1/sheet/magic_items')[0].body as { data: Record<string, unknown>[] }).data
    expect(data[0]).toMatchObject({ name: 'Ring of Jaedear', stage: 'Level 1', catalog_item_id: 'cat-1', attuned: true })
  })

  it('saves a companion\'s creature section against the companion, with its own revision', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({ sheet: sheetOf('pc1', { revision: 7 }), companions: [{ pc: rex, sheet: sheetOf('c1', { revision: 2, creature: { ac: '12' } }) }] }),
      },
      'PUT /api/pcs/c1/sheet/creature': { body: { revision: 3 } },
    })
    renderPage()
    await screen.findByRole('button', { name: 'Edit Rex' })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Rex' }))
    const ac = screen.getByLabelText('AC')
    await userEvent.clear(ac)
    await userEvent.type(ac, '13+Prof')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/c1/sheet/creature')).toHaveLength(1))
    expect(putsTo(calls, '/api/pcs/c1/sheet/creature')[0].body).toMatchObject({ revision: 2, data: { ac: '13+Prof' } })
  })

  it('edits wild shape forms with the shared creature fields', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/wild_shapes': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Wild Shapes' }))
    await userEvent.type(screen.getByLabelText('Limitations'), 'No Fly Speed')
    await userEvent.click(screen.getByRole('button', { name: '+ Add form' }))
    const form = within(screen.getByRole('group', { name: 'form 1' }))
    await userEvent.type(form.getAllByLabelText('Name')[0], 'Rat')
    await userEvent.type(form.getByLabelText('AC'), '10')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/wild_shapes')).toHaveLength(1))
    const data = (putsTo(calls, '/api/pcs/pc1/sheet/wild_shapes')[0].body as { data: Record<string, unknown> }).data
    expect(data.limitations).toBe('No Fly Speed')
    expect((data.forms as Record<string, unknown>[])[0]).toMatchObject({ name: 'Rat', ac: '10' })
  })
})

describe('CharacterSheetPage — save failures', () => {
  it('on a revision conflict keeps the input and offers a reload, then saves with the new revision', async () => {
    let revision = 1
    let putStatus = 409
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': () => ({ body: response({ sheet: sheetOf('pc1', { revision }) }) }),
      'PUT /api/pcs/pc1/sheet/notes': () => ({ status: putStatus, body: { revision: revision + 1 } }),
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Notes' }))
    await userEvent.type(screen.getByLabelText('Personality'), 'Curious')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('changed elsewhere')
    expect(screen.getByLabelText('Personality')).toHaveValue('Curious')

    revision = 5 // somebody else saved in the meantime
    putStatus = 200
    await userEvent.click(screen.getByRole('button', { name: 'Reload latest' }))
    await waitFor(() => expect(calls.filter(c => c.method === 'GET')).toHaveLength(2))
    expect(screen.getByLabelText('Personality')).toHaveValue('Curious')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/notes')).toHaveLength(2))
    expect(putsTo(calls, '/api/pcs/pc1/sheet/notes')[1].body).toMatchObject({ revision: 5 })
  })

  it('shows the server\'s message when a save is rejected and keeps the editor open', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/combat': { status: 400, text: 'max_hp must be greater than 0' },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Combat' }))
    await userEvent.clear(screen.getByLabelText('Max HP'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('max_hp must be greater than 0')
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})

describe('CharacterSheetPage — remaining sections', () => {
  it('shows attacks, actions, features, weapon masteries, skills and pending proficiencies', async () => {
    stubFetch({
      'GET /api/pcs/pc1/sheet': {
        body: response({
          sheet: sheetOf('pc1', {
            attacks: [{ name: 'Sting', ability: 'Dex', range: 'Melee (5 ft)', to_hit: 'DC 12', damage: '2d4+2', damage_type: 'Poison', notes: 'Con save' }],
            actions: { bonus_action: [{ name: 'Toma wacho!', source: 'Kid Tricks (feat)', is_default: false }] },
            features: [{ kind: 'feat', name: 'Alert', text: 'Add proficiency to initiative.', source: 'PHB', level: 1 }],
            weapon_masteries: [{ weapon: 'Sling', mastery: 'Slow' }],
            proficiencies: { languages: [{ name: 'from Human', pending: true }] },
            skills: { entries: [{ key: 'Stealth', ability: 'Dex', bonus: 4, proficient: true, expertise: true }], passive_perception: 16 },
            defenses: { vision: 'Darkvision 60 ft', resistances: ['Fire', 'Cold'] },
            notes: { personality: 'Curious', backstory: 'Raised by dryads.' },
          }),
        }),
      },
    })
    renderPage()

    expect(await screen.findByText('Sting')).toBeInTheDocument()
    expect(screen.getByText('DC 12')).toBeInTheDocument()
    expect(screen.getByText('Poison')).toBeInTheDocument()
    expect(screen.getByText('Toma wacho!')).toBeInTheDocument()
    expect(screen.getByText('Kid Tricks (feat)')).toBeInTheDocument()
    expect(screen.getByText('Alert')).toBeInTheDocument()
    expect(screen.getByText('Add proficiency to initiative.')).toBeInTheDocument()
    expect(screen.getByText('Sling')).toBeInTheDocument()
    expect(screen.getByText('Slow')).toBeInTheDocument()
    expect(screen.getByText('Choice pending')).toBeInTheDocument()
    expect(screen.getByText('Stealth')).toBeInTheDocument()
    expect(screen.getByText('Passive Perception')).toBeInTheDocument()
    expect(screen.getByText('Fire, Cold')).toBeInTheDocument()
    expect(screen.getByText('Raised by dryads.')).toBeInTheDocument()
  })

  it('edits abilities and defenses, sending only what was filled in', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/abilities': { body: { revision: 1 } },
      'PUT /api/pcs/pc1/sheet/defenses': { body: { revision: 2 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Abilities & Saves' }))
    expect(screen.getByText(/nothing is recalculated/)).toBeInTheDocument()
    const str = within(screen.getByRole('group', { name: 'Strength' }))
    await userEvent.type(str.getByLabelText('Score'), '18')
    await userEvent.type(str.getByLabelText('Modifier'), '-1') // deliberately not 18's real modifier
    await userEvent.click(str.getByLabelText('Save proficient'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/abilities')).toHaveLength(1))
    const abilities = (putsTo(calls, '/api/pcs/pc1/sheet/abilities')[0].body as { data: Record<string, unknown> }).data
    expect(abilities.str).toMatchObject({ score: 18, mod: -1, save_proficient: true })
    expect(abilities).not.toHaveProperty('dex')

    await userEvent.click(await screen.findByRole('button', { name: 'Edit Defenses' }))
    await userEvent.type(screen.getByLabelText('Resistances (one per line)'), 'Fire{Enter}Cold')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/defenses')).toHaveLength(1))
    expect(putsTo(calls, '/api/pcs/pc1/sheet/defenses')[0].body).toMatchObject({ data: { resistances: ['Fire', 'Cold'] } })
  })

  it.each([
    {
      title: 'Actions', section: 'actions', add: '+ Add bonus action', group: 'bonus action 1',
      fill: { Name: 'Toma wacho!', Source: 'Kid Tricks (feat)' },
      expected: (d: Record<string, unknown>) => expect((d.bonus_action as unknown[])[0]).toMatchObject({ name: 'Toma wacho!', source: 'Kid Tricks (feat)', is_default: false }),
    },
    {
      title: 'Features & Feats', section: 'features', add: '+ Add feature', group: 'feature 1',
      fill: { Name: 'Alert', Text: 'Initiative bonus.' },
      expected: (d: unknown) => expect((d as unknown[])[0]).toMatchObject({ kind: 'class', name: 'Alert', text: 'Initiative bonus.' }),
    },
    {
      title: 'Weapon Masteries', section: 'weapon_masteries', add: '+ Add weapon mastery', group: 'weapon mastery 1',
      fill: { Weapon: 'Sling', Mastery: 'Slow' },
      expected: (d: unknown) => expect((d as unknown[])[0]).toMatchObject({ weapon: 'Sling', mastery: 'Slow' }),
    },
    {
      title: 'Limited Uses', section: 'limited_uses', add: '+ Add limited use', group: 'limited use 1',
      fill: { Name: 'Balm', 'Max uses': '2d6', Recovery: 'LR/SS' },
      expected: (d: unknown) => expect((d as unknown[])[0]).toMatchObject({ name: 'Balm', max: '2d6', recovery: 'LR/SS' }),
    },
    {
      title: 'Proficiencies', section: 'proficiencies', add: '+ Add language', group: 'language 1',
      fill: { Name: 'from Human' },
      expected: (d: Record<string, unknown>) => expect((d.languages as unknown[])[0]).toMatchObject({ name: 'from Human', pending: false }),
    },
    {
      title: 'Skills', section: 'skills', add: '+ Add skill', group: 'skill 1',
      fill: { Skill: 'Stealth', Bonus: '4' },
      expected: (d: Record<string, unknown>) => expect((d.entries as unknown[])[0]).toMatchObject({ key: 'Stealth', bonus: 4 }),
    },
  ])('adds a row to $title and saves it to /sheet/$section', async ({ title, section, add, group, fill, expected }) => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      [`PUT /api/pcs/pc1/sheet/${section}`]: { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: `Edit ${title}` }))
    await userEvent.click(screen.getByRole('button', { name: add }))
    const row = within(screen.getByRole('group', { name: group }))
    for (const [label, text] of Object.entries(fill)) {
      await userEvent.type(row.getByLabelText(label), text)
    }
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, `/api/pcs/pc1/sheet/${section}`)).toHaveLength(1))
    expected((putsTo(calls, `/api/pcs/pc1/sheet/${section}`)[0].body as { data: never }).data)
  })

  it('marks a proficiency choice as pending', async () => {
    const calls = stubFetch({
      'GET /api/pcs/pc1/sheet': { body: response() },
      'PUT /api/pcs/pc1/sheet/proficiencies': { body: { revision: 1 } },
    })
    renderPage()
    await screen.findByRole('heading', { name: /Aria/ })

    await userEvent.click(screen.getByRole('button', { name: 'Edit Proficiencies' }))
    await userEvent.click(screen.getByRole('button', { name: '+ Add tool' }))
    const row = within(screen.getByRole('group', { name: 'tool 1' }))
    await userEvent.type(row.getByLabelText('Name'), 'Trance: tool or weapon')
    await userEvent.click(row.getByLabelText('Choice pending'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(putsTo(calls, '/api/pcs/pc1/sheet/proficiencies')).toHaveLength(1))
    const data = (putsTo(calls, '/api/pcs/pc1/sheet/proficiencies')[0].body as { data: { tools: unknown[] } }).data
    expect(data.tools[0]).toMatchObject({ name: 'Trance: tool or weapon', pending: true })
  })
})
