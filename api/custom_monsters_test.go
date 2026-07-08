package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"combatapp/store"
)

// failOnUpdateCustomMonsterStore embeds a real fake so GetCustomMonsterByID
// (called first by UpdateCustomMonster) succeeds, but forces the subsequent
// UpdateCustomMonster call to fail — fakeCustomMonsterStore's single
// forceErr flag can't express "succeed on read, fail on write" for a
// handler that calls both.
type failOnUpdateCustomMonsterStore struct{ *fakeCustomMonsterStore }

func (failOnUpdateCustomMonsterStore) UpdateCustomMonster(string, store.CustomMonster) (store.CustomMonster, error) {
	return store.CustomMonster{}, errors.New("forced update error")
}

// failOnDeleteCustomMonsterStore is the DeleteCustomMonster equivalent of
// failOnUpdateCustomMonsterStore, above.
type failOnDeleteCustomMonsterStore struct{ *fakeCustomMonsterStore }

func (failOnDeleteCustomMonsterStore) DeleteCustomMonster(string) error {
	return errors.New("forced delete error")
}

// multipartCustomMonsterRequest builds an authenticated multipart/form-data
// POST to /api/custom-monsters with the given text fields and, optionally,
// a "pdf" file part.
func multipartCustomMonsterRequest(t *testing.T, fields map[string]string, includePDF bool) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("WriteField(%q) error = %v", k, err)
		}
	}
	if includePDF {
		fw, err := mw.CreateFormFile("pdf", "monster.pdf")
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := fw.Write([]byte("%PDF-1.4 fake pdf content")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/custom-monsters", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.AddCookie(&http.Cookie{Name: "combatapp_session", Value: "test-token"})
	return r
}

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

// --- CreateCustomMonster: auth/validation/db-error branches on the JSON path ---

func TestCreateCustomMonster_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/custom-monsters", strings.NewReader(`{}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateCustomMonster_MalformedJSON(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`not json`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateCustomMonster_DatabaseError(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced create error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/custom-monsters", []byte(`{"name":"Goblin","edition":"5e","max_hp":7}`))

	CreateCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- CreateCustomMonster: multipart/form-data (PDF upload) path ---

func TestCreateCustomMonster_Multipart_TooLarge(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("pdf", "monster.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := fw.Write(make([]byte, 21<<20)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/custom-monsters", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.AddCookie(&http.Cookie{Name: "combatapp_session", Value: "test-token"})

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}
}

