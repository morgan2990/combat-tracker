package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/store"
)

func withFakeMonsterStore(t *testing.T, fake *fakeMonsterStore) {
	t.Helper()
	original := monsterDB
	monsterDB = fake
	t.Cleanup(func() { monsterDB = original })
}

func TestGetMonster_NoNameRejected(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters/", nil)
	r.SetPathValue("name", "")

	GetMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetMonster_NotFound(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters/Goblin", nil)
	r.SetPathValue("name", "Goblin")

	GetMonster(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetMonster_Success(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore(store.Monster{Name: "Goblin", MaxHP: 7}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters/Goblin", nil)
	r.SetPathValue("name", "Goblin")

	GetMonster(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchMonsters_RequiresQueryAndEdition(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore())
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/monsters/search?q=&edition=5e", nil)

	SearchMonsters(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSearchMonsters_InvalidEdition(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore())
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/monsters/search?q=Gob&edition=3e", nil)

	SearchMonsters(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSearchMonsters_Success(t *testing.T) {
	fake := newFakeMonsterStore()
	fake.searchHits = []store.MonsterHit{{Name: "Goblin"}}
	withFakeMonsterStore(t, fake)
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/monsters/search?q=Gob&edition=5e", nil)

	SearchMonsters(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestStreamMonsterPDF_NoNameRejected(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters//pdf", nil)
	r.SetPathValue("name", "")

	StreamMonsterPDF(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStreamMonsterPDF_NotFound(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters/Goblin/pdf", nil)
	r.SetPathValue("name", "Goblin")

	StreamMonsterPDF(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStreamMonsterPDF_NoPDFAttached(t *testing.T) {
	withFakeMonsterStore(t, newFakeMonsterStore(store.Monster{Name: "Goblin", PDFObjectKey: ""}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/monsters/Goblin/pdf", nil)
	r.SetPathValue("name", "Goblin")

	StreamMonsterPDF(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
