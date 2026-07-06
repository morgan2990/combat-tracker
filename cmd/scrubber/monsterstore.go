package main

import "combatapp/store"

// monsterUpserter is the subset of store.MonsterStore's methods called from
// this package. store.GlobalMonsters is the production implementation;
// tests swap monsterDB for an in-memory fake.
type monsterUpserter interface {
	UpsertMonster(m store.Monster) (store.Monster, store.UpsertOutcome, error)
}

var monsterDB monsterUpserter = &store.GlobalMonsters
