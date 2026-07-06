package api

import "combatapp/store"

// fakeCatalogItemStore is an in-memory catalogItemStore for tests.
type fakeCatalogItemStore struct {
	byID       map[string]*store.CatalogItem
	searchHits []store.CatalogItemHit
	forceErr   error
}

func newFakeCatalogItemStore(items ...store.CatalogItem) *fakeCatalogItemStore {
	f := &fakeCatalogItemStore{byID: make(map[string]*store.CatalogItem)}
	for i := range items {
		it := items[i]
		f.byID[it.ID.Hex()] = &it
	}
	return f
}

func (f *fakeCatalogItemStore) SearchItems(query, edition string) ([]store.CatalogItemHit, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.searchHits, nil
}

func (f *fakeCatalogItemStore) GetItemByID(id string) (*store.CatalogItem, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	ci, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *ci
	return &cp, nil
}
