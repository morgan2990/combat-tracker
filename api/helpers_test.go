package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireUser_NoCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	userID, ok := requireUser(w, r)

	if ok {
		t.Fatal("requireUser() ok = true, want false with no session cookie")
	}
	if userID != "" {
		t.Errorf("requireUser() userID = %q, want empty", userID)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDecodeJSON_Valid(t *testing.T) {
	body := strings.NewReader(`{"name":"Aragorn","hp":42}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	var v struct {
		Name string `json:"name"`
		HP   int    `json:"hp"`
	}
	ok := decodeJSON(w, r, &v)

	if !ok {
		t.Fatal("decodeJSON() = false, want true for valid JSON")
	}
	if v.Name != "Aragorn" || v.HP != 42 {
		t.Errorf("decoded = %+v, want {Aragorn 42}", v)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want unchanged (200)", w.Code)
	}
}

func TestDecodeJSON_Invalid(t *testing.T) {
	body := strings.NewReader(`not json`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	var v struct{}
	ok := decodeJSON(w, r, &v)

	if ok {
		t.Fatal("decodeJSON() = true, want false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"room_id": "abc123"})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if got["room_id"] != "abc123" {
		t.Errorf("body room_id = %q, want abc123", got["room_id"])
	}
}

func TestIsValidEdition(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"5e", true},
		{"5.5e", true},
		{"", false},
		{"6e", false},
		{"5E", false},
	}
	for _, c := range cases {
		if got := isValidEdition(c.in); got != c.want {
			t.Errorf("isValidEdition(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRequireValidEdition(t *testing.T) {
	w := httptest.NewRecorder()
	if !requireValidEdition(w, "5e") {
		t.Error("requireValidEdition(\"5e\") = false, want true")
	}
	if w.Code != http.StatusOK {
		t.Errorf("valid edition should not write a response; status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	if requireValidEdition(w, "invalid") {
		t.Error(`requireValidEdition("invalid") = true, want false`)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestResolveEditionOrDefault(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"5e", "5e"},
		{"5.5e", "5.5e"},
		{"", "5e"},
		{"bogus", "5e"},
	}
	for _, c := range cases {
		if got := resolveEditionOrDefault(c.in); got != c.want {
			t.Errorf("resolveEditionOrDefault(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
