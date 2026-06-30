# Epic 4: Combat Resolution and State Reset

## US4.1: End Combat and Room Cleanup
**As a** Dungeon Master,  
**I want to** have an "End Combat" button in my panel,  
**So that** I can instantly finish the encounter and wipe out all temporary enemies while keeping the party intact.

### Acceptance Criteria:
- **AC 1:** The DM interface must feature a prominent "End Combat" button, visible only when a combat session is active or after initiative has been locked.
- **AC 2:** When clicked, the system must prompt the DM with a confirmation dialog to prevent accidental clicks during live gameplay.
- **AC 3:** Upon confirmation, the Go backend must process the room's volatile memory array and:
    - **Remove** all ephemeral entities created by the DM (monsters, hostile NPCs, environmental hazards).
    - **Keep** all Player Characters (PCs).
    - **Keep** all Player Companions / Summons linked to an active `player_id` (as defined in US2.3).
- **AC 4:** The "Round Counter" must be reset to 0, and the active turn indicator must be cleared.
- **AC 5:** The updated and cleaned room state must be broadcasted via WebSockets to all connected clients instantly, transitioning their view back to a "Waiting / Out of Combat" state.