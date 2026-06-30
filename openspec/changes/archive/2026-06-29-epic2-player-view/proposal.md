## Why

With rooms and WebSocket connections established (Epic 1), players have no way to see the combat state, update their character, or add companions. This epic delivers the complete player-facing experience: a phone-optimized view showing the initiative tracker, tools to update their own HP and conditions, and the ability to manage summoned creatures.

## What Changes

- New character setup flow: after connecting, players who don't yet have an entity fill a setup form (Max HP + Initiative); reconnecting players with a matching entity skip setup and re-link automatically
- Server-side entity sorting: `State.Entities` is kept sorted descending by initiative after every mutation; `active_index` always refers to this sorted order
- WS message dispatcher: the server's read loop now parses and routes action messages (`setup_character`, `update_entity`, `add_companion`) instead of discarding them
- Backend authorization: `update_entity` is rejected unless the sender owns the entity (by session) or owns it as a companion (by `owner_id`)
- Full `PlayerView` implementation replacing the Epic 1 placeholder: initiative list with fog-of-war, hybrid HP editor (delta buttons + direct set), condition toggles, companion management
- Reconnection re-linking: on rejoin, `ValidateAndRegister` finds an existing entity by name and updates its `session_id` to the new session

## Capabilities

### New Capabilities
- `player-character-setup`: Player configures their entity (Max HP + Initiative) after connecting; reconnecting players re-link to their existing entity automatically
- `player-entity-update`: Player updates their own HP, Temporary HP, and conditions in real-time; server enforces ownership and broadcasts to all clients
- `companion-management`: Player adds a companion/summon (Name, Max HP, Initiative) linked to their entity; companion persists in the room after player disconnects

### Modified Capabilities
- `room-state`: Entity sorting is now a server responsibility — `State.Entities` is maintained in descending initiative order at all times; `active_index` is meaningful only within this sorted slice
- `room-connection`: On reconnection with an existing entity name, the server re-links `entity.session_id` to the new session rather than treating the name as a conflict

## Impact

- **`room/room.go`**: new `SetupCharacter()`, `UpdateEntity()`, `AddCompanion()` methods; `sortEntities()` private method; `ValidateAndRegister()` updated for reconnection re-linking
- **`ws/handler.go`**: read loop replaced with JSON message dispatcher
- **`frontend/src/App.tsx`**: post-connect logic to detect setup-needed vs. reconnection
- **`frontend/src/components/PlayerView.tsx`**: full implementation (initiative list, HP editor, condition toggles, companion button)
- **New React components**: `SetupForm`, `CompanionForm`
- **No new HTTP endpoints**: all interaction goes through the existing WebSocket connection
- **No new dependencies**: standard library JSON parsing on Go side; no new npm packages needed
