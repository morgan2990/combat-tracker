// Package auth provides cookie-based session helpers shared by the REST API
// and the WebSocket handler.
package auth

import (
	"net/http"
	"os"
	"time"

	"combatapp/store"
)

const cookieName = "combatapp_session"

// secureCookies reports whether the session cookie should carry the Secure
// attribute. Defaults to true; set COOKIE_SECURE=false for local/LAN HTTP testing.
func secureCookies() bool {
	return os.Getenv("COOKIE_SECURE") != "false"
}

// SetSessionCookie writes the session cookie for a newly created session.
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(90 * 24 * time.Hour),
	})
}

// ClearSessionCookie removes the session cookie (logout).
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// ResolveUserID reads the session cookie from r, validates it against the
// session store, and returns the authenticated user's id. ok is false if
// there is no valid, non-expired session.
func ResolveUserID(r *http.Request) (userID string, ok bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	sess, err := store.GlobalUsers.ResolveSession(c.Value)
	if err != nil || sess == nil {
		return "", false
	}
	return sess.UserID, true
}
