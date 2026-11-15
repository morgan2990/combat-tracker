import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { Dashboard } from './Dashboard'
import type { ImportPreview, MeResponse } from '../types'
import { useLayoutTier } from '../hooks/useLayoutTier'

vi.mock('../hooks/useLayoutTier')

afterEach(() => {
  vi.unstubAllGlobals()
  vi.mocked(useLayoutTier).mockReset()
  vi.restoreAllMocks()
})

const me: MeResponse = {
  user: { username: 'aria', display_name: 'Aria' },
  rooms: [], pcs: [], recent_rooms: [], parties: [],
}

const preview: ImportPreview = {
  import_id: 'imp1', new_character: true, name: 'Puck', max_hp: 21, source: {},
  warnings: [], changes: [], companions: { removed: [], added: [] },
  inventory: { action: 'keep', changed: false, old_items: 0, new_items: 0 },
}

function stubFetch() {
  const calls: string[] = []
  vi.stubGlobal('fetch', vi.fn((url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${url}`
    calls.push(key)
    let body: unknown = []
    if (key === 'POST /api/sheet-imports') body = preview
    if (key === 'POST /api/sheet-imports/imp1/confirm') body = { pc_id: 'new1', revision: 1 }
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body), text: () => Promise.resolve('') })
  }))
  return calls
}

describe('Dashboard — new character from PDF', () => {
  it('opens the new sheet and refreshes the list after confirming', async () => {
    vi.mocked(useLayoutTier).mockReturnValue('desktop')
    const calls = stubFetch()
    const onRefresh = vi.fn()
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<Dashboard me={me} onOpenRoomAsDM={vi.fn()} onJoinAsPlayer={vi.fn()} onLogout={vi.fn()} onRefresh={onRefresh} />} />
          <Route path="/characters/:id/sheet" element={<div>sheet page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await userEvent.click(screen.getByRole('button', { name: 'New character from PDF' }))
    await userEvent.upload(screen.getByLabelText(/MPMB character sheet/), new File(['%PDF'], 'puck.pdf', { type: 'application/pdf' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Create character' }))

    expect(await screen.findByText('sheet page')).toBeInTheDocument()
    expect(onRefresh).toHaveBeenCalled()
    expect(calls).toContain('POST /api/sheet-imports')
  })

  it('discards the draft and stays on the Dashboard when cancelled', async () => {
    vi.mocked(useLayoutTier).mockReturnValue('desktop')
    const calls = stubFetch()
    render(
      <MemoryRouter>
        <Dashboard me={me} onOpenRoomAsDM={vi.fn()} onJoinAsPlayer={vi.fn()} onLogout={vi.fn()} />
      </MemoryRouter>,
    )

    await userEvent.click(screen.getByRole('button', { name: 'New character from PDF' }))
    await userEvent.upload(screen.getByLabelText(/MPMB character sheet/), new File(['%PDF'], 'puck.pdf', { type: 'application/pdf' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(calls).toContain('DELETE /api/sheet-imports/imp1')
  })
})
