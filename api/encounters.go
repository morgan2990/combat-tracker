package api

import (
	"encoding/json"
	"net/http"

	"combatapp/auth"
	"combatapp/store"
)

func validEncounterEdition(edition string) bool {
	return edition == "5e" || edition == "5.5e"
}

func CreateEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var e store.Encounter
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if e.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if !validEncounterEdition(e.Edition) {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	e.OwnerID = userID
	saved, err := store.GlobalEncounters.CreateEncounter(e)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func GetEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	e, err := store.GlobalEncounters.GetEncounterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if e == nil {
		http.NotFound(w, r)
		return
	}
	if e.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e)
}

func UpdateEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.GlobalEncounters.GetEncounterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body store.Encounter
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if !validEncounterEdition(body.Edition) {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	body.OwnerID = existing.OwnerID
	saved, err := store.GlobalEncounters.UpdateEncounter(id, body)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func DeleteEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.GlobalEncounters.GetEncounterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := store.GlobalEncounters.DeleteEncounter(id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListMyEncounters(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	edition := r.URL.Query().Get("edition")
	encounters, err := store.GlobalEncounters.ListEncountersByOwner(userID, edition)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if encounters == nil {
		encounters = []store.Encounter{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(encounters)
}
