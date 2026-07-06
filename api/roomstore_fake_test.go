package api

import "combatapp/store"

// fakeRoomStore is an in-memory roomStore for tests.
type fakeRoomStore struct {
	byOwner  map[string][]store.RoomSummary
	deleted  []string
	forceErr error
}

func newFakeRoomStore() *fakeRoomStore {
	return &fakeRoomStore{byOwner: make(map[string][]store.RoomSummary)}
}

func (f *fakeRoomStore) Delete(roomID string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	f.deleted = append(f.deleted, roomID)
	return nil
}

func (f *fakeRoomStore) ListByOwner(ownerUserID string) ([]store.RoomSummary, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.byOwner[ownerUserID], nil
}
