package api

import (
	"encoding/json"
	"net/http"

	"combatapp/auth"
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
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	pc, err := store.Global.CreatePC(userID, body.Name, body.MaxHP)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pc)
}

func UpdatePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.Global.GetPCByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
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
	if err := store.Global.UpdatePC(id, body.Name, body.MaxHP, body.Items, body.Currency); err != nil {
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
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	pc, err := store.Global.GetPCByID(id)
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
	companions, err := store.Global.GetCompanionsByParentID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if companions == nil {
		companions = []store.PC{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pcResponse{PC: pc, Companions: companions})
}

func CreateCompanion(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	parentID := r.PathValue("id")
	parent, err := store.Global.GetPCByID(parentID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if parent == nil || parent.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	companion, err := store.Global.CreateCompanion(parentID, userID, body.Name, body.MaxHP, body.SharesInitiative)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companion)
}
