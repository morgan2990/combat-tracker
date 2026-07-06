package main

import "combatapp/store"

// itemUpserter is the subset of store.CatalogItemStore's methods called
// from this package. store.GlobalCatalogItems is the production
// implementation; tests swap itemDB for an in-memory fake.
type itemUpserter interface {
	UpsertItem(ci store.CatalogItem) (store.CatalogItem, store.UpsertOutcome, error)
}

var itemDB itemUpserter = &store.GlobalCatalogItems
