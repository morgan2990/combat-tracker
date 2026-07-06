package api

import "combatapp/store"

// pcStore is the subset of store.Store's methods called from this package.
// store.Global (a *store.Store) is the production implementation; tests
// swap pcDB for an in-memory fake so handler logic — validation, ownership
// checks, response shaping, error paths — is testable without a live
// MongoDB connection.
type pcStore interface {
	CreatePC(ownerUserID, name string, maxHP int) (*store.PC, error)
	GetPCByID(id string) (*store.PC, error)
	UpdatePC(id string, fields store.PCUpdateFields) error
	DeletePC(id string) error
	UpdatePCInventory(id string, items []store.Item, currency store.Currency) (*store.PC, error)
	GetCompanionsByParentID(parentID string) ([]store.PC, error)
	CreateCompanion(parentPCID, ownerUserID, name string, maxHP int, sharesInitiative bool) (*store.PC, error)
	OwnsAnyPC(ownerUserID string, pcIDs []string) (bool, error)
	ListPCsByOwner(ownerUserID string) ([]store.PC, error)
}

var pcDB pcStore = &store.Global
