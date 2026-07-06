package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

// withFakePCStore swaps pcDB for fake for the duration of the test.
func withFakePCStore(t *testing.T, fake *fakePCStore) {
	t.Helper()
	original := pcDB
	pcDB = fake
	t.Cleanup(func() { pcDB = original })
}

func TestRequireOwnedPC_NotFound(t *testing.T) {
	withFakePCStore(t, newFakePCStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := requireOwnedPC(w, r, "missing-id", "user1")

	if ok {
		t.Fatal("requireOwnedPC() ok = true, want false for a missing PC")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRequireOwnedPC_Forbidden(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := requireOwnedPC(w, r, "pc1", "someone-else")

	if ok {
		t.Fatal("requireOwnedPC() ok = true, want false for a PC owned by someone else")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireOwnedPC_Success(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	pc, ok := requireOwnedPC(w, r, "pc1", "owner1")

	if !ok {
		t.Fatal("requireOwnedPC() ok = false, want true for the owner's own PC")
	}
	if pc.Name != "Aragorn" {
		t.Errorf("pc.Name = %q, want Aragorn", pc.Name)
	}
	if w.Code != http.StatusOK {
		t.Errorf("requireOwnedPC should not write a response on success; status = %d", w.Code)
	}
}

func TestRequireOwnedPC_DatabaseError(t *testing.T) {
	fake := newFakePCStore()
	fake.forceErr = errors.New("connection reset")
	withFakePCStore(t, fake)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := requireOwnedPC(w, r, "pc1", "owner1")

	if ok {
		t.Fatal("requireOwnedPC() ok = true, want false on a database error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestNewPartyView_ResolvesUnknownMemberFromStore(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", Name: "Legolas"}))

	p := &store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}

	view, err := newPartyView(p, nil)
	if err != nil {
		t.Fatalf("newPartyView() error = %v, want nil", err)
	}
	if view.MemberNames["pc1"] != "Legolas" {
		t.Errorf(`MemberNames["pc1"] = %q, want "Legolas"`, view.MemberNames["pc1"])
	}
}

func TestNewPartyView_SkipsMemberThatNoLongerExists(t *testing.T) {
	withFakePCStore(t, newFakePCStore()) // empty — pc1 was deleted

	p := &store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}

	view, err := newPartyView(p, nil)
	if err != nil {
		t.Fatalf("newPartyView() error = %v, want nil", err)
	}
	if _, ok := view.MemberNames["pc1"]; ok {
		t.Error("MemberNames should not contain an entry for a PC that no longer exists")
	}
}

func TestNewPartyView_PropagatesStoreError(t *testing.T) {
	fake := newFakePCStore()
	fake.forceErr = errors.New("connection reset")
	withFakePCStore(t, fake)

	p := &store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}

	_, err := newPartyView(p, nil)
	if err == nil {
		t.Fatal("newPartyView() error = nil, want the store error to propagate")
	}
}
