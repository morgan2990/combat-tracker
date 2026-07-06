package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

func withFakeEncounterStore(t *testing.T, fake *fakeEncounterStore) {
	t.Helper()
	original := encounterDB
	encounterDB = fake
	t.Cleanup(func() { encounterDB = original })
}

func TestCreateEncounter_ValidationFailure(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/encounters", []byte(`{"name":"","edition":"5e"}`))

	CreateEncounter(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateEncounter_InvalidEdition(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/encounters", []byte(`{"name":"Goblin Ambush","edition":"3e"}`))

	CreateEncounter(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateEncounter_Success(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/encounters", []byte(`{"name":"Goblin Ambush","edition":"5e"}`))

	CreateEncounter(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var e store.Encounter
	json.Unmarshal(w.Body.Bytes(), &e)
	if e.Name != "Goblin Ambush" || e.OwnerID != "owner1" {
		t.Errorf("created encounter = %+v, want Name=Goblin Ambush OwnerID=owner1", e)
	}
}

func TestGetEncounter_NotFound(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/encounters/missing", nil)
	r.SetPathValue("id", "missing")

	GetEncounter(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetEncounter_ForbiddenForNonOwner(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore(store.Encounter{ID: "e1", Name: "Ambush", OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/encounters/e1", nil)
	r.SetPathValue("id", "e1")

	GetEncounter(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGetEncounter_Success(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore(store.Encounter{ID: "e1", Name: "Ambush", OwnerID: "owner1"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/encounters/e1", nil)
	r.SetPathValue("id", "e1")

	GetEncounter(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUpdateEncounter_ForbiddenForNonOwner(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore(store.Encounter{ID: "e1", Name: "Ambush", OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/encounters/e1", []byte(`{"name":"Ambush 2","edition":"5e"}`))
	r.SetPathValue("id", "e1")

	UpdateEncounter(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdateEncounter_PreservesOwner(t *testing.T) {
	fake := newFakeEncounterStore(store.Encounter{ID: "e1", Name: "Ambush", OwnerID: "owner1"})
	withFakeEncounterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/encounters/e1", []byte(`{"name":"Ambush 2","edition":"5e"}`))
	r.SetPathValue("id", "e1")

	UpdateEncounter(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	updated, _ := fake.GetEncounterByID("e1")
	if updated.Name != "Ambush 2" || updated.OwnerID != "owner1" {
		t.Errorf("updated encounter = %+v, want Name=Ambush 2 OwnerID=owner1 (preserved)", updated)
	}
}

func TestDeleteEncounter_ForbiddenForNonOwner(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore(store.Encounter{ID: "e1", OwnerID: "someone-else"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/encounters/e1", nil)
	r.SetPathValue("id", "e1")

	DeleteEncounter(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestDeleteEncounter_Success(t *testing.T) {
	fake := newFakeEncounterStore(store.Encounter{ID: "e1", OwnerID: "owner1"})
	withFakeEncounterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/encounters/e1", nil)
	r.SetPathValue("id", "e1")

	DeleteEncounter(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got, _ := fake.GetEncounterByID("e1"); got != nil {
		t.Error("encounter should have been deleted")
	}
}

func TestListMyEncounters_FiltersByOwnerAndEdition(t *testing.T) {
	withFakeEncounterStore(t, newFakeEncounterStore(
		store.Encounter{ID: "e1", OwnerID: "owner1", Edition: "5e"},
		store.Encounter{ID: "e2", OwnerID: "owner1", Edition: "5.5e"},
		store.Encounter{ID: "e3", OwnerID: "someone-else", Edition: "5e"},
	))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/encounters/mine?edition=5e", nil)

	ListMyEncounters(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var encounters []store.Encounter
	json.Unmarshal(w.Body.Bytes(), &encounters)
	if len(encounters) != 1 || encounters[0].ID != "e1" {
		t.Errorf("encounters = %+v, want exactly [e1]", encounters)
	}
}
