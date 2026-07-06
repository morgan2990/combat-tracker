package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

func TestCreatePC_RequiresAuth(t *testing.T) {
	withFakePCStore(t, newFakePCStore())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/pcs", nil) // no auth cookie

	CreatePC(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreatePC_ValidationFailure(t *testing.T) {
	withFakePCStore(t, newFakePCStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/pcs", []byte(`{"name":"","max_hp":10}`))

	CreatePC(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreatePC_Success(t *testing.T) {
	withFakePCStore(t, newFakePCStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/pcs", []byte(`{"name":"Aragorn","max_hp":45}`))

	CreatePC(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var pc store.PC
	if err := json.Unmarshal(w.Body.Bytes(), &pc); err != nil {
		t.Fatalf("response body is not a valid PC: %v", err)
	}
	if pc.Name != "Aragorn" || pc.MaxHP != 45 || pc.OwnerUserID != "owner1" {
		t.Errorf("created PC = %+v, want Name=Aragorn MaxHP=45 OwnerUserID=owner1", pc)
	}
}

func TestUpdatePC_NotFound(t *testing.T) {
	withFakePCStore(t, newFakePCStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/pcs/missing", []byte(`{"name":"Aragorn","max_hp":45}`))
	r.SetPathValue("id", "missing")

	UpdatePC(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdatePC_Forbidden(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else", Name: "Aragorn", MaxHP: 45}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/pcs/pc1", []byte(`{"name":"Aragorn","max_hp":45}`))
	r.SetPathValue("id", "pc1")

	UpdatePC(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdatePC_NegativeCurrencyRejected(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn", MaxHP: 45}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	body := []byte(`{"name":"Aragorn","max_hp":45,"currency":{"gp":-5}}`)
	r := authedRequest(http.MethodPut, "/api/pcs/pc1", body)
	r.SetPathValue("id", "pc1")

	UpdatePC(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdatePC_Success(t *testing.T) {
	fake := newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn", MaxHP: 45})
	withFakePCStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	body := []byte(`{"name":"Strider","max_hp":50}`)
	r := authedRequest(http.MethodPut, "/api/pcs/pc1", body)
	r.SetPathValue("id", "pc1")

	UpdatePC(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	updated, _ := fake.GetPCByID("pc1")
	if updated.Name != "Strider" || updated.MaxHP != 50 {
		t.Errorf("stored PC after update = %+v, want Name=Strider MaxHP=50", updated)
	}
}

func TestDeletePC_Success(t *testing.T) {
	fake := newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn", MaxHP: 45})
	withFakePCStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/pcs/pc1", nil)
	r.SetPathValue("id", "pc1")

	DeletePC(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got, _ := fake.GetPCByID("pc1"); got != nil {
		t.Error("PC should have been deleted")
	}
}

func TestDeletePC_Forbidden(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/pcs/pc1", nil)
	r.SetPathValue("id", "pc1")

	DeletePC(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGetPC_NotOwned_ReturnsNotFound(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/pcs/pc1", nil)
	r.SetPathValue("id", "pc1")

	GetPC(w, r)

	// Not owned is reported as 404, not 403, to avoid leaking existence of
	// other users' PCs.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetPC_WithCompanions(t *testing.T) {
	fake := newFakePCStore(
		store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn"},
		store.PC{ID: "comp1", OwnerUserID: "owner1", Type: "companion", ParentPCID: "pc1", Name: "Brego"},
	)
	withFakePCStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/pcs/pc1", nil)
	r.SetPathValue("id", "pc1")

	GetPC(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp pcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not valid: %v", err)
	}
	if len(resp.Companions) != 1 || resp.Companions[0].Name != "Brego" {
		t.Errorf("Companions = %+v, want [Brego]", resp.Companions)
	}
}

func TestCreateCompanion_ForbiddenWhenParentNotOwned(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/pcs/pc1/companions", []byte(`{"name":"Brego","max_hp":20}`))
	r.SetPathValue("id", "pc1")

	CreateCompanion(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCreateCompanion_Success(t *testing.T) {
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", OwnerUserID: "owner1", Name: "Aragorn"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/pcs/pc1/companions", []byte(`{"name":"Brego","max_hp":20}`))
	r.SetPathValue("id", "pc1")

	CreateCompanion(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var companion store.PC
	if err := json.Unmarshal(w.Body.Bytes(), &companion); err != nil {
		t.Fatalf("response body is not a valid PC: %v", err)
	}
	if companion.Name != "Brego" || companion.ParentPCID != "pc1" || companion.Type != "companion" {
		t.Errorf("companion = %+v, want Name=Brego ParentPCID=pc1 Type=companion", companion)
	}
}
