package api

import "combatapp/store"

// sheetStore is the subset of store.CharacterSheetStore's methods called from
// this package. store.GlobalCharacterSheets is the production implementation;
// tests swap sheetDB for an in-memory fake.
type sheetStore interface {
	GetByPCID(pcID string) (*store.CharacterSheet, error)
	GetByPCIDs(pcIDs []string) (map[string]store.CharacterSheet, error)
	SaveSection(pcID, section string, revision int, value any) (int, error)
	DeleteByPCIDs(pcIDs []string) error
}

var sheetDB sheetStore = &store.GlobalCharacterSheets
