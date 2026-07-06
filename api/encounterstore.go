package api

import "combatapp/store"

// encounterStore is the subset of store.EncounterStore's methods called
// from this package. store.GlobalEncounters is the production
// implementation; tests swap encounterDB for an in-memory fake.
type encounterStore interface {
	CreateEncounter(e store.Encounter) (store.Encounter, error)
	GetEncounterByID(id string) (*store.Encounter, error)
	UpdateEncounter(id string, e store.Encounter) (store.Encounter, error)
	DeleteEncounter(id string) error
	ListEncountersByOwner(ownerID string, edition string) ([]store.Encounter, error)
}

var encounterDB encounterStore = &store.GlobalEncounters
