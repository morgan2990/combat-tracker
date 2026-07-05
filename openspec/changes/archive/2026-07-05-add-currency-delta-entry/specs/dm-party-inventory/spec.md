## ADDED Requirements

### Requirement: PartyInventoryPanel's currency editor offers the same delta-entry modal as InventoryPanel

Each party member's currency editor in `PartyInventoryPanel` SHALL offer the same delta-entry modal specified for `CurrencyEditor` in `pc-inventory`, scoped independently to each member's own currency — applying a delta while editing one member SHALL NOT affect any other member's currency.

#### Scenario: Applying a delta for one member does not affect another
- **WHEN** the DM applies a currency delta while editing one party member's currency in the aggregated view
- **THEN** only that member's currency SHALL be affected; other members' currency SHALL remain unchanged
