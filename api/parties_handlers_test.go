package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

// withFakePartyStore swaps partyDB for fake for the duration of the test.
func withFakePartyStore(t *testing.T, fake *fakePartyStore) {
	t.Helper()
	original := partyDB
	partyDB = fake
	t.Cleanup(func() { partyDB = original })
}

func TestCreateParty_ValidationFailure(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties", []byte(`{"name":""}`))

	CreateParty(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateParty_Success(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties", []byte(`{"name":"The Fellowship"}`))

	CreateParty(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var view partyView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatalf("response body is not valid: %v", err)
	}
	if view.Name != "The Fellowship" {
		t.Errorf("party.Name = %q, want %q", view.Name, "The Fellowship")
	}
}

func TestGetParty_NotFound(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/parties/missing", nil)
	r.SetPathValue("id", "missing")

	GetParty(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetParty_Success(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/parties/party1", nil)
	r.SetPathValue("id", "party1")

	GetParty(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUpdateParty_NegativeCurrencyRejected(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/parties/party1", []byte(`{"currency":{"gp":-5}}`))
	r.SetPathValue("id", "party1")

	UpdateParty(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateParty_ForbiddenWhenNotAMember(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else"}))
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship", MemberPCIDs: []string{"pc1"}}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/parties/party1", []byte(`{"member_pc_ids":["pc1"]}`))
	r.SetPathValue("id", "party1")

	UpdateParty(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdateParty_AllowedWhenNoExistingMembers(t *testing.T) {
	withFakePCStore(t, newFakePCStore())
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/parties/party1", []byte(`{"member_pc_ids":["pc1"]}`))
	r.SetPathValue("id", "party1")

	UpdateParty(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUpdateParty_AllowedWhenOwningAMember(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1"}))
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship", MemberPCIDs: []string{"pc1"}}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/parties/party1", []byte(`{"member_pc_ids":["pc1","pc2"],"currency":{"gp":10}}`))
	r.SetPathValue("id", "party1")

	UpdateParty(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestJoinParty_MissingFields(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties/join", []byte(`{"join_code":""}`))

	JoinParty(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestJoinParty_UnknownJoinCode(t *testing.T) {
	withFakePartyStore(t, newFakePartyStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties/join", []byte(`{"join_code":"ZZZZZ","pc_id":"pc1"}`))

	JoinParty(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestJoinParty_ForbiddenWhenPCNotOwned(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else"}))
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship", JoinCode: "ABCDE"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties/join", []byte(`{"join_code":"ABCDE","pc_id":"pc1"}`))

	JoinParty(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestJoinParty_Success(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Frodo"}))
	fake := newFakePartyStore(store.Party{ID: "party1", Name: "The Fellowship", JoinCode: "ABCDE"})
	withFakePartyStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/parties/join", []byte(`{"join_code":"ABCDE","pc_id":"pc1"}`))

	JoinParty(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var view partyView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatalf("response body is not valid: %v", err)
	}
	if view.MemberNames["pc1"] != "Frodo" {
		t.Errorf(`MemberNames["pc1"] = %q, want "Frodo"`, view.MemberNames["pc1"])
	}
	updated, _ := fake.GetPartyByID("party1")
	if len(updated.MemberPCIDs) != 1 || updated.MemberPCIDs[0] != "pc1" {
		t.Errorf("stored party MemberPCIDs = %v, want [pc1]", updated.MemberPCIDs)
	}
}
