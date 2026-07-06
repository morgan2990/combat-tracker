package main

import "combatapp/store"

// fakeMonsterUpserter is an in-memory monsterUpserter for tests. It mimics
// store.MonsterStore.UpsertMonster's real semantics closely enough for the
// scrub pipeline's _copy resolution to behave the same way against it as
// against Mongo: keyed by (name, edition), reports insert vs. update, and
// returns the stored document (including any fields Mongo would echo back
// unchanged, like MaxHP/InitiativeModifier) for downstream inheritance.
type fakeMonsterUpserter struct {
	byKey    map[string]store.Monster
	forceErr error
	calls    []store.Monster
}

func newFakeMonsterUpserter() *fakeMonsterUpserter {
	return &fakeMonsterUpserter{byKey: make(map[string]store.Monster)}
}

func (f *fakeMonsterUpserter) UpsertMonster(m store.Monster) (store.Monster, store.UpsertOutcome, error) {
	if f.forceErr != nil {
		return store.Monster{}, 0, f.forceErr
	}
	f.calls = append(f.calls, m)

	key := m.Name + "|" + m.Edition
	outcome := store.UpsertUpdated
	if _, ok := f.byKey[key]; !ok {
		outcome = store.UpsertInserted
	}
	f.byKey[key] = m
	return m, outcome, nil
}
