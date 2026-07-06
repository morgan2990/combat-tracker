package main

import "combatapp/store"

// fakeItemUpserter is an in-memory itemUpserter for tests, keyed by
// (name, edition) the same way store.CatalogItemStore.UpsertItem is.
type fakeItemUpserter struct {
	byKey    map[string]store.CatalogItem
	forceErr error
	calls    []store.CatalogItem
}

func newFakeItemUpserter() *fakeItemUpserter {
	return &fakeItemUpserter{byKey: make(map[string]store.CatalogItem)}
}

func (f *fakeItemUpserter) UpsertItem(ci store.CatalogItem) (store.CatalogItem, store.UpsertOutcome, error) {
	if f.forceErr != nil {
		return store.CatalogItem{}, 0, f.forceErr
	}
	f.calls = append(f.calls, ci)

	key := ci.Name + "|" + ci.Edition
	outcome := store.UpsertUpdated
	if _, ok := f.byKey[key]; !ok {
		outcome = store.UpsertInserted
	}
	f.byKey[key] = ci
	return ci, outcome, nil
}
