package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

func TestNewPartyView_ResolvesFromKnownNames(t *testing.T) {
	p := &store.Party{
		ID:          "party1",
		Name:        "The Fellowship",
		MemberPCIDs: []string{"pc1", "pc2"},
	}
	knownNames := map[string]string{"pc1": "Frodo", "pc2": "Sam"}

	view, err := newPartyView(p, knownNames)
	if err != nil {
		t.Fatalf("newPartyView() error = %v, want nil", err)
	}
	if view.MemberNames["pc1"] != "Frodo" || view.MemberNames["pc2"] != "Sam" {
		t.Errorf("MemberNames = %+v, want {pc1:Frodo pc2:Sam}", view.MemberNames)
	}
	if view.Party != p {
		t.Error("view.Party should be the same pointer passed in")
	}
}

func TestNewPartyView_EmptyMembers(t *testing.T) {
	p := &store.Party{ID: "party1", Name: "Solo"}

	view, err := newPartyView(p, nil)
	if err != nil {
		t.Fatalf("newPartyView() error = %v, want nil", err)
	}
	if len(view.MemberNames) != 0 {
		t.Errorf("MemberNames = %+v, want empty", view.MemberNames)
	}
}

func TestWritePartyView_Success(t *testing.T) {
	p := &store.Party{ID: "party1", Name: "The Fellowship", MemberPCIDs: []string{"pc1"}}
	knownNames := map[string]string{"pc1": "Frodo"}
	w := httptest.NewRecorder()

	ok := writePartyView(w, http.StatusOK, p, knownNames)

	if !ok {
		t.Fatal("writePartyView() = false, want true")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
