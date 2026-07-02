package api

import (
	"net/http"

	"combatapp/auth"
	"combatapp/store"
	"golang.org/x/crypto/bcrypt"
)

type authPayload struct {
	Username   string `json:"username"`
	Passphrase string `json:"passphrase"`
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	var body authPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Username == "" || body.Passphrase == "" {
		http.Error(w, "username and passphrase required", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Passphrase), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not hash passphrase", http.StatusInternalServerError)
		return
	}
	user, err := store.GlobalUsers.CreateUser(body.Username, string(hash))
	if err != nil {
		http.Error(w, "username taken", http.StatusConflict)
		return
	}
	sess, err := store.GlobalUsers.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sess.Token)
	writeJSON(w, http.StatusCreated, map[string]string{"username": user.Username})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var body authPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	user, err := store.GlobalUsers.GetUserByUsername(body.Username)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PassphraseHash), []byte(body.Passphrase)) != nil {
		http.Error(w, "invalid username or passphrase", http.StatusUnauthorized)
		return
	}
	sess, err := store.GlobalUsers.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sess.Token)
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("combatapp_session"); err == nil && c.Value != "" {
		store.GlobalUsers.DeleteSession(c.Value)
	}
	auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	user, err := store.GlobalUsers.GetUserByID(userID)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	pcs, err := store.Global.ListPCsByOwner(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pcs == nil {
		pcs = []store.PC{}
	}
	rooms, err := store.GlobalRooms.ListByOwner(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if rooms == nil {
		rooms = []store.RoomSummary{}
	}
	memberships, err := store.GlobalMemberships.ListByUser(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if memberships == nil {
		memberships = []store.RoomMembership{}
	}
	pcIDs := make([]string, len(pcs))
	for i, pc := range pcs {
		pcIDs[i] = pc.ID
	}
	parties, err := store.GlobalParties.ListPartiesByMemberPCIDs(pcIDs)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if parties == nil {
		parties = []store.Party{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":         map[string]string{"username": user.Username, "display_name": user.DisplayName},
		"rooms":        rooms,
		"pcs":          pcs,
		"recent_rooms": memberships,
		"parties":      parties,
	})
}
