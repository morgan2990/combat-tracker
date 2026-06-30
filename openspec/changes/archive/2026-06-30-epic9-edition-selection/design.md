## Context

Epic 8 introduced an edition-aware monster schema and scrubber. The `monsters` collection now holds distinct documents per `{name, edition}`. Until this change, nothing in the room or search layer uses that edition field — every monster lookup is ambiguous when two editions of the same creature exist.

`RoomState` currently has no edition field. `POST /api/rooms` takes no body. The DM join/create screen passes no configuration to the server. The existing `GET /api/monsters/{name}` is a direct lookup that returns whichever document MongoDB finds first.

## Goals / Non-Goals

**Goals:**
- Add `edition` to `RoomState` so all clients know the room's ruleset
- Wire room creation (backend + frontend) to accept and persist the DM's edition choice
- Establish `GET /api/search/monsters?q=&edition=` as the stable search contract for Epic 12
- Wire the DM panel search bar to the new endpoint

**Non-Goals:**
- Edition is not changeable after room creation — no mid-session edition switching
- No fuzzy/prefix search — exact name match only until Epic 12 replaces the query
- No Typesense integration — that is Epic 12's scope
- No filtering of the player view by edition — edition is a DM concern only

## Decisions

### Decision: Edition defaults to "5e" if omitted from POST /api/rooms
**Rationale:** Preserves backward compatibility with any tooling that calls the endpoint without a body. The vast majority of existing content is 5e. An explicit default is safer than rejecting the request.

### Decision: Edition is set once at creation and immutable
**Rationale:** Changing edition mid-session would invalidate all creature references already in the room (their `reference_url` and statblock URLs are edition-specific). Immutability avoids this class of bug entirely. DMs who want a different edition start a new room.

### Decision: New route GET /api/search/monsters rather than extending GET /api/monsters/{name}
**Rationale:** The existing endpoint is a direct lookup by path parameter. The search endpoint uses query parameters and returns an array — a different contract. Keeping them separate avoids conflating lookup with search and makes Epic 12's replacement surgery clean: only the handler body changes, not the route shape.

### Decision: Search returns an array even for exact-match
**Rationale:** Epic 12 will return a ranked list. If the search endpoint returns a single object today, the frontend would need to change its parsing when Epic 12 lands. Returning `[]` or `[monster]` now means zero frontend changes at Epic 12 time.

### Decision: Frontend reads edition from WebSocket room state, not a separate API call
**Rationale:** `RoomState` is already broadcast on every mutation and is the single source of truth for all room-derived UI. Adding `edition` to the broadcast is free — no new API call, no local storage, no prop drilling from a separate fetch.

## Risks / Trade-offs

- **Exact-match search is limited UX** → Mitigated by Epic 12 (typo-tolerant autocomplete). The interim is acceptable because the DM already knows monster names.
- **Edition field missing from old in-memory rooms** → Any room created before this deploy will have an empty `Edition` field. The broadcast will send `edition: ""` to clients. Frontend should treat empty edition as `"5e"` to avoid breaking existing sessions.
- **GET /api/monsters/{name} remains ambiguous** → Still returns whichever document MongoDB finds first for a given name. This is a known gap explicitly deferred until Epic 9 lands and the old endpoint can be updated or deprecated.

## Open Questions

- Should the DM be able to see the room's edition displayed somewhere in the UI after creation (e.g., a badge in the header)? Deferred to a future UX pass.
