package api

import "combatapp/store"

// fakeMembershipStore is an in-memory membershipStore for tests.
type fakeMembershipStore struct {
	byUser        map[string][]store.RoomMembership
	deletedByRoom []string
	deletedByOne  [][2]string // [userID, roomID]
	forceErr      error
}

func newFakeMembershipStore() *fakeMembershipStore {
	return &fakeMembershipStore{byUser: make(map[string][]store.RoomMembership)}
}

func (f *fakeMembershipStore) Upsert(userID, roomID, pcID string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	f.byUser[userID] = append(f.byUser[userID], store.RoomMembership{UserID: userID, RoomID: roomID, LastPCID: pcID})
	return nil
}

func (f *fakeMembershipStore) ListByUser(userID string) ([]store.RoomMembership, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.byUser[userID], nil
}

func (f *fakeMembershipStore) DeleteByRoom(roomID string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	f.deletedByRoom = append(f.deletedByRoom, roomID)
	return nil
}

func (f *fakeMembershipStore) DeleteOne(userID, roomID string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	f.deletedByOne = append(f.deletedByOne, [2]string{userID, roomID})
	return nil
}
