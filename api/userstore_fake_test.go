package api

import (
	"errors"
	"fmt"

	"combatapp/store"
)

// fakeUserStore is an in-memory userStore for tests.
type fakeUserStore struct {
	byID       map[string]*store.User
	byUsername map[string]*store.User
	nextID     int
	forceErr   error
}

func newFakeUserStore(users ...store.User) *fakeUserStore {
	f := &fakeUserStore{byID: make(map[string]*store.User), byUsername: make(map[string]*store.User)}
	for i := range users {
		u := users[i]
		f.byID[u.ID] = &u
		f.byUsername[u.Username] = &u
	}
	return f
}

func (f *fakeUserStore) CreateUser(username, passphraseHash string) (*store.User, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	if _, exists := f.byUsername[username]; exists {
		return nil, errors.New("username taken")
	}
	f.nextID++
	u := &store.User{
		ID:             fmt.Sprintf("fake-user-%d", f.nextID),
		Username:       username,
		PassphraseHash: passphraseHash,
		DisplayName:    username,
	}
	f.byID[u.ID] = u
	f.byUsername[username] = u
	return u, nil
}

func (f *fakeUserStore) GetUserByUsername(username string) (*store.User, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	u, ok := f.byUsername[username]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) GetUserByID(id string) (*store.User, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	u, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) CreateSession(userID string) (*store.Session, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return &store.Session{Token: "fake-session-token", UserID: userID}, nil
}

func (f *fakeUserStore) DeleteSession(token string) error {
	return f.forceErr
}
