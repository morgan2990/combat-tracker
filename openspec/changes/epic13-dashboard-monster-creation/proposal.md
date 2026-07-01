## Why

Custom monster creation currently only opens from inside an active combat room (DMView's header button), which forces a DM to spin up a room just to prep homebrew creatures. Epic 13 moves that entry point to the main dashboard, alongside the existing "+ New Character" pattern for players, so DMs can prepare monsters ahead of a session.

## What Changes

- Remove the `+ Monster` button and its `navigate('/monsters/new')` handler from the DMView combat room header (DMView.tsx). Drop the `useNavigate` import there if nothing else in the file uses it.
- Add a `+ New Monster` link to the Dashboard's "As DM" panel, styled and positioned the same way as the existing `+ New Character` link in the "As Player" panel. It links to the existing `/monsters/new` route.
- On MonsterForm's post-save confirmation screen, rename the "Back to Join" button to "Back to Dashboard" (label only — it already calls `navigate('/')`). The "Add Another" button is unchanged, preserving the ability to batch-create several monsters in one sitting.

Not changing: the `/monsters/new` route registration in App.tsx, the `POST /api/monsters` request shape (edition, initiative_modifier, multipart PDF path), or any backend/MongoDB/Typesense behavior — all established in prior epics. No toast/notification component is being introduced; the "Back to Dashboard" button satisfies the redirect-to-dashboard need without new UI infrastructure.

## Capabilities

### New Capabilities
- `dashboard-monster-creation`: Defines the dashboard's "As DM" entry point for monster creation, the removal of the in-room entry point, and the post-save exit path back to the dashboard.

### Modified Capabilities
_None._ The `monster-form` capability's request/response contract (edition selector, initiative modifier, multipart PDF upload) is unchanged — only the surrounding entry/exit navigation moves, which is covered by the new capability above.

## Impact

- **Frontend files**: `frontend/src/components/DMView.tsx` (remove button + navigate call), `frontend/src/components/Dashboard.tsx` (add link), `frontend/src/components/MonsterForm.tsx` (button label only).
- **No backend, database, or search-index changes.**
- **No route changes** in `frontend/src/App.tsx` — `/monsters/new` already resolves to `MonsterForm`.
