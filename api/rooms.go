package api

import (
	"encoding/json"
	"net/http"

	"combatapp/auth"
	"combatapp/room"
	"combatapp/store"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Edition string `json:"edition"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint — empty body is fine
	edition := body.Edition
	if edition != "5e" && edition != "5.5e" {
		edition = "5e"
	}
	roomID := room.Global.CreateRoom(edition, userID)
	if rm, ok := room.Global.GetRoom(roomID); ok {
		go rm.PersistNow(&store.GlobalRooms)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"room_id": roomID,
		"edition": edition,
	})
}

func GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")
	if roomID == "" {
		http.Error(w, "room_id required", http.StatusBadRequest)
		return
	}
	rm, found := room.Global.GetOrRestoreRoom(roomID, &store.GlobalRooms)
	if !found {
		http.NotFound(w, r)
		return
	}
	id, edition, isCombatActive := rm.Summary()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"room_id":          id,
		"edition":          edition,
		"is_combat_active": isCombatActive,
	})
}
