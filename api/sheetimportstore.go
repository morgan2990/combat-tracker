package api

import "combatapp/store"

// sheetImportStore is the subset of store.CharacterSheetImportStore's methods
// called from this package; tests swap importDB for an in-memory fake.
type sheetImportStore interface {
	Create(imp store.CharacterSheetImport) (*store.CharacterSheetImport, error)
	Get(id string) (*store.CharacterSheetImport, error)
	Claim(id string) (*store.CharacterSheetImport, error)
	Release(id string) error
	MarkApplied(id, pcID string) error
	Delete(id string) error
	DeleteOthersForPC(pcID, keepID string) error
}

var importDB sheetImportStore = &store.GlobalSheetImports
