package api

import (
	"encoding/json"
	"errors"
	"io"
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
	// An empty body is valid (edition defaults to "5e" below) and decodes
	// as io.EOF, not a real error — decodeJSON doesn't distinguish that
	// from malformed JSON, so it's handled here instead of reused.
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	edition := resolveEditionOrDefault(body.Edition)
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
