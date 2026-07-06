package api

import (
	"fmt"

	"combatapp/store"
)

// fakePCStore is an in-memory pcStore for tests. It mirrors the observable
// behavior of store.Store's real MongoDB-backed methods (nil-not-found
// instead of an error, DeletePC cascading to companions) closely enough for
// handler-level tests, without needing a live MongoDB connection.
type fakePCStore struct {
	byID   map[string]*store.PC
	nextID int
	// forceErr, if set, is returned by every method instead of doing anything —
	// lets tests exercise handlers' "database error" branches.
	forceErr error
}

func newFakePCStore(pcs ...store.PC) *fakePCStore {
	f := &fakePCStore{byID: make(map[string]*store.PC)}
	for i := range pcs {
		pc := pcs[i]
		f.byID[pc.ID] = &pc
	}
	return f
}

func (f *fakePCStore) CreatePC(ownerUserID, name string, maxHP int) (*store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	f.nextID++
	pc := &store.PC{
		ID:          fmt.Sprintf("fake-pc-%d", f.nextID),
		OwnerUserID: ownerUserID,
		Name:        name,
		Type:        "pc",
		MaxHP:       maxHP,
		Items:       []store.Item{},
	}
	f.byID[pc.ID] = pc
	return pc, nil
}

func (f *fakePCStore) GetPCByID(id string) (*store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	pc, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *pc
	return &cp, nil
}

func (f *fakePCStore) UpdatePC(id string, fields store.PCUpdateFields) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	pc, ok := f.byID[id]
	if !ok {
		return nil
	}
	pc.Name = fields.Name
	pc.MaxHP = fields.MaxHP
	pc.SharesInitiative = fields.SharesInitiative
	items := fields.Items
	if items == nil {
		items = []store.Item{}
	}
	pc.Items = items
	pc.Currency = fields.Currency
	return nil
}

func (f *fakePCStore) DeletePC(id string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	delete(f.byID, id)
	for pcID, pc := range f.byID {
		if pc.Type == "companion" && pc.ParentPCID == id {
			delete(f.byID, pcID)
		}
	}
	return nil
}

func (f *fakePCStore) UpdatePCInventory(id string, items []store.Item, currency store.Currency) (*store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	pc, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	if items == nil {
		items = []store.Item{}
	}
	pc.Items = items
	pc.Currency = currency
	cp := *pc
	return &cp, nil
}

func (f *fakePCStore) GetCompanionsByParentID(parentID string) ([]store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	var out []store.PC
	for _, pc := range f.byID {
		if pc.Type == "companion" && pc.ParentPCID == parentID {
			out = append(out, *pc)
		}
	}
	return out, nil
}

func (f *fakePCStore) CreateCompanion(parentPCID, ownerUserID, name string, maxHP int, sharesInitiative bool) (*store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	f.nextID++
	pc := &store.PC{
		ID:               fmt.Sprintf("fake-companion-%d", f.nextID),
		OwnerUserID:      ownerUserID,
		Name:             name,
		Type:             "companion",
		MaxHP:            maxHP,
		ParentPCID:       parentPCID,
		SharesInitiative: sharesInitiative,
		Items:            []store.Item{},
	}
	f.byID[pc.ID] = pc
	return pc, nil
}

func (f *fakePCStore) OwnsAnyPC(ownerUserID string, pcIDs []string) (bool, error) {
	if f.forceErr != nil {
		return false, f.forceErr
	}
	for _, id := range pcIDs {
		if pc, ok := f.byID[id]; ok && pc.OwnerUserID == ownerUserID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakePCStore) ListPCsByOwner(ownerUserID string) ([]store.PC, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	var out []store.PC
	for _, pc := range f.byID {
		if pc.Type == "pc" && pc.OwnerUserID == ownerUserID {
			out = append(out, *pc)
		}
	}
	return out, nil
}
