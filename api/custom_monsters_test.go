package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

func withFakeUserStore(t *testing.T, fake *fakeUserStore) {
	t.Helper()
	original := userDB
	userDB = fake
	t.Cleanup(func() { userDB = original })
}

func withFakeCustomMonsterStore(t *testing.T, fake *fakeCustomMonsterStore) {
	t.Helper()
	original := customMonsterDB
	customMonsterDB = fake
	t.Cleanup(func() { customMonsterDB = original })
}

func TestCreateCustomMonster_RequiresKnownUser(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore()) // no users at all
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`{"name":"Goblin","edition":"5e","max_hp":7}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateCustomMonster_ValidationFailure(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`{"name":"","edition":"5e","max_hp":7}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateCustomMonster_InvalidEdition(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`{"name":"Goblin","edition":"3e","max_hp":7}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateCustomMonster_Success(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`{"name":"Goblin","edition":"5e","max_hp":7}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var m store.CustomMonster
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("response body is not valid: %v", err)
	}
	if m.Name != "Goblin" || m.OwnerID != "owner1" || m.OwnerDisplayName != "DM One" {
		t.Errorf("created monster = %+v, want Name=Goblin OwnerID=owner1 OwnerDisplayName=\"DM One\"", m)
	}
}

func TestGetCustomMonster_NotFound(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/missing", nil)
	r.SetPathValue("id", "missing")

	GetCustomMonster(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCustomMonster_PrivateForbiddenForNonOwner(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", Name: "Goblin", Private: true, OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	GetCustomMonster(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGetCustomMonster_PublicVisibleToAnyone(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", Name: "Goblin", Private: false, OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	GetCustomMonster(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUpdateCustomMonster_ForbiddenForNonOwner(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", Name: "Goblin", OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"Goblin King","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdateCustomMonster_PreservesOwnerAndPDFKey(t *testing.T) {
	fake := newFakeCustomMonsterStore(store.CustomMonster{
		ID: "m1", Name: "Goblin", OwnerID: "owner1", OwnerDisplayName: "DM One", PDFObjectKey: "custom-monsters/m1.pdf",
	})
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"Goblin King","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	updated, _ := fake.GetCustomMonsterByID("m1")
	if updated.Name != "Goblin King" || updated.OwnerID != "owner1" || updated.PDFObjectKey != "custom-monsters/m1.pdf" {
		t.Errorf("updated monster = %+v, want Name changed but OwnerID/PDFObjectKey preserved", updated)
	}
}

func TestDeleteCustomMonster_ForbiddenForNonOwner(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestDeleteCustomMonster_Success(t *testing.T) {
	fake := newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"})
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got, _ := fake.GetCustomMonsterByID("m1"); got != nil {
		t.Error("monster should have been deleted")
	}
}

func TestListMyCustomMonsters_FiltersByOwnerAndEdition(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(
		store.CustomMonster{ID: "m1", OwnerID: "owner1", Edition: "5e"},
		store.CustomMonster{ID: "m2", OwnerID: "owner1", Edition: "5.5e"},
		store.CustomMonster{ID: "m3", OwnerID: "someone-else", Edition: "5e"},
	))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/mine?edition=5e", nil)

	ListMyCustomMonsters(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var monsters []store.CustomMonster
	if err := json.Unmarshal(w.Body.Bytes(), &monsters); err != nil {
		t.Fatalf("response body is not valid: %v", err)
	}
	if len(monsters) != 1 || monsters[0].ID != "m1" {
		t.Errorf("monsters = %+v, want exactly [m1]", monsters)
	}
}

func TestStreamCustomMonsterPDF_NotFound(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/missing/pdf", nil)
	r.SetPathValue("id", "missing")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStreamCustomMonsterPDF_NoPDFAttached(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1", PDFObjectKey: ""}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1/pdf", nil)
	r.SetPathValue("id", "m1")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStreamCustomMonsterPDF_PrivateForbiddenForNonOwner(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{
		ID: "m1", Private: true, OwnerID: "someone-else", PDFObjectKey: "custom-monsters/m1.pdf",
	}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1/pdf", nil)
	r.SetPathValue("id", "m1")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
