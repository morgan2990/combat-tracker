## MODIFIED Requirements

### Requirement: DM can list their own custom monster templates
The system SHALL expose `GET /api/custom-monsters` (optionally with an `edition` query parameter) that returns `custom_monsters` documents whose `owner_id` matches the authenticated requester, regardless of their `private` value. The request SHALL require authentication. When `edition` is provided, the response SHALL be filtered to only documents matching that edition; when omitted, all of the requester's documents are returned regardless of edition.

#### Scenario: DM lists their own monsters
- **WHEN** an authenticated DM who has created three custom monsters (two public, one private) requests their own list with no `edition` parameter
- **THEN** the server SHALL return all three documents

#### Scenario: DM's list excludes other owners' monsters
- **WHEN** an authenticated DM requests their own list and another DM has also created custom monsters
- **THEN** the response SHALL NOT include the other DM's documents, public or private

#### Scenario: DM lists their own monsters filtered by edition
- **WHEN** an authenticated DM who owns monsters in both `"5e"` and `"5.5e"` requests `GET /api/custom-monsters?edition=5e`
- **THEN** the server SHALL return only that DM's `"5e"` documents

#### Scenario: Edition filter omitted returns every edition
- **WHEN** an authenticated DM requests `GET /api/custom-monsters` with no `edition` parameter
- **THEN** the server SHALL return that DM's documents across all editions, exactly as it did before this filter existed
