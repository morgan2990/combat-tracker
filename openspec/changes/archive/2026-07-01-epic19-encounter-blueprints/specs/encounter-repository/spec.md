## ADDED Requirements

### Requirement: Encounter Data Model

The system SHALL represent a saved encounter blueprint using the following structure, stored in a new MongoDB `encounters` collection:

```go
type Encounter struct {
    ID       string             `bson:"id" json:"id"`
    Name     string             `bson:"name" json:"name"`
    OwnerID  string              `bson:"owner_id" json:"owner_id"`
    Edition  string              `bson:"edition" json:"edition"`
    Monsters []EncounterMonster `bson:"monsters" json:"monsters"`
}

type EncounterMonster struct {
    Name        string `bson:"name" json:"name"`
    MonsterID   string `bson:"monster_id,omitempty" json:"monster_id,omitempty"`
    IsCustom    bool   `bson:"is_custom" json:"is_custom"`
    Quantity    int    `bson:"quantity" json:"quantity"`
    DisplayName string `bson:"display_name,omitempty" json:"display_name,omitempty"`
}
```

- `ID` is a server-generated random string (matching `CustomMonster.ID`'s generation via `store.NewID()`), never client-supplied on create.
- `Edition` SHALL be one of `"5e"` or `"5.5e"`.
- For `EncounterMonster` entries where `IsCustom` is `false`, `Name` identifies an official monster (resolved by name at injection time); `MonsterID` is empty.
- For entries where `IsCustom` is `true`, `MonsterID` identifies a custom monster document (resolved by id at injection time); `Name` is a display label only, not used for resolution.
- `Quantity` SHALL be at least 1.

#### Scenario: Encounter stores a mix of official and custom monster groups
- **WHEN** an encounter is saved with one group `{ name: "Goblin", is_custom: false, quantity: 3 }` and another `{ name: "My Homebrew Ooze", monster_id: "abc123", is_custom: true, quantity: 1, display_name: "The Blob" }`
- **THEN** both groups persist on the encounter document exactly as submitted

### Requirement: DM can create an encounter

The system SHALL expose `POST /api/encounters` accepting a JSON body with `name`, `edition`, and `monsters` (array of `EncounterMonster`). The request SHALL require authentication. The server SHALL set `owner_id` from the authenticated session (never trusted from the request body) and insert a new document with a fresh `id`.

#### Scenario: DM creates an encounter
- **WHEN** an authenticated DM sends `POST /api/encounters` with a valid `name`, `edition`, and non-empty `monsters` array
- **THEN** the server SHALL insert a document into `encounters` with `owner_id` set from the session, and return HTTP 200 with the document's `id`

#### Scenario: Unauthenticated request is rejected
- **WHEN** a request to `POST /api/encounters` is made with no valid session
- **THEN** the server SHALL return HTTP 401

#### Scenario: Missing or invalid edition is rejected
- **WHEN** an authenticated DM sends `POST /api/encounters` with `edition` absent or not one of `"5e"`/`"5.5e"`
- **THEN** the server SHALL return HTTP 400

### Requirement: DM can list their own encounters

The system SHALL expose `GET /api/encounters`, optionally filtered by an `edition` query parameter, returning only encounters owned by the authenticated requester.

#### Scenario: DM lists all their encounters
- **WHEN** an authenticated DM sends `GET /api/encounters` with no `edition` param
- **THEN** the server SHALL return every encounter document where `owner_id` matches the requester, regardless of edition

#### Scenario: DM lists encounters filtered by edition
- **WHEN** an authenticated DM sends `GET /api/encounters?edition=5e`
- **THEN** the server SHALL return only that DM's encounters where `edition == "5e"`

#### Scenario: DM with no encounters gets an empty list
- **WHEN** an authenticated DM with zero saved encounters sends `GET /api/encounters`
- **THEN** the server SHALL return HTTP 200 with an empty JSON array, not `null`

### Requirement: DM can fetch a single encounter by ID

The system SHALL expose `GET /api/encounters/{id}`, returning the encounter only if it is owned by the authenticated requester.

#### Scenario: Owner fetches their encounter
- **WHEN** the encounter's owner sends `GET /api/encounters/{id}`
- **THEN** the server SHALL return HTTP 200 with the full encounter document

#### Scenario: Non-owner cannot fetch another DM's encounter
- **WHEN** an authenticated DM who does not own the encounter sends `GET /api/encounters/{id}`
- **THEN** the server SHALL return HTTP 403

#### Scenario: Fetching a nonexistent encounter
- **WHEN** a request targets an `id` with no matching document
- **THEN** the server SHALL return HTTP 404

### Requirement: DM can update an encounter

The system SHALL expose `PUT /api/encounters/{id}`, replacing the encounter's `name`, `edition`, and `monsters` in full, after verifying the authenticated requester owns the existing document. `owner_id` SHALL be preserved from the existing document, never overwritten from the request body.

#### Scenario: Owner updates their encounter
- **WHEN** the encounter's owner sends `PUT /api/encounters/{id}` with an updated `name` and `monsters` array
- **THEN** the server SHALL replace the document's fields (except `id` and `owner_id`) and return the updated document

#### Scenario: Non-owner cannot update another DM's encounter
- **WHEN** an authenticated DM who does not own the encounter sends `PUT /api/encounters/{id}`
- **THEN** the server SHALL return HTTP 403 and the document SHALL remain unchanged

### Requirement: DM can delete an encounter

The system SHALL expose `DELETE /api/encounters/{id}`, after verifying the authenticated requester owns the document.

#### Scenario: Owner deletes their encounter
- **WHEN** the encounter's owner sends `DELETE /api/encounters/{id}`
- **THEN** the server SHALL remove the document and return HTTP 204

#### Scenario: Non-owner cannot delete another DM's encounter
- **WHEN** an authenticated DM who does not own the encounter sends `DELETE /api/encounters/{id}`
- **THEN** the server SHALL return HTTP 403 and the document SHALL remain in the collection

#### Scenario: Deleting an encounter does not affect monsters it references
- **WHEN** an encounter referencing a custom monster is deleted
- **THEN** the referenced `CustomMonster` document SHALL remain unaffected — encounters do not own or cascade-affect the monsters they reference
