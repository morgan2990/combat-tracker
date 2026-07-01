# Spec: Monster Search

## Purpose

Defines the edition-filtered monster search endpoint used by the DM Combat Panel, and the debounced autocomplete UX built on top of it. The endpoint queries the Typesense search index (see `monster-search-index`) for typo-tolerant, prefix-matched, edition-filtered results.

## Requirements

### Requirement: Backend exposes an edition-filtered monster search endpoint
The system SHALL expose `GET /api/search/monsters?q=<query>&edition=<edition>`. The request SHALL require authentication. Both query parameters are required. The endpoint SHALL return HTTP 400 if either parameter is missing or if `edition` is not `"5e"` or `"5.5e"`, and HTTP 401 if the requester is not authenticated. The endpoint SHALL query the Typesense `monsters` collection with typo tolerance and prefix matching on `name`, filtered to `edition:=<edition>` AND a visibility filter that only matches: official monsters (`is_custom:=false`), public custom monsters (`is_custom:=true && private:=false`), or the requester's own private custom monsters (`is_custom:=true && private:=true && owner_id:=<requester id>`). Matching hits SHALL be returned as a JSON array of `{ id, name, max_hp, initiative_modifier, is_custom, owner_display_name }` objects, with `owner_display_name` present only for custom hits.

#### Scenario: Typo-tolerant match
- **WHEN** an authenticated DM sends `GET /api/search/monsters?q=Goblen&edition=5e`
- **THEN** the server SHALL return results including "Goblin" despite the misspelling

#### Scenario: Prefix match
- **WHEN** an authenticated DM sends `GET /api/search/monsters?q=Beh&edition=5e`
- **THEN** the server SHALL return results including "Beholder"

#### Scenario: Results are filtered to the requested edition
- **WHEN** a document `{ name: "Goblin", edition: "5e" }` exists and an authenticated DM sends `GET /api/search/monsters?q=Goblin&edition=5.5e`
- **THEN** the 5e document SHALL NOT appear in the results

#### Scenario: Official and public custom hits both appear
- **WHEN** an official "Goblin" and a public custom "Goblin" (different owner) both exist for the requested edition
- **THEN** the DM's search results SHALL include both, distinguishable via `is_custom` and `owner_display_name`

#### Scenario: Other DMs' private monsters are excluded
- **WHEN** a different DM has a private custom "Goblin" for the requested edition
- **THEN** the requesting DM's search results SHALL NOT include that document

#### Scenario: A DM's own private monster is included in their own search
- **WHEN** the requesting DM has their own private custom "Goblin" for the requested edition
- **THEN** their search results SHALL include it, with `owner_display_name` set to their own display name

#### Scenario: Unauthenticated search is rejected
- **WHEN** a request to `GET /api/search/monsters` is made with no valid session
- **THEN** the server SHALL return HTTP 401

#### Scenario: No matches
- **WHEN** an authenticated DM sends `GET /api/search/monsters?q=Xyzzy&edition=5e` and nothing matches
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]`

#### Scenario: Missing q parameter
- **WHEN** an authenticated DM sends `GET /api/search/monsters?edition=5e` with no `q`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Invalid edition parameter
- **WHEN** an authenticated DM sends `GET /api/search/monsters?q=Goblin&edition=4e`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Typesense unavailable at query time
- **WHEN** an authenticated DM sends a valid `GET /api/search/monsters` request and Typesense cannot be reached
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]` rather than an error

---

### Requirement: DM Combat Panel search bar uses the edition-filtered endpoint
The DM Combat Panel's monster search SHALL be a debounced, live autocomplete dropdown, separate from the staging Name/Max HP/Quantity fields used to finalize a creature for `add_creature`.

The search input SHALL NOT dispatch any request to `GET /api/search/monsters` while the input contains fewer than 3 characters. At 3 or more characters, the frontend SHALL debounce requests by approximately 175ms after the user stops typing before dispatching the request, using the room's current `edition`. If the input drops back below 3 characters, the frontend SHALL immediately clear and close the dropdown without dispatching a request, cancelling any pending debounced request.

Matching results SHALL render in a dropdown below the search input, each showing the monster's name, an edition badge, its `max_hp`, and — for custom hits — an indicator of its author (e.g. "by Alice") so the DM can distinguish same-named monsters. Selecting a result (via click or `Enter`) SHALL: clear the search input and close the dropdown; autofill the staging Name and Max HP fields from the selected hit; and issue a single follow-up request to recover `source_type`, `reference_url`, and `pdf_object_key` for statblock-reference linking — `GET /api/monsters/{name}` for an official hit (`is_custom: false`), or `GET /api/custom-monsters/{id}` for a custom hit (`is_custom: true`), since these fields are not present in the search index or its response.

The staging Name/Max HP fields SHALL remain freely editable at all times, whether populated by a dropdown selection or typed directly, so creatures with no matching search result can still be added.

#### Scenario: No request below 3 characters
- **WHEN** the DM has typed 2 characters into the search input
- **THEN** the frontend SHALL NOT dispatch any request to `GET /api/search/monsters`

#### Scenario: Debounced request at 3+ characters
- **WHEN** the DM types a 3rd character and then pauses
- **THEN** the frontend SHALL wait approximately 175ms after the pause before dispatching `GET /api/search/monsters?q=<input>&edition=<room edition>`

#### Scenario: Dropping below 3 characters clears instantly
- **WHEN** the DM deletes characters such that the input drops from 3 to 2 characters
- **THEN** the frontend SHALL immediately clear and close the dropdown and SHALL NOT dispatch a request, even if a debounced request was pending

#### Scenario: Selecting an official result via click
- **WHEN** the DM clicks a dropdown result for official "Beholder"
- **THEN** the frontend SHALL clear the search input, close the dropdown, autofill the staging Name field with "Beholder" and Max HP with the hit's `max_hp`, and issue `GET /api/monsters/Beholder` to recover statblock-reference fields

#### Scenario: Selecting a custom result via click
- **WHEN** the DM clicks a dropdown result for a custom "Goblin" hit with `id: "abc123"`
- **THEN** the frontend SHALL autofill the staging fields as above and issue `GET /api/custom-monsters/abc123` to recover statblock-reference fields

#### Scenario: Selecting a result via Enter
- **WHEN** the DM presses `Enter` while a dropdown result is highlighted
- **THEN** the same selection behavior SHALL occur as a click, routed by `is_custom` as above

#### Scenario: Free-text entry with no selection
- **WHEN** the DM types a creature name that matches no search result and submits the staging form without selecting anything
- **THEN** the creature SHALL be added using the manually-entered Name and Max HP, with no statblock reference attached

#### Scenario: Search respects room edition automatically
- **WHEN** the room was created with `edition: "5.5e"` and the DM searches for "Goblin"
- **THEN** the frontend SHALL send `edition=5.5e` in the request without any manual configuration by the DM
