package api

import "combatapp/store"

// monsterStore is the subset of store.MonsterStore's methods called from
// this package. store.GlobalMonsters is the production implementation;
// tests swap monsterDB for an in-memory fake.
type monsterStore interface {
	GetMonsterByName(name string) (*store.Monster, error)
	SearchMonsters(query, edition, requesterID string) ([]store.MonsterHit, error)
}

var monsterDB monsterStore = &store.GlobalMonsters
