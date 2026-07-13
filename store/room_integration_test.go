//go:build integration

package store

import "testing"

func intPtr(n int) *int {
	return &n
}

func strPtr(s string) *string {
	return &s
}

func TestRoomStore_SaveAndGetSnapshot(t *testing.T) {
	snap := RoomSnapshot{
		RoomID:             "room-save-get",
		OwnerUserID:        "owner1",
		IsCombatActive:     true,
		CurrentRound:       2,
		ActiveTurnEntityID: strPtr("entity1"),
		Edition:            "2024",
		PartyID:            "party1",
		Entities: []RoomEntitySnapshot{
			{ID: "entity1", Name: "Goblin", Type: "monster", MaxHP: 10, CurrentHP: 10, Initiative: intPtr(15)},
		},
	}

	if err := GlobalRooms.SaveRoomSnapshot(snap); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}

	got, err := GlobalRooms.GetRoomSnapshot("room-save-get")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetRoomSnapshot() = nil, want the snapshot just saved")
	}
	if got.OwnerUserID != "owner1" || got.CurrentRound != 2 || !got.IsCombatActive {
		t.Errorf("GetRoomSnapshot() = %+v, want OwnerUserID=owner1 CurrentRound=2 IsCombatActive=true", got)
	}
	if len(got.Entities) != 1 || got.Entities[0].Name != "Goblin" {
		t.Errorf("GetRoomSnapshot() Entities = %+v, want one entity named Goblin", got.Entities)
	}
}

func TestRoomStore_GetRoomSnapshot_NotFound(t *testing.T) {
	got, err := GlobalRooms.GetRoomSnapshot("does-not-exist")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetRoomSnapshot() = %+v, want nil", got)
	}
}

func TestRoomStore_SaveRoomSnapshot_UpsertReplaces(t *testing.T) {
	roomID := "room-upsert-replace"
	first := RoomSnapshot{RoomID: roomID, OwnerUserID: "owner1", CurrentRound: 1, Edition: "2014"}
	if err := GlobalRooms.SaveRoomSnapshot(first); err != nil {
		t.Fatalf("SaveRoomSnapshot() (first) error = %v", err)
	}

	second := RoomSnapshot{RoomID: roomID, OwnerUserID: "owner1", CurrentRound: 5, Edition: "2024", IsCombatActive: true}
	if err := GlobalRooms.SaveRoomSnapshot(second); err != nil {
		t.Fatalf("SaveRoomSnapshot() (second) error = %v", err)
	}

	got, err := GlobalRooms.GetRoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if got == nil || got.CurrentRound != 5 || got.Edition != "2024" || !got.IsCombatActive {
		t.Errorf("GetRoomSnapshot() after replace = %+v, want the second snapshot's fields", got)
	}
}

func TestRoomStore_Delete(t *testing.T) {
	roomID := "room-delete"
	if err := GlobalRooms.SaveRoomSnapshot(RoomSnapshot{RoomID: roomID, OwnerUserID: "owner1"}); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}

	if err := GlobalRooms.Delete(roomID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err := GlobalRooms.GetRoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if got != nil {
		t.Errorf("GetRoomSnapshot() after Delete() = %+v, want nil", got)
	}

	// Deleting an already-deleted (or never-existing) room is a no-op, not an error.
	if err := GlobalRooms.Delete(roomID); err != nil {
		t.Errorf("Delete() on missing room error = %v, want nil", err)
	}
}

func TestRoomStore_ListByOwner(t *testing.T) {
	owner := "owner-list-rooms"
	if err := GlobalRooms.SaveRoomSnapshot(RoomSnapshot{RoomID: "room-list-1", OwnerUserID: owner, Edition: "2014", IsCombatActive: true}); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}
	if err := GlobalRooms.SaveRoomSnapshot(RoomSnapshot{RoomID: "room-list-2", OwnerUserID: owner, Edition: "2024"}); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}
	if err := GlobalRooms.SaveRoomSnapshot(RoomSnapshot{RoomID: "room-list-other", OwnerUserID: "someone-else"}); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}

	results, err := GlobalRooms.ListByOwner(owner)
	if err != nil {
		t.Fatalf("ListByOwner() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ListByOwner() returned %d rooms, want 2", len(results))
	}
	byID := map[string]RoomSummary{}
	for _, r := range results {
		byID[r.RoomID] = r
	}
	if r, ok := byID["room-list-1"]; !ok || r.Edition != "2014" || !r.IsCombatActive {
		t.Errorf("ListByOwner() room-list-1 = %+v, want Edition=2014 IsCombatActive=true", r)
	}
	if r, ok := byID["room-list-2"]; !ok || r.Edition != "2024" || r.IsCombatActive {
		t.Errorf("ListByOwner() room-list-2 = %+v, want Edition=2024 IsCombatActive=false", r)
	}
	if _, ok := byID["room-list-other"]; ok {
		t.Error("ListByOwner() unexpectedly included a room owned by someone-else")
	}
}
