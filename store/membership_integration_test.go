//go:build integration

package store

import (
	"testing"
	"time"
)

func TestMembershipStore_Upsert_CreatesAndUpdates(t *testing.T) {
	userID, roomID := "member-user-1", "member-room-1"

	if err := GlobalMemberships.Upsert(userID, roomID, "pc1"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	memberships, err := GlobalMemberships.ListByUser(userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(memberships) != 1 || memberships[0].LastPCID != "pc1" {
		t.Fatalf("ListByUser() = %+v, want one membership with LastPCID=pc1", memberships)
	}

	// Upserting the same (user, room) pair again should update in place, not
	// insert a second document.
	if err := GlobalMemberships.Upsert(userID, roomID, "pc2"); err != nil {
		t.Fatalf("Upsert() (second call) error = %v", err)
	}
	memberships, err = GlobalMemberships.ListByUser(userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(memberships) != 1 || memberships[0].LastPCID != "pc2" {
		t.Errorf("ListByUser() after re-Upsert() = %+v, want exactly one membership with LastPCID=pc2", memberships)
	}
}

func TestMembershipStore_ListByUser_OrderedByRecency(t *testing.T) {
	userID := "member-user-recency"

	if err := GlobalMemberships.Upsert(userID, "member-room-older", "pc1"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := GlobalMemberships.Upsert(userID, "member-room-newer", "pc2"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	memberships, err := GlobalMemberships.ListByUser(userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(memberships) != 2 {
		t.Fatalf("ListByUser() returned %d memberships, want 2", len(memberships))
	}
	if memberships[0].RoomID != "member-room-newer" || memberships[1].RoomID != "member-room-older" {
		t.Errorf("ListByUser() = %+v, want [member-room-newer, member-room-older] (most recent first)", memberships)
	}
}

func TestMembershipStore_ListByUser_Empty(t *testing.T) {
	memberships, err := GlobalMemberships.ListByUser("no-such-user")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(memberships) != 0 {
		t.Errorf("ListByUser() = %+v, want empty", memberships)
	}
}

func TestMembershipStore_DeleteByRoom(t *testing.T) {
	roomID := "member-room-delete-by-room"
	if err := GlobalMemberships.Upsert("member-user-a", roomID, "pc1"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := GlobalMemberships.Upsert("member-user-b", roomID, "pc2"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := GlobalMemberships.Upsert("member-user-a", "member-room-untouched", "pc3"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if err := GlobalMemberships.DeleteByRoom(roomID); err != nil {
		t.Fatalf("DeleteByRoom() error = %v", err)
	}

	for _, userID := range []string{"member-user-a", "member-user-b"} {
		memberships, err := GlobalMemberships.ListByUser(userID)
		if err != nil {
			t.Fatalf("ListByUser(%s) error = %v", userID, err)
		}
		for _, m := range memberships {
			if m.RoomID == roomID {
				t.Errorf("ListByUser(%s) still includes deleted room %s", userID, roomID)
			}
		}
	}

	memberships, err := GlobalMemberships.ListByUser("member-user-a")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	found := false
	for _, m := range memberships {
		if m.RoomID == "member-room-untouched" {
			found = true
		}
	}
	if !found {
		t.Error("DeleteByRoom() unexpectedly removed a membership in a different room")
	}
}

func TestMembershipStore_DeleteOne(t *testing.T) {
	userID, roomID := "member-user-delete-one", "member-room-delete-one"
	if err := GlobalMemberships.Upsert(userID, roomID, "pc1"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := GlobalMemberships.Upsert(userID, "member-room-other", "pc2"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if err := GlobalMemberships.DeleteOne(userID, roomID); err != nil {
		t.Fatalf("DeleteOne() error = %v", err)
	}

	memberships, err := GlobalMemberships.ListByUser(userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(memberships) != 1 || memberships[0].RoomID != "member-room-other" {
		t.Errorf("ListByUser() after DeleteOne() = %+v, want exactly [member-room-other]", memberships)
	}

	// Deleting an already-deleted (or never-existing) membership is a no-op, not an error.
	if err := GlobalMemberships.DeleteOne(userID, roomID); err != nil {
		t.Errorf("DeleteOne() on missing membership error = %v, want nil", err)
	}
}
