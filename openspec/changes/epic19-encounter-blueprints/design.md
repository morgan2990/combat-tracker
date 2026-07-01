## Context

This change has three distinct concerns mapping directly to the epic's three user stories: persistence (`encounter-repository`), the authoring UI (`encounter-builder`), and the in-room injection pipeline (`encounter-injection`). Each has a near-exact existing precedent to mirror:

- `CustomMonster` (`store/custom_monster.go`) is the template for ownership-scoped CRUD: string `ID` via `newID()`, `OwnerID`/`OwnerDisplayName` set server-side (never trusted from the request body), full-document-replace semantics on update, `ListByOwner` for listing.
- `MonsterForm.tsx` is the template for the create/edit split UI (a single component keyed off whether a route param is present).
- `AddCreature` (`room/room.go`) already contains 100% of the "spawn N instances with per-instance rolls and aliasing" logic AC3 of US19.3 asks for — this change orchestrates it, it doesn't reimplement it.

Exploration surfaced that official monsters (`store.Monster`) and custom monsters (`store.CustomMonster`) live in genuinely different ID spaces: official monsters are looked up by `{name, edition}` via `GetMonsterByName` (no by-ID lookup exists, even though the Mongo doc has an `ObjectID`), while custom monsters are looked up by a string `ID`. The frontend's `MonsterSearchHit` already carries `is_custom` specifically to disambiguate this at search time — this change's `EncounterMonster` schema reuses that exact discriminator rather than inventing a new one.

## Goals / Non-Goals

**Goals:**
- A DM can build, save, edit, and delete encounter blueprints from the Dashboard, entirely outside of any live room.
- A DM can inject a saved encounter into a live room with one click; the room ends up in exactly the state it would be in if the DM had manually run `add_creature` once per monster group.
- Deleting a monster referenced by a saved encounter never corrupts the encounter or blocks future injections — the dangling group is simply skipped.

**Non-Goals:**
- No versioning/history of encounter edits — `UpdateEncounter` is a full-document replace, same as `UpdateCustomMonster`.
- No sharing encounters between DMs — strictly owner-scoped, same as custom monsters.
- No change to `AddCreature`'s signature or behavior — `InjectEncounter` calls the same underlying per-instance logic, it does not modify the existing method's contract.
- No new by-ID lookup for official monsters — resolved by name, reusing `GetMonsterByName` as-is.

## Decisions

**`EncounterMonster` shape carries a name/ID union, not a single `monster_id`, deviating from AC2's literal field:**
```go
type EncounterMonster struct {
    Name               string `bson:"name" json:"name"`                 // official: monster name; custom: display label only
    MonsterID          string `bson:"monster_id,omitempty" json:"monster_id,omitempty"` // custom monsters only
    IsCustom           bool   `bson:"is_custom" json:"is_custom"`
    Quantity           int    `bson:"quantity" json:"quantity"`
    DisplayName        string `bson:"display_name,omitempty" json:"display_name,omitempty"`
}
```
At injection time: `IsCustom == false` → resolve via `GetMonsterByName(Name)`; `IsCustom == true` → resolve via `GetCustomMonsterByID(MonsterID)`. This is the same branch the frontend's `selectMonster()` already performs at search time — `EncounterForm.tsx` populates both fields from the same `MonsterSearchHit` the DM picks, so no new frontend resolution logic is needed either.

**`Encounter` collection schema mirrors `CustomMonster` exactly:**
```go
type Encounter struct {
    ID       string             `bson:"id" json:"id"`
    Name     string             `bson:"name" json:"name"`
    OwnerID  string             `bson:"owner_id" json:"owner_id"`
    Edition  string             `bson:"edition" json:"edition"`
    Monsters []EncounterMonster `bson:"monsters" json:"monsters"`
}
```
No `OwnerDisplayName` needed — unlike custom monsters, encounters are never shown to anyone but their owner (no cross-DM discovery/search), so there's nothing to display it for.

**Routes: `/api/encounters` is a new top-level path, no collision risk.** The existing `custom-monsters` route comment already documents why `monsters/custom/{id}` was rejected in favor of a distinct top-level path (ServeMux prefix-overlap risk with `monsters/{name}`). `/api/encounters` has no such overlap with anything, so the full CRUD set registers directly: `POST /api/encounters`, `GET /api/encounters`, `GET /api/encounters/{id}`, `PUT /api/encounters/{id}`, `DELETE /api/encounters/{id}`.

**Edition filtering on `GET /api/encounters` is a query param, not client-side filtering.** `ListMyEncounters` accepts an optional `?edition=` query param and filters server-side, matching `SearchMonsters`' existing `edition` query param convention. The Dashboard's "My Encounters" list (no room context) omits the param and shows all editions; the DM Panel's "Encounter Templates" dropdown (inside a room, which has one fixed edition) always passes it.

**`InjectEncounter` takes pre-resolved data, not an encounter ID.** `ws/handler.go` does the Mongo lookups (fetch the encounter, resolve each `EncounterMonster` to a name/maxHP/initiativeModifier/sourceType/referenceURL/pdfObjectKey tuple, silently dropping any group that fails to resolve) *before* calling into `room`. `room.Room.InjectEncounter(sessionID string, groups []ResolvedMonsterGroup) error` then does exactly what `AddCreature`'s inner loop does, once per group, all under one `r.mu.Lock()` — followed by one `r.sortEntities()` and the caller performing one `rm.BroadcastState()`. This keeps `room` package free of any Mongo/HTTP concerns (it already only imports `store` for the persistence-snapshot types, never queries Mongo directly), consistent with the existing separation of concerns.

**Skipped-group failure is silent at the protocol level, not silent to the DM.** The `inject_encounter` response is still just a `RoomState` broadcast (no separate error message type exists in this app's WS protocol for partial failures), so there's no dedicated "N groups skipped" notification — the DM's evidence is simply seeing fewer creatures appear than the blueprint listed. This matches the app's existing minimal-signaling style (e.g. `add_creature`/`dm_update_entity` also never send acknowledgement or partial-failure messages back).

## Risks / Trade-offs

- **[Risk]** `EncounterMonster.Name` is stored even for custom-monster entries (as a label), but a DM could rename or delete that custom monster later, leaving a stale label that no longer matches the live monster's current name. → **Mitigation**: accepted; this is purely cosmetic in the builder's staging list before injection — the actual injected creature's name always comes from a fresh lookup at injection time, never from the stored label.
- **[Risk]** Building full update support (a decision made during exploration, beyond AC3's implied scope) adds a full edit screen and `PUT` endpoint that could have been deferred. → **Mitigation**: accepted; the marginal cost is small since it's a structural copy of `MonsterForm`'s existing edit-mode pattern, not new design.

## Open Questions

None outstanding — monster-reference resolution strategy, dangling-reference handling, and update-support scope were resolved during exploration prior to this proposal.
