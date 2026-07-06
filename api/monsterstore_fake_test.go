package api

import "combatapp/store"

// fakeMonsterStore is an in-memory monsterStore for tests.
type fakeMonsterStore struct {
	byName     map[string]*store.Monster
	searchHits []store.MonsterHit
	forceErr   error
}

func newFakeMonsterStore(monsters ...store.Monster) *fakeMonsterStore {
	f := &fakeMonsterStore{byName: make(map[string]*store.Monster)}
	for i := range monsters {
		m := monsters[i]
		f.byName[m.Name] = &m
	}
	return f
}

func (f *fakeMonsterStore) GetMonsterByName(name string) (*store.Monster, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	m, ok := f.byName[name]
	if !ok {
		return nil, nil
	}
	cp := *m
	return &cp, nil
}

func (f *fakeMonsterStore) SearchMonsters(query, edition, requesterID string) ([]store.MonsterHit, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.searchHits, nil
}
