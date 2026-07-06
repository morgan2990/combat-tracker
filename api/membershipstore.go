package api

import "combatapp/store"

// membershipStore is the subset of store.MembershipStore's methods called
// from this package. store.GlobalMemberships is the production
// implementation; tests swap membershipDB for an in-memory fake.
type membershipStore interface {
	Upsert(userID, roomID, pcID string) error
	ListByUser(userID string) ([]store.RoomMembership, error)
	DeleteByRoom(roomID string) error
	DeleteOne(userID, roomID string) error
}

var membershipDB membershipStore = &store.GlobalMemberships
