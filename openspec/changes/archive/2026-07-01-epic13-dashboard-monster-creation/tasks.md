## 1. Remove in-room entry point

- [x] 1.1 Remove the `+ Monster` button (and its `onClick={() => navigate('/monsters/new')}` handler) from the DM panel header in `frontend/src/components/DMView.tsx`
- [x] 1.2 Check whether `useNavigate` is still used elsewhere in `DMView.tsx`; remove the import and the `navigate` variable if it's now dead

## 2. Add dashboard entry point

- [x] 2.1 Add a `+ New Monster` `Link` to `/monsters/new` inside the "As DM" panel in `frontend/src/components/Dashboard.tsx`, styled/positioned to match the existing `+ New Character` link in the "As Player" panel

## 3. Update post-save screen copy

- [x] 3.1 In `frontend/src/components/MonsterForm.tsx`, rename the "Back to Join" button to "Back to Dashboard" (keep its existing `navigate('/')` behavior unchanged)
- [x] 3.2 Confirm the "Add Another" button's reset behavior is untouched

## 4. Verify

- [x] 4.1 Manually verify: dashboard shows `+ New Monster` in "As DM" panel and navigates to `/monsters/new`
- [x] 4.2 Manually verify: an active combat room's DM panel no longer shows any monster-creation control
- [x] 4.3 Manually verify: saving a monster, clicking "Add Another" keeps the user on `/monsters/new` with a blank form; clicking "Back to Dashboard" navigates to `/`
- [x] 4.4 Run frontend type-check/build to confirm no dangling references to removed code
