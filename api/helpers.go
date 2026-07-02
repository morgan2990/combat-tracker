package api

import (
	"encoding/json"
	"net/http"

	"combatapp/auth"
)

// requireUser resolves the authenticated user for the request, writing a 401
// and returning ok=false if there isn't one.
func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return "", false
	}
	return userID, true
}

// decodeJSON decodes the request body into v, writing a 400 and returning
// false on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return false
	}
	return true
}

// writeJSON writes v as a JSON response body with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
