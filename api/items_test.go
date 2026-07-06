package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func withFakeCatalogItemStore(t *testing.T, fake *fakeCatalogItemStore) {
	t.Helper()
	original := catalogItemDB
	catalogItemDB = fake
	t.Cleanup(func() { catalogItemDB = original })
}

func TestSearchItems_RequiresQueryAndEdition(t *testing.T) {
	withFakeCatalogItemStore(t, newFakeCatalogItemStore())
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/search?q=&edition=5e", nil)

	SearchItems(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSearchItems_InvalidEdition(t *testing.T) {
	withFakeCatalogItemStore(t, newFakeCatalogItemStore())
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/search?q=sword&edition=3e", nil)

	SearchItems(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSearchItems_Success(t *testing.T) {
	fake := newFakeCatalogItemStore()
	fake.searchHits = []store.CatalogItemHit{{Name: "Longsword"}}
	withFakeCatalogItemStore(t, fake)
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/search?q=sword&edition=5e", nil)

	SearchItems(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetItem_NoIDRejected(t *testing.T) {
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/", nil)
	r.SetPathValue("id", "")

	GetItem(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	withFakeCatalogItemStore(t, newFakeCatalogItemStore())
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/000000000000000000000000", nil)
	r.SetPathValue("id", "000000000000000000000000")

	GetItem(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetItem_Success(t *testing.T) {
	id := bson.NewObjectID()
	withFakeCatalogItemStore(t, newFakeCatalogItemStore(store.CatalogItem{ID: id, Name: "Longsword"}))
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/items/"+id.Hex(), nil)
	r.SetPathValue("id", id.Hex())

	GetItem(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
