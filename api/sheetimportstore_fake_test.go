package api

import (
	"testing"
	"time"

	"combatapp/store"
)

// fakeImportStore is an in-memory sheetImportStore for tests.
type fakeImportStore struct {
	byID     map[string]*store.CharacterSheetImport
	nextID   int
	forceErr error
}

func newFakeImportStore() *fakeImportStore {
	return &fakeImportStore{byID: map[string]*store.CharacterSheetImport{}}
}

func withFakeImportStore(t *testing.T, fake *fakeImportStore) {
	t.Helper()
	original := importDB
	importDB = fake
	t.Cleanup(func() { importDB = original })
}

func (f *fakeImportStore) Create(imp store.CharacterSheetImport) (*store.CharacterSheetImport, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	f.nextID++
	imp.ID = "imp-" + string(rune('0'+f.nextID))
	imp.Status = store.ImportDraft
	exp := time.Now().Add(store.ImportDraftTTL)
	imp.ExpiresAt = &exp
	f.byID[imp.ID] = &imp
	cp := imp
	return &cp, nil
}

func (f *fakeImportStore) Get(id string) (*store.CharacterSheetImport, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	imp, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *imp
	return &cp, nil
}

func (f *fakeImportStore) Claim(id string) (*store.CharacterSheetImport, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	imp, ok := f.byID[id]
	if !ok || imp.Status != store.ImportDraft {
		return nil, nil
	}
	imp.Status = store.ImportApplying
	cp := *imp
	return &cp, nil
}

func (f *fakeImportStore) Release(id string) error {
	if imp, ok := f.byID[id]; ok && imp.Status == store.ImportApplying {
		imp.Status = store.ImportDraft
	}
	return nil
}

func (f *fakeImportStore) MarkApplied(id, pcID string) error {
	if imp, ok := f.byID[id]; ok {
		imp.Status, imp.PCID, imp.ExpiresAt = store.ImportApplied, pcID, nil
	}
	return nil
}

func (f *fakeImportStore) Delete(id string) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeImportStore) DeleteOthersForPC(pcID, keepID string) error {
	for id, imp := range f.byID {
		if imp.PCID == pcID && id != keepID {
			delete(f.byID, id)
		}
	}
	return nil
}
