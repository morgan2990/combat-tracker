package api

import "combatapp/store"

// catalogItemStore is the subset of store.CatalogItemStore's methods called
// from this package. store.GlobalCatalogItems is the production
// implementation; tests swap catalogItemDB for an in-memory fake.
type catalogItemStore interface {
	SearchItems(query, edition string) ([]store.CatalogItemHit, error)
	GetItemByID(id string) (*store.CatalogItem, error)
}

var catalogItemDB catalogItemStore = &store.GlobalCatalogItems
