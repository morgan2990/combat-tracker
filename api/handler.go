package api

import (
	"encoding/json"
	"net/http"

	"combatapp/room"
	"combatapp/store"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	roomID, dmToken := room.Global.CreateRoom()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"room_id":  roomID,
		"dm_token": dmToken,
	})
}

func UpsertEntity(w http.ResponseWriter, r *http.Request) {
	var p store.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if p.Name == "" || p.MaxHP <= 0 || (p.Type != "player" && p.Type != "companion") {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if p.Type == "companion" && p.ParentPCName == "" {
		http.Error(w, "companion requires parent_pc_name", http.StatusBadRequest)
		return
	}
	if err := store.Global.UpsertEntity(p); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

type profileResponse struct {
	Profile    *store.Profile  `json:"profile"`
	Companions []store.Profile `json:"companions"`
}

func GetEntity(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	profile, err := store.Global.GetEntityByName(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if profile == nil {
		http.NotFound(w, r)
		return
	}
	companions, err := store.Global.GetCompanionsByParent(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if companions == nil {
		companions = []store.Profile{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResponse{Profile: profile, Companions: companions})
}
