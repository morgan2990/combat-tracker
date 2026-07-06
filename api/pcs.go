package api

import (
	"net/http"

	"combatapp/store"
)

type pcPayload struct {
	Name             string         `json:"name"`
	MaxHP            int            `json:"max_hp"`
	SharesInitiative bool           `json:"shares_initiative"`
	Items            []store.Item   `json:"items"`
	Currency         store.Currency `json:"currency"`
}

func CreatePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body pcPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	pc, err := pcDB.CreatePC(userID, body.Name, body.MaxHP)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, pc)
}

// requireOwnedPC fetches the PC or companion document by id and verifies
// the requesting user owns it, writing the appropriate error response and
// returning ok=false if it doesn't exist or isn't owned by userID.
func requireOwnedPC(w http.ResponseWriter, r *http.Request, id, userID string) (*store.PC, bool) {
	existing, err := pcDB.GetPCByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return nil, false
	}
	if existing == nil {
		http.NotFound(w, r)
		return nil, false
	}
	if existing.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return existing, true
}

func UpdatePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := requireOwnedPC(w, r, id, userID); !ok {
		return
	}
	var body pcPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	if body.Currency.IsNegative() {
		http.Error(w, "currency values cannot be negative", http.StatusBadRequest)
		return
	}
	fields := store.PCUpdateFields{
		Name:             body.Name,
		MaxHP:            body.MaxHP,
		SharesInitiative: body.SharesInitiative,
		Items:            body.Items,
		Currency:         body.Currency,
	}
	if err := pcDB.UpdatePC(id, fields); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeletePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := requireOwnedPC(w, r, id, userID); !ok {
		return
	}
	if err := pcDB.DeletePC(id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type pcResponse struct {
	PC         *store.PC  `json:"pc"`
	Companions []store.PC `json:"companions"`
}

func GetPC(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	pc, err := pcDB.GetPCByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pc == nil {
		http.NotFound(w, r)
		return
	}
	if pc.OwnerUserID != userID {
		http.NotFound(w, r)
		return
	}
	companions, err := pcDB.GetCompanionsByParentID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if companions == nil {
		companions = []store.PC{}
	}
	writeJSON(w, http.StatusOK, pcResponse{PC: pc, Companions: companions})
}

func CreateCompanion(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	parentID := r.PathValue("id")
	parent, err := pcDB.GetPCByID(parentID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if parent == nil || parent.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body pcPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	companion, err := pcDB.CreateCompanion(parentID, userID, body.Name, body.MaxHP, body.SharesInitiative)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, companion)
}
