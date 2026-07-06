package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
	"golang.org/x/crypto/bcrypt"
)

func TestSignUp_ValidationFailure(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/signup", bytes.NewReader([]byte(`{"username":"","passphrase":""}`)))

	SignUp(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSignUp_UsernameTaken(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "u1", Username: "aragorn"}))

	w := httptest.NewRecorder()
	body := []byte(`{"username":"aragorn","passphrase":"strider123"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/signup", bytes.NewReader(body))

	SignUp(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestSignUp_Success(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore())

	w := httptest.NewRecorder()
	body := []byte(`{"username":"aragorn","passphrase":"strider123"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/signup", bytes.NewReader(body))

	SignUp(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if len(w.Result().Cookies()) != 1 {
		t.Error("expected a session cookie to be set")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-pass"), bcrypt.DefaultCost)
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "u1", Username: "aragorn", PassphraseHash: string(hash)}))

	w := httptest.NewRecorder()
	body := []byte(`{"username":"aragorn","passphrase":"wrong-pass"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))

	Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_UnknownUsername(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore())

	w := httptest.NewRecorder()
	body := []byte(`{"username":"nobody","passphrase":"whatever"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))

	Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-pass"), bcrypt.DefaultCost)
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "u1", Username: "aragorn", PassphraseHash: string(hash)}))

	w := httptest.NewRecorder()
	body := []byte(`{"username":"aragorn","passphrase":"correct-pass"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))

	Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLogout_ClearsCookieRegardless(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/logout", nil) // no cookie at all

	Logout(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestMe_Unauthorized_UnknownUser(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore()) // requireUser succeeds via auth fake, but user lookup misses
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/me", nil)

	Me(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMe_Success(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "aragorn", DisplayName: "Aragorn"}))
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn", Type: "pc"}))
	withFakeRoomStore(t, newFakeRoomStore())
	fakeMemberships := newFakeMembershipStore()
	fakeMemberships.byUser["owner1"] = []store.RoomMembership{{UserID: "owner1", RoomID: "room1"}}
	withFakeMembershipStore(t, fakeMemberships)
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/me", nil)

	Me(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	pcs := resp["pcs"].([]any)
	if len(pcs) != 1 {
		t.Errorf("pcs = %v, want 1 entry", pcs)
	}
	recentRooms := resp["recent_rooms"].([]any)
	if len(recentRooms) != 1 {
		t.Errorf("recent_rooms = %v, want 1 entry", recentRooms)
	}
}
