## Context

The app has zero auth infrastructure today (no cookies, sessions, bcrypt, or JWT anywhere in the Go backend). DM identity is a bearer `dm_token` returned at room creation; player identity is a freeform name matched against a globally name-keyed `Profile` document. It runs in a Docker container behind Cloudflare — internet-reachable, not LAN-only — but the original epic1 design doc explicitly built this for a "trusted friend group, not adversarial users," and that low-friction, low-paranoia style should carry forward into this change rather than be replaced with enterprise-grade auth machinery.

Two existing facts shape the implementation:
- `golang.org/x/crypto` is already an indirect dependency (pulled in transitively) and includes `bcrypt` — no new third-party auth library is needed.
- The codebase already has an established "generate a short random token" helper (`newToken(...)`, used for entity IDs and room IDs in `room/room.go`) — new IDs (PCs, sessions, rooms unchanged) should reuse this rather than introducing a UUID library.

## Goals / Non-Goals

**Goals:**
- Self-serve username/passphrase accounts with bcrypt hashing.
- DB-backed, cookie-carried sessions with rolling ~90-day expiry.
- Replace `dm_token` with real room ownership (`owner_user_id`).
- Rename the persisted character concept from name-keyed `Profile` (`type: "player"`) to id-keyed `PC` (`type: "pc"`), owned by a `User`.
- A `RoomMembership` record powering a "recent rooms" list for players.
- A logged-in Dashboard replacing the anonymous `JoinScreen`.

**Non-Goals:**
- Email verification, password-reset-by-email, or any outbound email at all.
- OAuth/SSO.
- Self-service account recovery (explicitly accepted gap — operator fixes it by hand).
- Co-DM / shared room ownership (a room has exactly one `owner_user_id`; revisit later if needed).
- Brute-force/rate-limiting protection on login (matches the existing "not adversarial users" precedent; revisit if the trust model ever changes).
- Renaming OpenSpec capability folders or frontend component files to match the PC terminology (see Decision 6) — this change renames the *domain* concept, not every file/folder that happens to contain the word "player."
- Migrating existing Mongo data (explicitly a clean break — see Migration Plan).

## Decisions

### 1. Session storage and validation: DB-backed, validated once per connection/request, not re-checked per message

A `Session` document (`token`, `user_id`, `created_at`, `last_seen_at`, `expires_at`) lives in a new Mongo collection. For HTTP requests, an auth-check reads the cookie, looks up the session, and rejects if missing/expired. For WebSocket connections, the same check happens once at upgrade time and the resulting `user_id` is cached on the `Client` struct for the connection's lifetime — mirroring exactly how `Client.Role`/`Client.Name` are already trusted for a connection's lifetime today without being re-validated per message (`isDM(sessionID)` checks the cached `Role`, not the token, on every action). No in-memory session cache is introduced; at friend-group scale, a Mongo lookup per HTTP request and once per WS connect is cheap. Add one only if it becomes a measured bottleneck.

### 2. Rolling expiry is touched on connect, debounced thereafter — not on every message

Touch (`last_seen_at`/`expires_at` update) on: every authenticated REST request, and once when a WS connection is established. For long-lived WS connections (a multi-hour combat session), also touch on inbound WS messages, but **debounced** — only write if the last touch was more than 5 minutes ago. This keeps a marathon session alive without a Mongo write per HP update.

### 3. Cookie `Secure` flag must be configurable, not hardcoded true

This app is regularly run and tested over plain `http://` during local/LAN development (as just exercised manually for the previous change) — a hardcoded `Secure` attribute would silently break login in that environment, since browsers refuse to send `Secure` cookies over plain HTTP. Follow the existing pattern already used for `MONGODB_URI`/`TYPESENSE_URL` (env var with a sensible default): add a `COOKIE_SECURE` env var (or similar), defaulting to `true`, that the operator sets to `false` for local HTTP testing. `HttpOnly` and `SameSite=Lax` stay hardcoded (no reason to ever disable those).

### 4. `dm_token` removal is a direct swap, not a parallel system

`Room.DMToken` is removed; `Room.OwnerUserID` is added. `isDM(sessionID)` becomes `isOwner(sessionID)`, comparing the connection's cached `user_id` against `Room.OwnerUserID` — same call sites (`StartCombat`, `NextTurn`, etc.), same shape, different comparison. `POST /api/rooms` requires an authenticated session and sets `owner_user_id` from it instead of returning a `dm_token`.

### 5. PC identity: id-keyed, reusing the existing token-generation pattern

`store.Profile` is renamed `store.PC` and gains `OwnerUserID` and a generated `ID` (via the existing `newToken(...)` helper, matching entity/room ID generation already in the codebase). `Name` stops being a unique index; uniqueness (if any) is enforced only as "unique among this user's own PCs," not globally. `Companion.ParentPCName` becomes `Companion.ParentPCID`.

