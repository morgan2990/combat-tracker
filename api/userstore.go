package api

import "combatapp/store"

// userStore is the subset of store.UserStore's methods called from this
// package. store.GlobalUsers is the production implementation; tests swap
// userDB for an in-memory fake.
type userStore interface {
	CreateUser(username, passphraseHash string) (*store.User, error)
	GetUserByUsername(username string) (*store.User, error)
	GetUserByID(id string) (*store.User, error)
	CreateSession(userID string) (*store.Session, error)
	DeleteSession(token string) error
}

var userDB userStore = &store.GlobalUsers
