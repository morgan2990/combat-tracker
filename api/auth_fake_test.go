package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/auth"
	"combatapp/store"
)

// alwaysAuthedAs is a fake auth.SessionResolver that treats any non-empty
// session token as belonging to userID — tests don't need to synchronize a
// real token value between the request cookie and the resolver.
type alwaysAuthedAs struct{ userID string }

func (f alwaysAuthedAs) ResolveSession(token string) (*store.Session, error) {
	if token == "" {
		return nil, nil
	}
	return &store.Session{UserID: f.userID}, nil
}

// withFakeAuth makes requests built with authedRequest resolve to userID for
// the duration of the test.
func withFakeAuth(t *testing.T, userID string) {
	t.Helper()
	original := auth.Sessions
	auth.Sessions = alwaysAuthedAs{userID: userID}
	t.Cleanup(func() { auth.Sessions = original })
}

// authedRequest builds a request carrying a session cookie that
// withFakeAuth's resolver will honor. body may be nil.
func authedRequest(method, target string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	r.AddCookie(&http.Cookie{Name: "combatapp_session", Value: "test-token"})
	return r
}
