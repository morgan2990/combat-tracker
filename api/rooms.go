package api

import (
	"encoding/json"
	"net/http"

	"combatapp/room"
	"combatapp/store"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
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
	writeJSON(w, http.StatusCreated, map[string]string{
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
	writeJSON(w, http.StatusOK, map[string]any{
		"room_id":          id,
		"edition":          edition,
		"is_combat_active": isCombatActive,
	})
}
