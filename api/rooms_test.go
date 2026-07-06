package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/room"
	"combatapp/store"
)

func withFakeRoomStore(t *testing.T, fake *fakeRoomStore) {
	t.Helper()
	original := roomDB
	roomDB = fake
	t.Cleanup(func() { roomDB = original })
}

func withFakeMembershipStore(t *testing.T, fake *fakeMembershipStore) {
	t.Helper()
	original := membershipDB
	membershipDB = fake
	t.Cleanup(func() { membershipDB = original })
}

// A "room not found" test is deliberately absent here: requireOwnedRoom and
// GetRoom both route through room.Global.GetOrRestoreRoom(roomID,
// &store.GlobalRooms), which falls back to a real Mongo read (panicking on
// the uninitialized collection) whenever the room isn't already in the
// in-memory registry — exactly the case a "not found" test needs. That
// fallback takes a concrete *store.RoomStore, not an interface, so this
// package's seam can't reach it; would need either a room-package refactor
// or its own testcontainers tier.

func TestGetRoom_Success(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID, nil)
	r.SetPathValue("room_id", roomID)

	GetRoom(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["room_id"] != roomID || resp["edition"] != "5e" {
		t.Errorf("response = %+v, want room_id=%s edition=5e", resp, roomID)
	}
}

func TestGetPartyInventory_NoLinkedParty(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/rooms/"+roomID+"/party-inventory", nil)
	r.SetPathValue("room_id", roomID)

	GetPartyInventory(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp partyInventoryResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.PartyLinked {
		t.Error("PartyLinked should be false when no party is claimed")
	}
}

func TestGetPartyInventory_ForbiddenForNonOwner(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	withFakeAuth(t, "someone-else")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/rooms/"+roomID+"/party-inventory", nil)
	r.SetPathValue("room_id", roomID)

	GetPartyInventory(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGetPartyInventory_AggregatesMembers(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	rm, _ := room.Global.GetRoom(roomID)
	rm.ClaimParty("party1")

	withFakePartyStore(t, newFakePartyStore(store.Party{
		ID: "party1", MemberPCIDs: []string{"pc1"}, Currency: store.Currency{GP: 100},
	}))
	withFakePCStore(t, newFakePCStore(store.PC{
		ID: "pc1", Name: "Frodo", Currency: store.Currency{GP: 5},
		Items: []store.Item{{Name: "Ring", Quantity: 1}},
	}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/rooms/"+roomID+"/party-inventory", nil)
	r.SetPathValue("room_id", roomID)

	GetPartyInventory(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp partyInventoryResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.PartyLinked || len(resp.Items) != 1 || resp.Items[0].Name != "Ring" {
		t.Errorf("response = %+v, want PartyLinked=true with Ring item", resp)
	}
	if resp.MembersCurrencyTotal.GP != 5 || resp.PartyCurrency.GP != 100 {
		t.Errorf("currencies = members=%+v party=%+v, want members.GP=5 party.GP=100", resp.MembersCurrencyTotal, resp.PartyCurrency)
	}
}

func TestUpdatePartyInventoryPC_ForbiddenWhenNotAMember(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	rm, _ := room.Global.GetRoom(roomID)
	rm.ClaimParty("party1")
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/rooms/"+roomID+"/party-inventory/pc2", []byte(`{"items":[],"currency":{}}`))
	r.SetPathValue("room_id", roomID)
	r.SetPathValue("pc_id", "pc2")

	UpdatePartyInventoryPC(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdatePartyInventoryPC_NegativeCurrencyRejected(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	rm, _ := room.Global.GetRoom(roomID)
	rm.ClaimParty("party1")
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/rooms/"+roomID+"/party-inventory/pc1", []byte(`{"items":[],"currency":{"gp":-5}}`))
	r.SetPathValue("room_id", roomID)
	r.SetPathValue("pc_id", "pc1")

	UpdatePartyInventoryPC(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdatePartyInventoryPC_Success(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	rm, _ := room.Global.GetRoom(roomID)
	rm.ClaimParty("party1")
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}))
	withFakePCStore(t, newFakePCStore(store.PC{ID: "pc1", Name: "Frodo"}))
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/rooms/"+roomID+"/party-inventory/pc1", []byte(`{"items":[{"name":"Sting","quantity":1}],"currency":{"gp":10}}`))
	r.SetPathValue("room_id", roomID)
	r.SetPathValue("pc_id", "pc1")

	UpdatePartyInventoryPC(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestDeleteRoom_ForbiddenForNonOwner(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	withFakeRoomStore(t, newFakeRoomStore())
	withFakeMembershipStore(t, newFakeMembershipStore())
	withFakeAuth(t, "someone-else")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/rooms/"+roomID, nil)
	r.SetPathValue("room_id", roomID)

	DeleteRoom(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestDeleteRoom_Success(t *testing.T) {
	roomID := room.Global.CreateRoom("5e", "owner1")
	fakeRooms := newFakeRoomStore()
	fakeMemberships := newFakeMembershipStore()
	withFakeRoomStore(t, fakeRooms)
	withFakeMembershipStore(t, fakeMemberships)
	withFakeAuth(t, "owner1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/rooms/"+roomID, nil)
	r.SetPathValue("room_id", roomID)

	DeleteRoom(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if len(fakeRooms.deleted) != 1 || fakeRooms.deleted[0] != roomID {
		t.Errorf("fakeRooms.deleted = %v, want [%s]", fakeRooms.deleted, roomID)
	}
	if len(fakeMemberships.deletedByRoom) != 1 || fakeMemberships.deletedByRoom[0] != roomID {
		t.Errorf("fakeMemberships.deletedByRoom = %v, want [%s]", fakeMemberships.deletedByRoom, roomID)
	}
	if _, ok := room.Global.GetRoom(roomID); ok {
		t.Error("room should have been removed from the in-memory registry")
	}
}

func TestLeaveRoom_Success(t *testing.T) {
	fake := newFakeMembershipStore()
	withFakeMembershipStore(t, fake)
	withFakeAuth(t, "user1")

	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPost, "/api/rooms/room1/leave", nil)
	r.SetPathValue("room_id", "room1")

	LeaveRoom(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if len(fake.deletedByOne) != 1 || fake.deletedByOne[0] != [2]string{"user1", "room1"} {
		t.Errorf("fake.deletedByOne = %v, want [[user1 room1]]", fake.deletedByOne)
	}
}
