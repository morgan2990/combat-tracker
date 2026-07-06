package api

import "combatapp/store"

// partyStore is the subset of store.PartyStore's methods called from this
// package. store.GlobalParties is the production implementation; tests
// swap partyDB for an in-memory fake.
type partyStore interface {
	CreateParty(name string) (*store.Party, error)
	GetPartyByID(id string) (*store.Party, error)
	GetPartyByJoinCode(joinCode string) (*store.Party, error)
	AddMember(id, pcID string) (*store.Party, error)
	UpdateParty(id string, memberPCIDs []string, currency store.Currency) (*store.Party, error)
	ListPartiesByMemberPCIDs(pcIDs []string) ([]store.Party, error)
}

var partyDB partyStore = &store.GlobalParties
