## ADDED Requirements

### Requirement: Dashboard's pooled-currency editor offers the same delta-entry modal as InventoryPanel

Dashboard's pooled-currency editor SHALL offer the same delta-entry modal specified for `CurrencyEditor` in `pc-inventory`.

#### Scenario: User applies a delta to pooled currency from Dashboard
- **WHEN** a user who owns a member PC opens the delta modal on the party's pooled currency in Dashboard's Parties section, enters one or more deltas, and clicks Apply
- **THEN** the pooled currency's editable state SHALL reflect the applied deltas, ready to be saved via the existing `PUT /api/parties/:id` save action
