package api

import (
	"fmt"

	"combatapp/store"
)

// fakeCustomMonsterStore is an in-memory customMonsterStore for tests.
type fakeCustomMonsterStore struct {
	byID     map[string]*store.CustomMonster
	nextID   int
	forceErr error
}

func newFakeCustomMonsterStore(monsters ...store.CustomMonster) *fakeCustomMonsterStore {
	f := &fakeCustomMonsterStore{byID: make(map[string]*store.CustomMonster)}
	for i := range monsters {
		m := monsters[i]
		f.byID[m.ID] = &m
	}
	return f
}

func (f *fakeCustomMonsterStore) CreateCustomMonster(m store.CustomMonster) (store.CustomMonster, error) {
	if f.forceErr != nil {
		return store.CustomMonster{}, f.forceErr
	}
	if m.ID == "" {
		f.nextID++
		m.ID = fmt.Sprintf("fake-monster-%d", f.nextID)
	}
	m.IsCustom = true
	f.byID[m.ID] = &m
	return m, nil
}

func (f *fakeCustomMonsterStore) GetCustomMonsterByID(id string) (*store.CustomMonster, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	m, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *m
	return &cp, nil
}

func (f *fakeCustomMonsterStore) UpdateCustomMonster(id string, m store.CustomMonster) (store.CustomMonster, error) {
	if f.forceErr != nil {
		return store.CustomMonster{}, f.forceErr
	}
	m.ID = id
	m.IsCustom = true
	f.byID[id] = &m
	return m, nil
}

func (f *fakeCustomMonsterStore) DeleteCustomMonster(id string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeCustomMonsterStore) ListCustomMonstersByOwner(ownerID string, edition string) ([]store.CustomMonster, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	var out []store.CustomMonster
	for _, m := range f.byID {
		if m.OwnerID != ownerID {
			continue
		}
		if edition != "" && m.Edition != edition {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}
