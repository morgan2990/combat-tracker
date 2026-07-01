## Why

There is no real identity in this app today: a "DM" is just whoever holds a bearer `dm_token` copy-pasted at room creation (lose it, lose the room forever), and a "player" is just whoever types a character name that happens to match an existing globally-unique-by-name profile — no password, no ownership check, anyone who knows or guesses a name can claim or overwrite it. The app is internet-reachable (behind Cloudflare, not just LAN), and this model was an intentional MVP shortcut for a "trusted friend group" — but it can't support a friend owning multiple characters, recovering a room without the token, or the basic guarantee that your character is actually yours. Real `User` accounts fix this and let one person hold both a DM identity (linked to the rooms they run) and a player identity (linked to the characters they own) at the same time.

## What Changes

- New self-serve `User` accounts: username + passphrase (bcrypt-hashed), no email anywhere in the system.
- New DB-backed `Session` records behind an `HttpOnly`/`Secure`/`SameSite=Lax` cookie, rolling ~90-day expiry, refreshed on activity. Rides along automatically on the WebSocket upgrade (same-origin) — no separate token needs to be threaded through the connection URL.
- **BREAKING**: `dm_token` is retired entirely. Creating a room and joining a room as a player both now require being logged in — the app no longer supports any anonymous/account-less usage. `Room` gains `owner_user_id`; DM authority is "authenticated as the room's owner," not "holds the right string."
- **BREAKING**: the persisted character concept (`Profile`, Mongo `entities` collection, `type: "player"|"companion"`) is renamed to **PC** and becomes id-keyed with `owner_user_id`, instead of being keyed globally by `name`. `name` becomes a non-unique display label. Companions link via `parent_pc_id` instead of `parent_pc_name`. This also renames the `Entity.type` value `"player"` to `"pc"` everywhere it appears (Go structs, WS messages, frontend types) — the WS connection `role` (`dm`/`player`, describing the human's role at the table) is unaffected and keeps its existing name.
- **BREAKING**: existing unowned Mongo data (all dev/test, nothing live) is not migrated — clean break to the new owned/id-keyed model.
- New `RoomMembership` record (`user_id`, `room_id`, `last_pc_id`, `last_joined_at`), auto-upserted whenever a player joins a room, powering a "recent rooms" list. Grants no permission by itself — first-time room access is still via a shared room code.
- Frontend: the anonymous `JoinScreen` (DM tab / Player tab toggle, "Rejoin Room" form) is replaced by a login/signup screen (logged out) and a unified Dashboard (logged in) showing "My Rooms" (DM side) and "My Characters" + "Recent Rooms" + a join-by-code box (player side) side by side. `CharacterForm.tsx` becomes an id-based create/edit form scoped to the logged-in user's own PCs instead of a name-typed upsert.

## Capabilities

### New Capabilities
- `user-accounts`: self-serve signup, login, logout, passphrase hashing, and DB-backed session/cookie management.
- `room-membership`: tracks which rooms a player has joined and with which PC, powering the "recent rooms" dashboard list.

### Modified Capabilities
- `room-creation`: room creation requires an authenticated `User`; `dm_token` issuance is removed in favor of `owner_user_id`; the "Rejoin Existing Room" form is removed from the join flow.
- `room-connection`: WebSocket authentication moves from `dm_token`/freeform `name` query params to the session cookie plus room/PC ownership checks.
- `room-persistence`: the persisted room document drops `dm_token` and gains `owner_user_id`; persisted entity `type` values rename `"player"` to `"pc"`.
- `room-state`: the `Entity.type` enum value `"player"` renames to `"pc"`; role-based presentation rules are otherwise unchanged.
- `combat-turn-flow`: terminology only — "player and companion entity" becomes "pc and companion entity" in the start-combat gate; behavior is unchanged.
- `initiative-ui`: terminology only — entity-type references rename from "player" to "pc"; behavior is unchanged.
- `entity-schema`: terminology only — incidental "player" references in field descriptions rename to "pc".
- `player-profile-management`: redefined around ownership — profiles (now PCs) are id-keyed, owned by a `User`, created/edited only by their owner; the global name-uniqueness rule is removed.
- `profile-based-join`: redefined around login — joining a room as a player requires being logged in and selecting one of the user's own PCs (by id) instead of typing a name and looking it up; successful joins upsert a `RoomMembership`.

## Impact

- **Backend (Go)**: new `User`/`Session`/`RoomMembership` Mongo collections and handlers; `store.Profile` → `store.PC` (id-keyed); `room.Entity.Type` value rename; `room.Room` gains `OwnerUserID`, drops `DMToken`; `ws/handler.go` connection auth rewritten around session cookies; `room.go` `StartCombat`/ownership checks updated for the type rename.
- **Frontend (React)**: `JoinScreen.tsx` replaced by login/signup + Dashboard components; `CharacterForm.tsx` rewritten as an id-based owned-PC form; `App.tsx`'s `buildWsUrl`/connect flow simplified (drops `dm_token`/`name`, adds `pc_id` where relevant); `types.ts` `Entity.type` union updated; `PlayerView.tsx`/`DMView.tsx` updated for the type rename.
- **No data migration**: existing Mongo `entities`/`rooms` documents are not carried forward.
