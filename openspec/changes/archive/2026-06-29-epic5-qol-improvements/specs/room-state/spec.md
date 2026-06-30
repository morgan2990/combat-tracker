## ADDED Requirements

### Requirement: Entities at zero HP are displayed as Unconscious
An entity with `current_hp === 0` and `dead === false` is in the Unconscious state. All clients SHALL render such entities with a visual treatment distinct from both alive entities and Dead entities. No new server-side field is required; the state is derived from existing `current_hp` and `dead` fields.

#### Scenario: Player entity reaches zero HP without being killed
- **WHEN** a client renders an entity with `current_hp === 0` and `dead === false`
- **THEN** the client SHALL display an Unconscious badge (e.g., 😵) and an amber-tinted background; the entity SHALL NOT be greyed out (which is reserved for Dead entities)

#### Scenario: Revived entity is displayed as Unconscious
- **WHEN** the DM revives a dead entity by setting `dead: false` and `current_hp` is still 0
- **THEN** all clients SHALL transition the entity's display from Dead (greyed-out) to Unconscious (amber tint) immediately upon receiving the broadcast

#### Scenario: Dead state takes visual precedence over zero HP
- **WHEN** an entity has both `current_hp === 0` and `dead === true`
- **THEN** all clients SHALL render the entity as Dead (greyed-out, 💀 badge) not as Unconscious; Dead is the definitive confirmed state

#### Scenario: Entity recovers from Unconscious state
- **WHEN** an entity's `current_hp` is updated to a value greater than 0 while `dead === false`
- **THEN** all clients SHALL remove the Unconscious indicator and render the entity in its normal alive state
