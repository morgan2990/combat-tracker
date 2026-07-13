import { afterEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CollapsibleSection } from './CollapsibleSection'

afterEach(() => {
  localStorage.clear()
})

describe('CollapsibleSection', () => {
  it('renders the title, count, and children expanded by default', () => {
    render(
      <CollapsibleSection id="dm.rooms" title="Rooms" count={3}>
        <div>child content</div>
      </CollapsibleSection>,
    )
    expect(screen.getByText('Rooms')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('child content')).toBeInTheDocument()
    expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'true')
  })

  it('collapses on toggle click and hides children', async () => {
    render(
      <CollapsibleSection id="dm.rooms-collapse" title="Rooms" count={3}>
        <div>child content</div>
      </CollapsibleSection>,
    )

    await userEvent.click(screen.getByRole('button'))

    expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('child content')).not.toBeInTheDocument()
  })

  it('persists collapsed state to localStorage keyed by id', async () => {
    render(
      <CollapsibleSection id="dm.persist-test" title="Rooms" count={0}>
        <div>child</div>
      </CollapsibleSection>,
    )

    await userEvent.click(screen.getByRole('button'))

    expect(localStorage.getItem('collapsible-section.dm.persist-test')).toBe('false')
  })

  it('reads a previously-persisted collapsed state on mount', () => {
    localStorage.setItem('collapsible-section.dm.restore-test', 'false')

    render(
      <CollapsibleSection id="dm.restore-test" title="Rooms" count={0}>
        <div>child</div>
      </CollapsibleSection>,
    )

    expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'false')
  })

  it('renders headerAction next to the toggle', () => {
    render(
      <CollapsibleSection id="dm.header-action" title="Rooms" count={0} headerAction={<button>+ New</button>}>
        <div>child</div>
      </CollapsibleSection>,
    )
    expect(screen.getByRole('button', { name: '+ New' })).toBeInTheDocument()
  })
})
