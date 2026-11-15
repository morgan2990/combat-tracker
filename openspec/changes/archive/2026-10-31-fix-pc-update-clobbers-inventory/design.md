## Context

`pcPayload` decodes `items`, `currency`, and `shares_initiative` into plain values, so an omitted key is indistinguishable from an empty one. `UpdatePC` then writes all of them unconditionally. Three callers send different subsets:

| Caller | Sends |
|---|---|
| `CharacterForm`, saving a PC | `name`, `max_hp` |
| `CharacterForm`, saving a companion | `name`, `max_hp`, `shares_initiative` |
| `InventoryPanel` | `name`, `max_hp`, `items`, `currency` |

## Goals / Non-Goals

**Goals:**
- Fix the data loss on the server, for every caller.
- Keep explicit clearing of `items` working.

**Non-Goals:**
- Making `name` and `max_hp` optional.
- Restoring inventories that were already erased.

## Decisions

**Treat omitted fields as "keep the stored value".** `PCUpdateFields.Currency` and `SharesInitiative` become pointers; `Items` uses `nil` for omitted and a non-nil slice for present. JSON `[]` decodes to a non-nil empty slice, so clearing keeps working. An explicit `null` counts as omitted.

Alternative considered: have `CharacterForm` send the full document. Rejected because it leaves the destructive behavior in the API, and it could overwrite an inventory the DM changed after the form loaded.

**Build `$set` from the fields present,** rather than reading and rewriting the document. This avoids a read-modify-write race with `UpdatePCInventory`.

## Risks / Trade-offs

- [A client relying on omission to clear a field] → No such caller exists in the repo: the only callers are the two listed components. Clearing requires sending the empty value.
