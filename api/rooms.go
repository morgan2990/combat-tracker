package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"

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

type partyInventoryItemRow struct {
	OwnerPCID     string `json:"owner_pc_id"`
	OwnerName     string `json:"owner_name"`
	Name          string `json:"name"`
	Quantity      int    `json:"quantity"`
	CatalogItemID string `json:"catalog_item_id,omitempty"`
	ReferenceURL  string `json:"reference_url,omitempty"`
}

type partyInventoryMember struct {
	PCID     string         `json:"pc_id"`
	Name     string         `json:"name"`
	Currency store.Currency `json:"currency"`
}

type partyInventoryResponse struct {
	PartyLinked          bool                    `json:"party_linked"`
	Items                []partyInventoryItemRow `json:"items"`
	Members              []partyInventoryMember  `json:"members"`
	PartyCurrency        store.Currency          `json:"party_currency"`
	MembersCurrencyTotal store.Currency          `json:"members_currency_total"`
}

// GetPartyInventory returns the DM's aggregated view of the room's linked
// party: every member's items (tagged with owner), the pooled party
// currency, and the sum of members' personal currency.
func GetPartyInventory(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	rm, ok := requireOwnedRoom(w, r, r.PathValue("room_id"), userID)
	if !ok {
		return
	}

	party, err := resolveLinkedParty(rm)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if party == nil {
		writeJSON(w, http.StatusOK, partyInventoryResponse{Items: []partyInventoryItemRow{}, Members: []partyInventoryMember{}})
		return
	}

	items := []partyInventoryItemRow{}
	members := []partyInventoryMember{}
	var membersCurrency store.Currency
	for _, pcID := range party.MemberPCIDs {
		pc, err := store.Global.GetPCByID(pcID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if pc == nil {
			continue
		}
		for _, it := range pc.Items {
			items = append(items, partyInventoryItemRow{
				OwnerPCID:     pc.ID,
				OwnerName:     pc.Name,
				Name:          it.Name,
				Quantity:      it.Quantity,
				CatalogItemID: it.CatalogItemID,
				ReferenceURL:  it.ReferenceURL,
			})
		}
		members = append(members, partyInventoryMember{PCID: pc.ID, Name: pc.Name, Currency: pc.Currency})
		membersCurrency.PP += pc.Currency.PP
		membersCurrency.GP += pc.Currency.GP
		membersCurrency.EP += pc.Currency.EP
		membersCurrency.SP += pc.Currency.SP
		membersCurrency.CP += pc.Currency.CP
	}

	writeJSON(w, http.StatusOK, partyInventoryResponse{
		PartyLinked:          true,
		Items:                items,
		Members:              members,
		PartyCurrency:        party.Currency,
		MembersCurrencyTotal: membersCurrency,
	})
}

type partyInventoryEditPayload struct {
	Items    []store.Item   `json:"items"`
	Currency store.Currency `json:"currency"`
}

// UpdatePartyInventoryPC lets the room's DM overwrite one linked-party
// member's items and currency, without going through /api/pcs/:id's
// owner-only check.
func UpdatePartyInventoryPC(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	rm, ok := requireOwnedRoom(w, r, r.PathValue("room_id"), userID)
	if !ok {
		return
	}
	pcID := r.PathValue("pc_id")

	party, err := resolveLinkedParty(rm)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if party == nil || !slices.Contains(party.MemberPCIDs, pcID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body partyInventoryEditPayload
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Currency.IsNegative() {
		http.Error(w, "currency values cannot be negative", http.StatusBadRequest)
		return
	}

	updated, err := store.Global.UpdatePCInventory(pcID, body.Items, body.Currency)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if updated == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// requireOwnedRoom resolves roomID and verifies the requester owns it,
// writing the appropriate error response and returning ok=false on failure.
func requireOwnedRoom(w http.ResponseWriter, r *http.Request, roomID, userID string) (*room.Room, bool) {
	rm, found := room.Global.GetOrRestoreRoom(roomID, &store.GlobalRooms)
	if !found {
		http.NotFound(w, r)
		return nil, false
	}
	if rm.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return rm, true
}

// resolveLinkedParty returns rm's linked party, or nil if rm has no linked
// party (or that party no longer exists).
func resolveLinkedParty(rm *room.Room) (*store.Party, error) {
	partyID := rm.PartyID()
	if partyID == "" {
		return nil, nil
	}
	return store.GlobalParties.GetPartyByID(partyID)
}
