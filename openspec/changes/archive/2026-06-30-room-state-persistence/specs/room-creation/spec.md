## MODIFIED Requirements

### Requirement: Room state is mirrored to MongoDB
The system SHALL store the authoritative, fast-path room state in the Go server's process memory, and additionally mirror that state to a MongoDB `rooms` collection so it survives a server restart.

#### Scenario: State is accessible after creation
- **WHEN** a room has been created
- **THEN** subsequent WebSocket connections referencing that `room_id` SHALL find the room and its state in the server's in-memory registry

#### Scenario: State survives a restart
- **WHEN** the server process restarts after a room has had at least one persisted snapshot
- **THEN** a subsequent lookup for that room's `room_id` SHALL restore its state (entities, round, active turn, edition) from MongoDB rather than reporting the room as not found
