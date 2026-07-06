package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"combatapp/store"
)

// fakeSessionResolver is an in-memory SessionResolver for tests.
type fakeSessionResolver struct {
	sessions map[string]*store.Session
	forceErr error
}

func (f fakeSessionResolver) ResolveSession(token string) (*store.Session, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.sessions[token], nil
}

// withFakeSessions swaps Sessions for fake for the duration of the test.
func withFakeSessions(t *testing.T, fake SessionResolver) {
	t.Helper()
	original := Sessions
	Sessions = fake
	t.Cleanup(func() { Sessions = original })
}

func requestWithCookie(value string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if value != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: value})
	}
	return r
}

func TestResolveUserID_NoCookie(t *testing.T) {
	userID, ok := ResolveUserID(httptest.NewRequest(http.MethodGet, "/", nil))
	if ok || userID != "" {
		t.Errorf("ResolveUserID() = (%q, %v), want (\"\", false)", userID, ok)
	}
}

func TestResolveUserID_EmptyCookieValue(t *testing.T) {
	userID, ok := ResolveUserID(requestWithCookie(""))
	if ok || userID != "" {
		t.Errorf("ResolveUserID() = (%q, %v), want (\"\", false)", userID, ok)
	}
}

func TestResolveUserID_ValidSession(t *testing.T) {
	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{
		"tok-good": {Token: "tok-good", UserID: "user1", ExpiresAt: time.Now().Add(time.Hour)},
	}})

	userID, ok := ResolveUserID(requestWithCookie("tok-good"))
	if !ok || userID != "user1" {
		t.Errorf("ResolveUserID() = (%q, %v), want (\"user1\", true)", userID, ok)
	}
}

func TestResolveUserID_UnknownToken(t *testing.T) {
	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{}})

	userID, ok := ResolveUserID(requestWithCookie("tok-unknown"))
	if ok || userID != "" {
		t.Errorf("ResolveUserID() = (%q, %v), want (\"\", false)", userID, ok)
	}
}

func TestResolveUserID_StoreError(t *testing.T) {
	withFakeSessions(t, fakeSessionResolver{forceErr: errors.New("connection reset")})

	userID, ok := ResolveUserID(requestWithCookie("tok-good"))
	if ok || userID != "" {
		t.Errorf("ResolveUserID() = (%q, %v), want (\"\", false) on a store error", userID, ok)
	}
}

func TestSecureCookies_DefaultsTrue(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	if !secureCookies() {
		t.Error("secureCookies() = false, want true when COOKIE_SECURE is unset")
	}
}

func TestSecureCookies_FalseOptOut(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	if secureCookies() {
		t.Error("secureCookies() = true, want false when COOKIE_SECURE=false")
	}
}

func TestSetSessionCookie(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	w := httptest.NewRecorder()
	SetSessionCookie(w, "tok-123")

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName || c.Value != "tok-123" {
		t.Errorf("cookie = %+v, want Name=%s Value=tok-123", c, cookieName)
	}
	if !c.HttpOnly || !c.Secure {
		t.Errorf("cookie HttpOnly=%v Secure=%v, want both true", c.HttpOnly, c.Secure)
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearSessionCookie(w)

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	if cookies[0].MaxAge >= 0 {
		t.Errorf("cookie MaxAge = %d, want negative (delete)", cookies[0].MaxAge)
	}
}
