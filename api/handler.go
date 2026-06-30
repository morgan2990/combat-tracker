package api

import (
	"encoding/json"
	"net/http"

	"combatapp/room"
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
