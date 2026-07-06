package api

import (
	"net/http"

	"combatapp/store"
)

type partyPayload struct {
	Name        string         `json:"name"`
	MemberPCIDs []string       `json:"member_pc_ids"`
	Currency    store.Currency `json:"currency"`
}

// partyView is the JSON shape returned to clients for a party: the stored
// document plus each member's resolved name. member_pc_ids may include PCs
// owned by other users (via join codes), which the client has no other way
// to name — without this, a party member's card falls back to rendering raw
// PC IDs for any character it doesn't own.
type partyView struct {
	*store.Party
	MemberNames map[string]string `json:"member_names"`
}

// newPartyView resolves p's member names, skipping a GetPCByID round trip for
// any member already present in knownNames (e.g. the requester's own PCs,
// which callers like Me already have loaded). knownNames may be nil.
func newPartyView(p *store.Party, knownNames map[string]string) (partyView, error) {
	names := make(map[string]string, len(p.MemberPCIDs))
	for _, pcID := range p.MemberPCIDs {
		if name, ok := knownNames[pcID]; ok {
			names[pcID] = name
			continue
		}
		pc, err := pcDB.GetPCByID(pcID)
		if err != nil {
			return partyView{}, err
		}
		if pc != nil {
			names[pcID] = pc.Name
		}
	}
	return partyView{Party: p, MemberNames: names}, nil
}

// writePartyView resolves p into its client-facing view and writes it as the
// JSON response, or writes a 500 on resolution failure. Returns false on
// failure so callers can short-circuit consistently with the rest of the file.
func writePartyView(w http.ResponseWriter, status int, p *store.Party, knownNames map[string]string) bool {
	view, err := newPartyView(p, knownNames)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return false
	}
	writeJSON(w, status, view)
	return true
}

func CreateParty(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(w, r); !ok {
		return
	}
	var body partyPayload
	if !decodeJSON(w, r, &body) {
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
	writePartyView(w, http.StatusOK, p, nil)
}

func GetParty(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(w, r); !ok {
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
	writePartyView(w, http.StatusOK, p, nil)
}

// UpdateParty saves membership and pooled-currency changes. A requester may
// edit a party if it currently has no members (they're the first to join it)
// or if they own at least one PC already in member_pc_ids — any member's
// owner can add/remove members or adjust the pooled currency, mirroring how
// a real table's shared purse works.
func UpdateParty(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
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
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Currency.IsNegative() {
		http.Error(w, "currency values cannot be negative", http.StatusBadRequest)
		return
	}
	if len(existing.MemberPCIDs) > 0 {
		ownsMember, err := pcDB.OwnsAnyPC(userID, existing.MemberPCIDs)
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
	writePartyView(w, http.StatusOK, saved, nil)
}

type joinPartyPayload struct {
	JoinCode string `json:"join_code"`
	PCID     string `json:"pc_id"`
}

// JoinParty lets any authenticated user add one of their own PCs to a party
// by presenting its join code — unlike UpdateParty, this requires no existing
// membership, only that the requester owns the PC being added.
func JoinParty(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body joinPartyPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.JoinCode == "" || body.PCID == "" {
		http.Error(w, "join_code and pc_id required", http.StatusBadRequest)
		return
	}
	party, err := store.GlobalParties.GetPartyByJoinCode(body.JoinCode)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if party == nil {
		http.NotFound(w, r)
		return
	}
	pc, err := pcDB.GetPCByID(body.PCID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pc == nil || pc.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	saved, err := store.GlobalParties.AddMember(party.ID, body.PCID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writePartyView(w, http.StatusOK, saved, map[string]string{pc.ID: pc.Name})
}
