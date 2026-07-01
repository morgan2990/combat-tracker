## Context

Monster creation (`MonsterForm.tsx`, `POST /api/monsters`) already works end-to-end from Epics 7 and 10 — MongoDB persistence, Typesense indexing, edition selector, initiative modifier, and multipart PDF upload are all in place. The only thing this change relocates is the *entry point*: today it's a button inside `DMView.tsx`'s combat room header; it needs to move to the main `Dashboard.tsx`, next to the existing "+ New Character" pattern for players, so a DM can prep homebrew monsters without opening a room.

This is a small, mostly mechanical UI change (three files, no new state, no new route, no backend touch), included here mainly to record the two decisions made while scoping it, so they don't get re-litigated during implementation.

## Goals / Non-Goals

**Goals:**
- Move the monster-creation entry point from the in-room DM panel to the dashboard's "As DM" card.
- Preserve the ability to batch-create several monsters in one sitting (a real DM-prep workflow).
- Land AC1-AC5 of Epic 13 fully; land the redirect-to-dashboard part of AC6 without a toast.

**Non-Goals:**
- Building a toast/notification primitive. None exists anywhere in this codebase today, and Epic 13 is not a strong enough reason to introduce one — see Decisions below.
- Changing the `/monsters/new` route, the `MonsterForm` submission payload, or any backend/Mongo/Typesense behavior.

## Decisions

**Keep "Add Another" on MonsterForm's save-confirmation screen.**
The dashboard-relocation is explicitly meant to let DMs prep multiple homebrew monsters ahead of a session. Forcing a full redirect back to the dashboard after every single monster would undercut that workflow. Alternative considered: auto-redirect after every save (matches `CharacterForm`'s pattern exactly) — rejected because character creation and monster creation have different usage shapes (a DM rarely creates a dozen PCs in one sitting, but plausibly creates a dozen monsters for an encounter).

**Rename "Back to Join" → "Back to Dashboard"; skip the toast entirely.**
AC6 asks for a redirect to the dashboard with a "temporary success notification toast or message." The redirect half is satisfied by an explicit button (already wired to `navigate('/')`) rather than an automatic redirect, consistent with keeping "Add Another" as a real choice rather than a race against an auto-navigate timer. The toast half is dropped: no toast/notification component exists anywhere in the app yet (confirmed by search), and introducing one for a single button label fix is disproportionate. Alternatives considered:
- `location.state` flag read by `Dashboard` on mount — viable but adds a new cross-component contract for a one-off cosmetic confirmation.
- A reusable `<Toast>` component — reasonable *if* a second consumer (e.g. character save) shows up later, but speculative today. Revisit if another epic asks for the same thing.

## Risks / Trade-offs

- **AC6 is only partially implemented** (redirect yes, toast no). This is a deliberate scope reduction agreed during exploration, not an oversight — flagged here so it's visible at review/archive time rather than silently dropped.
- **Dead code risk**: removing the DMView header button may leave `useNavigate` unused in `DMView.tsx` if nothing else in that file calls it — check before removing the import, not just the button.
