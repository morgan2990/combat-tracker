import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { CharacterSheetPage } from './CharacterSheetPage'
import type { ImportPreview, PC, SheetResponse } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const aria: PC = {
  id: 'pc1', owner_user_id: 'u1', name: 'Aria', type: 'pc', max_hp: 16, shares_initiative: false,
  items: [], currency: { pp: 0, gp: 0, ep: 0, sp: 0, cp: 0 },
}
const rex: PC = { ...aria, id: 'c1', name: 'Rex', type: 'companion', parent_pc_id: 'pc1' }

const preview: ImportPreview = {
  import_id: 'imp1', new_character: false, name: 'Puck', max_hp: 21, source: {},
  warnings: [], changes: [], companions: { removed: [], added: [] },
  inventory: { action: 'keep', changed: false, old_items: 0, new_items: 0 },
}

function stubFetch(pc: PC) {
  let sheetLoads = 0
  const calls: string[] = []
  vi.stubGlobal('fetch', vi.fn((url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${url}`
    calls.push(key)
    let body: unknown = {}
    if (key.startsWith('GET') && url.endsWith('/sheet')) {
      sheetLoads++
      body = { pc: sheetLoads > 1 ? { ...pc, name: 'Puck' } : pc, sheet: { pc_id: pc.id, schema_version: 1, revision: sheetLoads }, companions: [] } satisfies SheetResponse
    } else if (key === 'POST /api/pcs/pc1/sheet/import') {
      body = preview
    } else if (key === 'POST /api/sheet-imports/imp1/confirm') {
      body = { pc_id: 'pc1', revision: 2 }
    }
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body), text: () => Promise.resolve('') })
  }))
  return calls
}

function renderPage(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes><Route path="/characters/:id/sheet" element={<CharacterSheetPage />} /></Routes>
    </MemoryRouter>,
  )
}

describe('CharacterSheetPage — import', () => {
  it('offers the import action to the owner of a top-level PC', async () => {
    stubFetch(aria)
    renderPage('/characters/pc1/sheet')
    expect(await screen.findByRole('button', { name: 'Import from PDF' })).toBeInTheDocument()
  })

  it('reloads the sheet after confirming', async () => {
    const calls = stubFetch(aria)
    renderPage('/characters/pc1/sheet')
    await userEvent.click(await screen.findByRole('button', { name: 'Import from PDF' }))
    await userEvent.upload(screen.getByLabelText(/MPMB character sheet/), new File(['%PDF'], 'puck.pdf', { type: 'application/pdf' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Replace sheet' }))

    expect(await screen.findByRole('heading', { name: /Puck/ })).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(calls.filter(c => c === 'GET /api/pcs/pc1/sheet')).toHaveLength(2)
  })

  it('does not offer an import in the DM view', async () => {
    stubFetch(aria)
    renderPage('/characters/pc1/sheet?room=R1')
    await screen.findByRole('heading', { name: /Aria/ })
    expect(screen.queryByRole('button', { name: 'Import from PDF' })).not.toBeInTheDocument()
  })

  it('does not offer an import on a companion', async () => {
    stubFetch(rex)
    renderPage('/characters/c1/sheet')
    await screen.findByRole('heading', { name: /Rex/ })
    expect(screen.queryByRole('button', { name: 'Import from PDF' })).not.toBeInTheDocument()
  })
})
