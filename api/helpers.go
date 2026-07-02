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

// isValidEdition reports whether s is a supported edition value.
func isValidEdition(s string) bool {
	return s == "5e" || s == "5.5e"
}

// requireValidEdition writes a 400 and returns false if edition isn't a
// supported value.
func requireValidEdition(w http.ResponseWriter, edition string) bool {
	if !isValidEdition(edition) {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return false
	}
	return true
}

// resolveEditionOrDefault returns edition if it's a supported value,
// otherwise "5e". Used by CreateRoom, which is intentionally lenient about
// the edition field (see openspec/specs/room-creation/spec.md) rather than
// rejecting an invalid/omitted value like every other endpoint does.
func resolveEditionOrDefault(edition string) string {
	if !isValidEdition(edition) {
		return "5e"
	}
	return edition
}
