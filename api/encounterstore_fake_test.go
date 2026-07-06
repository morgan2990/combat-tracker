package api

import (
	"fmt"

	"combatapp/store"
)

// fakeEncounterStore is an in-memory encounterStore for tests.
type fakeEncounterStore struct {
	byID     map[string]*store.Encounter
	nextID   int
	forceErr error
}

func newFakeEncounterStore(encounters ...store.Encounter) *fakeEncounterStore {
	f := &fakeEncounterStore{byID: make(map[string]*store.Encounter)}
	for i := range encounters {
		e := encounters[i]
		f.byID[e.ID] = &e
	}
	return f
}

func (f *fakeEncounterStore) CreateEncounter(e store.Encounter) (store.Encounter, error) {
	if f.forceErr != nil {
		return store.Encounter{}, f.forceErr
	}
	if e.ID == "" {
		f.nextID++
		e.ID = fmt.Sprintf("fake-encounter-%d", f.nextID)
	}
	f.byID[e.ID] = &e
	return e, nil
}

func (f *fakeEncounterStore) GetEncounterByID(id string) (*store.Encounter, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	e, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *e
	return &cp, nil
}

func (f *fakeEncounterStore) UpdateEncounter(id string, e store.Encounter) (store.Encounter, error) {
	if f.forceErr != nil {
		return store.Encounter{}, f.forceErr
	}
	e.ID = id
	f.byID[id] = &e
	return e, nil
}

func (f *fakeEncounterStore) DeleteEncounter(id string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeEncounterStore) ListEncountersByOwner(ownerID string, edition string) ([]store.Encounter, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	var out []store.Encounter
	for _, e := range f.byID {
		if e.OwnerID != ownerID {
			continue
		}
		if edition != "" && e.Edition != edition {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}
