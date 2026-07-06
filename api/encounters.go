package api

import (
	"net/http"

	"combatapp/store"
)

func CreateEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var e store.Encounter
	if !decodeJSON(w, r, &e) {
		return
	}
	if e.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if !requireValidEdition(w, e.Edition) {
		return
	}
	e.OwnerID = userID
	saved, err := encounterDB.CreateEncounter(e)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func GetEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	e, err := encounterDB.GetEncounterByID(id)
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
	writeJSON(w, http.StatusOK, e)
}

func UpdateEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	existing, err := encounterDB.GetEncounterByID(id)
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
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if !requireValidEdition(w, body.Edition) {
		return
	}
	body.OwnerID = existing.OwnerID
	saved, err := encounterDB.UpdateEncounter(id, body)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func DeleteEncounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	existing, err := encounterDB.GetEncounterByID(id)
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
	if err := encounterDB.DeleteEncounter(id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListMyEncounters(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	edition := r.URL.Query().Get("edition")
	encounters, err := encounterDB.ListEncountersByOwner(userID, edition)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if encounters == nil {
		encounters = []store.Encounter{}
	}
	writeJSON(w, http.StatusOK, encounters)
}
