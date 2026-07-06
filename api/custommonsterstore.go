package api

import "combatapp/store"

// customMonsterStore is the subset of store.CustomMonsterStore's methods
// called from this package. store.GlobalCustomMonsters is the production
// implementation; tests swap customMonsterDB for an in-memory fake.
type customMonsterStore interface {
	CreateCustomMonster(m store.CustomMonster) (store.CustomMonster, error)
	GetCustomMonsterByID(id string) (*store.CustomMonster, error)
	UpdateCustomMonster(id string, m store.CustomMonster) (store.CustomMonster, error)
	DeleteCustomMonster(id string) error
	ListCustomMonstersByOwner(ownerID string, edition string) ([]store.CustomMonster, error)
}

var customMonsterDB customMonsterStore = &store.GlobalCustomMonsters