### 6. Minimal renaming surface — distinguish "player" the human role from "player" the entity type

The codebase uses "player" for two different things, and only one of them is being renamed:
- The WS connection `role` (`"dm" | "player"`) describes **the human's role at the table** — this is correct as-is and is NOT renamed. `JoinScreen`'s "Player" framing, `PlayerView.tsx`'s name, and the `/characters/...` route naming ("character" is already a fine human-facing synonym for PC) all stay.
- `Entity.type` / `Profile.type`'s value `"player"` describes **the character/profile object itself** — this is what renames to `"pc"`, since it's the thing that gets confused with the human once `User` exists.

Net effect: no frontend component or file renames are required by this change. The rename is: `Entity.Type` value string, `store.Profile` Go type name → `store.PC`, the `entities` Mongo collection's `type` field value, `parent_pc_name`/`ParentPCName` → `parent_pc_id`/`ParentPCID`, and corresponding `types.ts` / WS message shapes. `room.go`'s `(e.Type == "player" || ...)` checks become `(e.Type == "pc" || ...)`.

Similarly, OpenSpec capability folders (`player-profile-management`, `profile-based-join`) are **not** renamed as part of this change — delta specs are requirement-level, not folder-level, and a folder rename is unrelated churn. The folder name already tolerated this mismatch before (`parent_pc_name` lived inside `player-profile-management` pre-rename). Leave as a possible future housekeeping pass.

### 7. WS connection params shrink to `room_id`, `role`, and `pc_id`

`name` and `dm_token` are dropped from the WS query string entirely. Identity comes from the session cookie (sent automatically on the same-origin upgrade request); the server resolves the connecting user from it. For `role=player`, the client also passes `pc_id`; the server verifies that PC belongs to the connected user, then derives the entity's display name from the PC's stored `name`. For `role=dm`, the server checks `room.OwnerUserID` against the session's user — no client-supplied credential at all.

Close codes follow the existing scheme with one addition: 4004 (room not found) and 4009 (name/PC already active in room) keep their meaning; 4003 (forbidden — connecting as `dm` without being the owner, or as `player` with a `pc_id` you don't own) is reused rather than introduced fresh; a new code is needed for "no valid session" (e.g. 4001) since that's a new failure mode that didn't exist before (anonymous connections were always allowed).

### 8. Dashboard data loads via one combined endpoint

`GET /api/me` returns `{ user, rooms: [...], pcs: [...], recent_rooms: [...] }` in one call, rather than three separate REST resources. This matches the app's existing preference for simplicity over REST purity (see epic1's "full state broadcast over delta events" precedent) and minimizes round-trips for the one screen that needs all three.

## Risks / Trade-offs

- **No login rate-limiting/lockout** → A friend's passphrase could theoretically be brute-forced. Accepted, matching the existing non-adversarial threat model; revisit only if the app is ever opened beyond a closed friend group.
- **No self-service recovery** → A locked-out friend needs the operator to manually edit their `passphrase_hash` in Mongo. Explicitly accepted, not a v1 gap to fill later with urgency.
- **`Secure` cookie vs. local HTTP dev** → Mitigated by the `COOKIE_SECURE` env toggle (Decision 3); forgetting to set it locally will manifest as "login silently doesn't persist," worth a clear error/log line when a cookie write is attempted without HTTPS in non-Secure mode.
- **Debounced session-touch could let a session expire mid-marathon-session in an edge case** (e.g., a WS connection open for 90+ days straight with the debounce window never triggering a write because the connection itself never closes/reopens) → low risk in practice since rolling expiry is 90 days and any reasonable debounce window (5 min) fires constantly during an active session; not worth more complexity for a TTRPG combat tracker.
- **Breaking change, no anonymous access ever again** → Intentional and already called out in the proposal as BREAKING. Existing shared room codes/dm_tokens stop working the moment this ships.

## Migration Plan

No data migration (explicit decision — existing Mongo `entities`/`rooms` documents are dev/test data only, nothing live runs on them). Deploy steps:
1. Drop or rename the existing `entities` and `rooms` Mongo collections before deploying the new schema (avoids confusing half-shaped legacy documents sitting alongside the new owned/id-keyed ones under the same collection name).
2. Deploy the new binary/frontend together (this is a single-binary app with embedded frontend — no rolling/partial deploy concern).
3. Rollback, if needed, is simply redeploying the previous binary; since old collections were dropped rather than mutated in place, there's no forward-migration to reverse.

## Open Questions

- Exact bcrypt cost factor — recommend the `bcrypt` package default (currently 10) unless a specific reason emerges to raise it; not behavior-affecting enough to block implementation.
- Whether `GET /api/me`'s combined shape holds up as more dashboard data is added later, or should split into separate resources — fine to revisit post-implementation without spec impact.
