## MODIFIED Requirements

### Requirement: Backend exposes an edition-filtered monster search endpoint
The system SHALL expose `GET /api/search/monsters?q=<query>&edition=<edition>`. Both query parameters are required. The endpoint SHALL return HTTP 400 if either parameter is missing or if `edition` is not `"5e"` or `"5.5e"`. The endpoint SHALL query the Typesense `monsters` collection with typo tolerance and prefix matching on `name`, filtered to `edition:=<edition>`, and return the top matching hits as a JSON array of `{ id, name, max_hp, initiative_modifier }` objects. This response shape (a top-N array of lightweight hits) supersedes the prior exact-match contract of a single-or-empty array of full `Monster` documents.

#### Scenario: Typo-tolerant match
- **WHEN** a client sends `GET /api/search/monsters?q=Goblen&edition=5e`
- **THEN** the server SHALL return results including "Goblin" despite the misspelling

#### Scenario: Prefix match
- **WHEN** a client sends `GET /api/search/monsters?q=Beh&edition=5e`
- **THEN** the server SHALL return results including "Beholder"

#### Scenario: Results are filtered to the requested edition
- **WHEN** a document `{ name: "Goblin", edition: "5e" }` exists and a client sends `GET /api/search/monsters?q=Goblin&edition=5.5e`
- **THEN** the 5e document SHALL NOT appear in the results

#### Scenario: No matches
- **WHEN** a client sends `GET /api/search/monsters?q=Xyzzy&edition=5e` and nothing matches
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]`

#### Scenario: Missing q parameter
- **WHEN** a client sends `GET /api/search/monsters?edition=5e` with no `q`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Invalid edition parameter
- **WHEN** a client sends `GET /api/search/monsters?q=Goblin&edition=4e`
- **THEN** the server SHALL return HTTP 400

#### Scenario: Typesense unavailable at query time
- **WHEN** a client sends a valid `GET /api/search/monsters` request and Typesense cannot be reached
- **THEN** the server SHALL return HTTP 200 with an empty JSON array `[]` rather than an error

### Requirement: DM Combat Panel search bar uses the edition-filtered endpoint
The DM Combat Panel's monster search SHALL be a debounced, live autocomplete dropdown, separate from the staging Name/Max HP/Quantity fields used to finalize a creature for `add_creature`.

The search input SHALL NOT dispatch any request to `GET /api/search/monsters` while the input contains fewer than 3 characters. At 3 or more characters, the frontend SHALL debounce requests by approximately 175ms after the user stops typing before dispatching the request, using the room's current `edition`. If the input drops back below 3 characters, the frontend SHALL immediately clear and close the dropdown without dispatching a request, cancelling any pending debounced request.

Matching results SHALL render in a dropdown below the search input, each showing the monster's name, an edition badge, and its `max_hp`. Selecting a result (via click or `Enter`) SHALL: clear the search input and close the dropdown; autofill the staging Name and Max HP fields from the selected hit; and issue a single follow-up `GET /api/monsters/{name}` request to recover `source_type`, `reference_url`, and `pdf_object_key` for statblock-reference linking, since these fields are not present in the search index or its response.

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

#### Scenario: Selecting a result via click
- **WHEN** the DM clicks a dropdown result for "Beholder"
- **THEN** the frontend SHALL clear the search input, close the dropdown, autofill the staging Name field with "Beholder" and Max HP with the hit's `max_hp`, and issue `GET /api/monsters/Beholder` to recover statblock-reference fields

#### Scenario: Selecting a result via Enter
- **WHEN** the DM presses `Enter` while a dropdown result is highlighted
- **THEN** the same selection behavior SHALL occur as a click

#### Scenario: Free-text entry with no selection
- **WHEN** the DM types a creature name that matches no search result and submits the staging form without selecting anything
- **THEN** the creature SHALL be added using the manually-entered Name and Max HP, with no statblock reference attached

#### Scenario: Search respects room edition automatically
- **WHEN** the room was created with `edition: "5.5e"` and the DM searches for "Goblin"
- **THEN** the frontend SHALL send `edition=5.5e` in the request without any manual configuration by the DM
