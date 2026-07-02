package api

import (
	"encoding/json"
	"net/http"

	"combatapp/auth"
	"combatapp/store"
)

type partyPayload struct {
	Name        string         `json:"name"`
	MemberPCIDs []string       `json:"member_pc_ids"`
	Currency    store.Currency `json:"currency"`
}

func CreateParty(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ResolveUserID(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body partyPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	p, err := store.GlobalParties.CreateParty(body.Name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func GetParty(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ResolveUserID(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	p, err := store.GlobalParties.GetPartyByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// UpdateParty saves membership and pooled-currency changes. A requester may
// edit a party if it currently has no members (they're the first to join it)
// or if they own at least one PC already in member_pc_ids — any member's
// owner can add/remove members or adjust the pooled currency, mirroring how
// a real table's shared purse works.
func UpdateParty(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.GlobalParties.GetPartyByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	var body partyPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Currency.IsNegative() {
		http.Error(w, "currency values cannot be negative", http.StatusBadRequest)
		return
	}
	if len(existing.MemberPCIDs) > 0 {
		ownsMember, err := store.Global.OwnsAnyPC(userID, existing.MemberPCIDs)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if !ownsMember {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	saved, err := store.GlobalParties.UpdateParty(id, body.MemberPCIDs, body.Currency)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}
