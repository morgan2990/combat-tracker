import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ImportSheetDialog } from './ImportSheetDialog'
import type { ImportPreview } from '../types'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function previewOf(over: Partial<ImportPreview> = {}): ImportPreview {
  return {
    import_id: 'imp1', new_character: false, name: 'Puck', max_hp: 21, source: { file_name: 'puck.pdf' },
    warnings: [], changes: [], companions: { removed: [], added: [] },
    inventory: { action: 'keep', changed: false, old_items: 0, new_items: 0 },
    ...over,
  }
}

interface Reply { status?: number; body?: unknown; text?: string }

function stubFetch(routes: Record<string, Reply>) {
  const calls: { method: string; url: string; body?: unknown }[] = []
  vi.stubGlobal('fetch', vi.fn((url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ method, url, body: init?.body })
    const r = routes[`${method} ${url}`] ?? { status: 404 }
    const status = r.status ?? 200
    return Promise.resolve({
      ok: status >= 200 && status < 300, status,
      json: () => Promise.resolve(r.body ?? {}), text: () => Promise.resolve(r.text ?? ''),
    })
  }))
  return calls
}

function renderDialog(props: Partial<Parameters<typeof ImportSheetDialog>[0]> = {}) {
  const onClose = vi.fn()
  const onConfirmed = vi.fn()
  render(<ImportSheetDialog title="Import from PDF" uploadUrl="/api/pcs/pc1/sheet/import" onClose={onClose} onConfirmed={onConfirmed} {...props} />)
  return { onClose, onConfirmed }
}

const pdf = () => new File(['%PDF'], 'puck.pdf', { type: 'application/pdf' })

async function choose(file: File = pdf()) {
  await userEvent.upload(screen.getByLabelText(/MPMB character sheet|Upload again/), file)
}

describe('ImportSheetDialog', () => {
  it('uploads the chosen file as multipart form data', async () => {
    const calls = stubFetch({ 'POST /api/pcs/pc1/sheet/import': { body: previewOf() } })
    renderDialog()
    await choose()

    expect(await screen.findByText('No differences.')).toBeInTheDocument()
    expect(calls[0].method).toBe('POST')
    expect((calls[0].body as FormData).get('file')).toBeInstanceOf(File)
  })

  it('lists warnings and differences by section before the confirm action', async () => {
    stubFetch({ 'POST /api/pcs/pc1/sheet/import': { body: previewOf({
      warnings: ['Max HP is missing'],
      changes: [
        { section: 'combat', path: 'ac', kind: 'changed', old: 14, new: 16 },
        { section: 'attacks', path: '[Sting]', kind: 'removed', old: { name: 'Sting' } },
      ],
      companions: { removed: ['Old Pet'], added: ['Rex'] },
      inventory: { action: 'replace', changed: true, old_items: 1, new_items: 3 },
    }) } })
    renderDialog()
    await choose()

    expect(await screen.findByText('Max HP is missing')).toBeInTheDocument()
    expect(screen.getByLabelText('Changes in combat')).toHaveTextContent('ac: 14 → 16')
    expect(screen.getByLabelText('Changes in attacks')).toHaveTextContent('Removed [Sting]')
    expect(screen.getByLabelText('Changes in companions')).toHaveTextContent('Removed: Old Pet')
    expect(screen.getByLabelText('Changes in companions')).toHaveTextContent('Added: Rex')
    expect(screen.getByText(/1 → 3 items/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Replace sheet' })).toBeInTheDocument()
  })

  it('confirms and reports the result', async () => {
    const calls = stubFetch({
      'POST /api/pcs/pc1/sheet/import': { body: previewOf() },
      'POST /api/sheet-imports/imp1/confirm': { body: { pc_id: 'pc1', revision: 3 } },
    })
    const { onConfirmed } = renderDialog()
    await choose()
    await userEvent.click(await screen.findByRole('button', { name: 'Replace sheet' }))

    await waitFor(() => expect(onConfirmed).toHaveBeenCalledWith({ pc_id: 'pc1', revision: 3 }))
    expect(calls.map(c => `${c.method} ${c.url}`)).toEqual(['POST /api/pcs/pc1/sheet/import', 'POST /api/sheet-imports/imp1/confirm'])
  })

  it('offers to create the character for a new-character preview', async () => {
    stubFetch({ 'POST /api/sheet-imports': { body: previewOf({ new_character: true }) } })
    renderDialog({ uploadUrl: '/api/sheet-imports' })
    await choose()
    expect(await screen.findByRole('button', { name: 'Create character' })).toBeInTheDocument()
  })

  it('discards the draft when cancelled after the preview', async () => {
    const calls = stubFetch({
      'POST /api/pcs/pc1/sheet/import': { body: previewOf() },
      'DELETE /api/sheet-imports/imp1': { status: 204 },
    })
    const { onClose } = renderDialog()
    await choose()
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    expect(onClose).toHaveBeenCalled()
    expect(calls.some(c => c.method === 'DELETE' && c.url === '/api/sheet-imports/imp1')).toBe(true)
  })

  it('closes without a request when cancelled before uploading', async () => {
    const calls = stubFetch({})
    const { onClose } = renderDialog()
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalled()
    expect(calls).toHaveLength(0)
  })

  it('shows the server message for a rejected file and lets the user pick again', async () => {
    stubFetch({ 'POST /api/pcs/pc1/sheet/import': { status: 400, text: 'this PDF is not an MPMB character sheet' } })
    renderDialog()
    await choose()

    expect(await screen.findByRole('alert')).toHaveTextContent('this PDF is not an MPMB character sheet')
    expect(screen.getByLabelText(/MPMB character sheet/)).toBeEnabled()
  })

  it('asks to retry on a 429', async () => {
    stubFetch({ 'POST /api/pcs/pc1/sheet/import': { status: 429, text: 'busy' } })
    renderDialog()
    await choose()
    expect(await screen.findByRole('alert')).toHaveTextContent(/try again in a moment/i)
  })

  it('says the sheet changed on a 409 and offers to upload again', async () => {
    const calls = stubFetch({
      'POST /api/pcs/pc1/sheet/import': { body: previewOf() },
      'POST /api/sheet-imports/imp1/confirm': { status: 409, text: 'conflict' },
      'DELETE /api/sheet-imports/imp1': { status: 204 },
    })
    const { onConfirmed } = renderDialog()
    await choose()
    await userEvent.click(await screen.findByRole('button', { name: 'Replace sheet' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/sheet changed/i)
    expect(screen.getByLabelText('Upload again')).toBeEnabled()
    expect(onConfirmed).not.toHaveBeenCalled()
    await waitFor(() => expect(calls.some(c => c.method === 'DELETE')).toBe(true))
  })

  it('keeps the preview and shows the message when confirm fails otherwise', async () => {
    stubFetch({
      'POST /api/pcs/pc1/sheet/import': { body: previewOf() },
      'POST /api/sheet-imports/imp1/confirm': { status: 500, text: 'database error' },
    })
    renderDialog()
    await choose()
    await userEvent.click(await screen.findByRole('button', { name: 'Replace sheet' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('database error')
    expect(screen.getByRole('button', { name: 'Replace sheet' })).toBeEnabled()
  })
})