func TestCreateCustomMonster_Multipart_InvalidForm(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	r := httptest.NewRequest(http.MethodPost, "/api/custom-monsters", strings.NewReader("not a valid multipart body"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	r.AddCookie(&http.Cookie{Name: "combatapp_session", Value: "test-token"})

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateCustomMonster_Multipart_ValidationFailure(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	r := multipartCustomMonsterRequest(t, map[string]string{"name": "", "edition": "5e", "max_hp": "7"}, false)

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateCustomMonster_Multipart_InvalidEdition(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	r := multipartCustomMonsterRequest(t, map[string]string{"name": "Goblin", "edition": "3e", "max_hp": "7"}, false)

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateCustomMonster_Multipart_MissingPDFFile(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	r := multipartCustomMonsterRequest(t, map[string]string{"name": "Goblin", "edition": "5e", "max_hp": "7"}, false)

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestCreateCustomMonster_Multipart_UploadFailure exercises the full
// multipart parse + validate + initiative_modifier + file-open path, then
// hits store.UploadCustomMonsterPDF's own nil-guard (no Minio configured in
// this unit test) — the same "storage error" 502 branch production hits
// when Minio is genuinely unreachable. The success path past this point
// requires a real Minio and is intentionally not covered here.
func TestCreateCustomMonster_Multipart_UploadFailure(t *testing.T) {
	withFakeUserStore(t, newFakeUserStore(store.User{ID: "owner1", Username: "dm1", DisplayName: "DM One"}))
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	r := multipartCustomMonsterRequest(t, map[string]string{
		"name": "Goblin", "edition": "5e", "max_hp": "7",
		"private": "true", "initiative_modifier": "3",
	}, true)

	w := httptest.NewRecorder()
	CreateCustomMonster(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadGateway, w.Body.String())
	}
}

// --- GetCustomMonster ---

func TestGetCustomMonster_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	GetCustomMonster(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetCustomMonster_DatabaseError(t *testing.T) {
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced get error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	GetCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- UpdateCustomMonster ---

func TestUpdateCustomMonster_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/custom-monsters/m1", strings.NewReader(`{}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUpdateCustomMonster_DatabaseErrorOnRead(t *testing.T) {
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced read error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"Goblin King","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestUpdateCustomMonster_NotFound(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/missing", []byte(`{"name":"Goblin King","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "missing")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateCustomMonster_MalformedJSON(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`not json`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateCustomMonster_ValidationFailure(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateCustomMonster_InvalidEdition(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"Goblin King","edition":"3e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateCustomMonster_DatabaseErrorOnWrite(t *testing.T) {
	fake := failOnUpdateCustomMonsterStore{newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"})}
	withFakeCustomMonsterStore(t, fake.fakeCustomMonsterStore) // registers cleanup to restore the real store
	customMonsterDB = fake
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/custom-monsters/m1", []byte(`{"name":"Goblin King","edition":"5e","max_hp":20}`))
	r.SetPathValue("id", "m1")

	UpdateCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- DeleteCustomMonster ---

func TestDeleteCustomMonster_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDeleteCustomMonster_DatabaseErrorOnRead(t *testing.T) {
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced read error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDeleteCustomMonster_NotFound(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/custom-monsters/missing", nil)
	r.SetPathValue("id", "missing")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteCustomMonster_DatabaseErrorOnDelete(t *testing.T) {
	fake := failOnDeleteCustomMonsterStore{newFakeCustomMonsterStore(store.CustomMonster{ID: "m1", OwnerID: "owner1"})}
	withFakeCustomMonsterStore(t, fake.fakeCustomMonsterStore) // registers cleanup to restore the real store
	customMonsterDB = fake
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/custom-monsters/m1", nil)
	r.SetPathValue("id", "m1")

	DeleteCustomMonster(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- ListMyCustomMonsters ---

func TestListMyCustomMonsters_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/custom-monsters/mine", nil)

	ListMyCustomMonsters(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestListMyCustomMonsters_DatabaseError(t *testing.T) {
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced list error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/mine", nil)

	ListMyCustomMonsters(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestListMyCustomMonsters_NoneOwned_ReturnsEmptyArrayNotNull(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore())
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/mine", nil)

	ListMyCustomMonsters(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(w.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

// --- StreamCustomMonsterPDF ---

func TestStreamCustomMonsterPDF_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/custom-monsters/m1/pdf", nil)
	r.SetPathValue("id", "m1")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestStreamCustomMonsterPDF_DatabaseError(t *testing.T) {
	fake := newFakeCustomMonsterStore()
	fake.forceErr = errors.New("forced get error")
	withFakeCustomMonsterStore(t, fake)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1/pdf", nil)
	r.SetPathValue("id", "m1")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestStreamCustomMonsterPDF_StorageUnavailable covers the case where the
// monster record is found (and visible to the requester) but the
// underlying object store fetch fails — the same 404 branch production
// hits when Minio is unreachable, reproduced here via Minio's own
// nil-client guard (no Minio configured in this unit test). Streaming an
// actual PDF body requires a real Minio and is intentionally not covered
// here.
func TestStreamCustomMonsterPDF_StorageUnavailable(t *testing.T) {
	withFakeCustomMonsterStore(t, newFakeCustomMonsterStore(store.CustomMonster{
		ID: "m1", OwnerID: "owner1", PDFObjectKey: "custom-monsters/m1.pdf",
	}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/custom-monsters/m1/pdf", nil)
	r.SetPathValue("id", "m1")

	StreamCustomMonsterPDF(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
