package api

import (
	"fmt"
	"slices"

	"combatapp/store"
)

// fakePartyStore is an in-memory partyStore for tests.
type fakePartyStore struct {
	byID     map[string]*store.Party
	nextID   int
	forceErr error
}

func newFakePartyStore(parties ...store.Party) *fakePartyStore {
	f := &fakePartyStore{byID: make(map[string]*store.Party)}
	for i := range parties {
		p := parties[i]
		f.byID[p.ID] = &p
	}
	return f
}

func (f *fakePartyStore) CreateParty(name string) (*store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	f.nextID++
	p := &store.Party{
		ID:          fmt.Sprintf("fake-party-%d", f.nextID),
		Name:        name,
		JoinCode:    fmt.Sprintf("CODE%d", f.nextID),
		MemberPCIDs: []string{},
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakePartyStore) GetPartyByID(id string) (*store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	p, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakePartyStore) GetPartyByJoinCode(joinCode string) (*store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	for _, p := range f.byID {
		if p.JoinCode == joinCode {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakePartyStore) AddMember(id, pcID string) (*store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	p, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	if slices.Contains(p.MemberPCIDs, pcID) {
		cp := *p
		return &cp, nil
	}
	p.MemberPCIDs = append(p.MemberPCIDs, pcID)
	cp := *p
	return &cp, nil
}

func (f *fakePartyStore) UpdateParty(id string, memberPCIDs []string, currency store.Currency) (*store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	p, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	if memberPCIDs == nil {
		memberPCIDs = []string{}
	}
	p.MemberPCIDs = memberPCIDs
	p.Currency = currency
	cp := *p
	return &cp, nil
}

func (f *fakePartyStore) ListPartiesByMemberPCIDs(pcIDs []string) ([]store.Party, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	if len(pcIDs) == 0 {
		return []store.Party{}, nil
	}
	var out []store.Party
	for _, p := range f.byID {
		for _, pcID := range p.MemberPCIDs {
			if slices.Contains(pcIDs, pcID) {
				out = append(out, *p)
				break
			}
		}
	}
	return out, nil
}
