package api

import "combatapp/store"

// roomStore is the subset of store.RoomStore's methods called directly from
// this package (not the ones room.Room's own methods call internally with a
// concrete *store.RoomStore parameter, which this seam can't reach).
// store.GlobalRooms is the production implementation; tests swap roomDB for
// an in-memory fake.
type roomStore interface {
	Delete(roomID string) error
	ListByOwner(ownerUserID string) ([]store.RoomSummary, error)
}

var roomDB roomStore = &store.GlobalRooms
