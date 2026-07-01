## 1. Auth/session infra (backend foundation)

- [x] 1.1 Add `User` Mongo collection/struct (`username` unique index, `passphrase_hash`, `created_at`, `display_name`) and `Session` collection/struct (`token`, `user_id`, `created_at`, `last_seen_at`, `expires_at`).
- [x] 1.2 Promote `golang.org/x/crypto` to a direct dependency; implement signup/login passphrase hashing and verification with `bcrypt`.
- [x] 1.3 Implement `POST /api/signup`, `POST /api/login`, `POST /api/logout` handlers per the `user-accounts` spec.
- [x] 1.4 Implement session-resolution middleware/helper for HTTP requests: reads the cookie, looks up the `Session`, rejects (401) if missing/expired, debounced-touches `last_seen_at`/`expires_at` (skip write if touched <5 min ago).
- [x] 1.5 Add the `COOKIE_SECURE` env-configurable flag (default true) controlling the cookie's `Secure` attribute; `HttpOnly` and `SameSite=Lax` always on.
- [x] 1.6 Implement `GET /api/me` returning `{ user, rooms, pcs, recent_rooms }` (rooms/pcs/recent_rooms populated as stubs until tasks in sections 2–3 wire up real data).

## 2. Ownership linking and player→PC rename (backend)

- [x] 2.1 Rename `store.Profile` → `store.PC`; add `ID` (generated via the existing `newToken(...)` pattern) and `OwnerUserID` fields; drop the global unique-by-`name` index, scope uniqueness (if any) to per-owner only.
- [x] 2.2 Rename `Companion.ParentPCName`/`parent_pc_name` → `ParentPCID`/`parent_pc_id` throughout Go and Mongo documents.
- [x] 2.3 Replace `POST/GET /api/entities...` with `POST /api/pcs`, `PUT /api/pcs/:id`, `GET /api/pcs/:id`, `POST /api/pcs/:parent_id/companions`, all requiring authentication and enforcing `owner_user_id` checks, per the `player-profile-management` spec.
- [x] 2.4 Rename `room.Entity.Type` value `"player"` → `"pc"` everywhere it's set or compared (`room.go` `StartCombat`, sorting/filtering logic, persistence snapshot/restore code).
- [x] 2.5 Add `Room.OwnerUserID`; remove `Room.DMToken`. Update `isDM(sessionID)` → `isOwner(sessionID)` comparing the connection's resolved `user_id` against `Room.OwnerUserID`.
- [x] 2.6 Update `POST /api/rooms` to require authentication, set `owner_user_id` from the session, and stop returning a `dm_token`, per the `room-creation` spec.
- [x] 2.7 Rewrite the `/ws` upgrade handler: resolve identity from the session cookie (reject with close code 4001 if absent/invalid); accept `room_id`, `role`, and (for `role=player`) `pc_id`; drop `name`/`dm_token` params; verify room ownership (DM) or PC ownership (player) before upgrading (close code 4003 on mismatch), per the `room-connection` spec.
- [x] 2.8 On player connect, resolve `max_hp`/`name` server-side from the owned PC document (not client-supplied query params); on `setup_character`, server-instantiate companion entities from all documents where `parent_pc_id` matches, replacing the client-driven `add_companion` auto-load, per `profile-based-join`.
- [x] 2.9 Update `refresh_from_profile` to resolve the PC via the connection's stored `pc_id` instead of a name lookup.
- [x] 2.10 Add the `RoomMembership` Mongo collection/struct; upsert it (by `user_id`+`room_id`) whenever a `role=player` connection completes `setup_character`, per `room-membership`.
- [x] 2.11 Wire `GET /api/me`'s `rooms` (owned), `pcs` (owned), and `recent_rooms` (from `RoomMembership`, ordered by `last_joined_at` descending) to real queries.
- [x] 2.12 Update `room-persistence` snapshot/restore code: persisted room document drops `dm_token`, gains `owner_user_id`; persisted entity `type` values use `"pc"`.
- [ ] 2.13 Deploy step: drop/rename the existing `entities` and `rooms` Mongo collections (clean break, no migration) before running the new schema against them.

## 3. Frontend

- [x] 3.1 Build a login/signup screen (replaces anonymous access) — username + passphrase, toggle between "log in" and "create account," calling `POST /api/login` / `POST /api/signup`.
- [x] 3.2 On app load, call `GET /api/me`; if unauthenticated, route to the login/signup screen; if authenticated, route to the Dashboard.
- [x] 3.3 Build the Dashboard at `/`: "As DM" section (My Rooms list with Open actions, edition selector + "+ New Room" calling `POST /api/rooms`); "As Player" section (My Characters list, "+ New Character", Recent Rooms list from `recent_rooms` with one-click rejoin using `last_pc_id`, plain room-code input for new invites).
- [x] 3.4 Remove `JoinScreen.tsx`'s DM/Player tab toggle and the "Rejoin Existing Room" form entirely. (`JoinScreen.tsx` deleted outright — fully superseded by `LoginScreen.tsx` + `Dashboard.tsx`.)
- [x] 3.5 Rewrite `CharacterForm.tsx` as an id-based create/edit form (`/characters/new`, `/characters/:id/edit`) against `/api/pcs`, scoped to the logged-in user's own PCs — remove the `?name=` lookup pre-fill.
- [x] 3.6 Update `App.tsx`'s `buildWsUrl`/connect flow: drop `name`/`dm_token` params, add `pc_id` for player connections; remove client-side `add_companion` auto-load calls (now server-side per task 2.8).
- [x] 3.7 Update `types.ts`: `Entity.type` union `'player' | 'creature' | 'companion'` → `'pc' | 'creature' | 'companion'`.
- [x] 3.8 Update `PlayerView.tsx`/`DMView.tsx` internal references from `entity.type === 'player'` to `'pc'` (no file/component renames needed — see design.md Decision 6).
- [x] 3.9 Add a persistent logout control (e.g. a small header/nav) calling `POST /api/logout` and returning to the login screen.

## 4. Verify

- [x] 4.1 Manually test: sign up a new account, confirm session cookie persists across a page reload (no re-login).
- [x] 4.2 Manually test: create a room as DM, confirm it appears in "My Rooms" after navigating away and back.
- [x] 4.3 Manually test: create a PC, join a room with it, confirm companions auto-load and the room appears in "Recent Rooms" with the correct `last_pc_id` on return visits.
- [x] 4.4 Manually test: a second account cannot connect as DM to the first account's room, and cannot join using a PC it doesn't own (expect close code 4003 in both cases).
- [x] 4.5 Manually test: log out, confirm the session is invalidated (cookie cleared, `GET /api/me` returns 401, and the old cookie value if replayed is rejected).
