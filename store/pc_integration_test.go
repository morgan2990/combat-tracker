//go:build integration

package store

import "testing"

func TestPCStore_CreateAndGetByID(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	if pc.ID == "" || pc.Type != "pc" {
		t.Fatalf("CreatePC() = %+v, want a non-empty ID and Type=pc", pc)
	}

	got, err := Global.GetPCByID(pc.ID)
	if err != nil {
		t.Fatalf("GetPCByID() error = %v", err)
	}
	if got == nil || got.Name != "Aragorn" || got.MaxHP != 45 {
		t.Errorf("GetPCByID() = %+v, want Name=Aragorn MaxHP=45", got)
	}
}

func TestPCStore_GetPCByID_NotFound(t *testing.T) {
	got, err := Global.GetPCByID("does-not-exist")
	if err != nil {
		t.Fatalf("GetPCByID() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetPCByID() = %+v, want nil", got)
	}
}

func TestPCStore_UpdatePC(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	err = Global.UpdatePC(pc.ID, PCUpdateFields{
		Name:             "Strider",
		MaxHP:            50,
		SharesInitiative: true,
		Currency:         Currency{GP: 10},
	})
	if err != nil {
		t.Fatalf("UpdatePC() error = %v", err)
	}

	got, err := Global.GetPCByID(pc.ID)
	if err != nil {
		t.Fatalf("GetPCByID() error = %v", err)
	}
	if got.Name != "Strider" || got.MaxHP != 50 || !got.SharesInitiative || got.Currency.GP != 10 {
		t.Errorf("after UpdatePC(), got = %+v", got)
	}
}

func TestPCStore_UpdatePCInventory(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	items := []Item{{Name: "Longsword", Quantity: 1}}
	currency := Currency{GP: 25}
	updated, err := Global.UpdatePCInventory(pc.ID, items, currency)
	if err != nil {
		t.Fatalf("UpdatePCInventory() error = %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].Name != "Longsword" || updated.Currency.GP != 25 {
		t.Errorf("UpdatePCInventory() = %+v, want Items=[Longsword] Currency.GP=25", updated)
	}
	// Name/MaxHP should be untouched.
	if updated.Name != "Aragorn" || updated.MaxHP != 45 {
		t.Errorf("UpdatePCInventory() changed Name/MaxHP: %+v", updated)
	}
}

func TestPCStore_UpdatePCInventory_NotFound(t *testing.T) {
	got, err := Global.UpdatePCInventory("does-not-exist", nil, Currency{})
	if err != nil {
		t.Fatalf("UpdatePCInventory() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("UpdatePCInventory() on missing PC = %+v, want nil", got)
	}
}

func TestPCStore_DeletePC_CascadesToCompanions(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	companion, err := Global.CreateCompanion(pc.ID, "owner1", "Brego", 20, true)
	if err != nil {
		t.Fatalf("CreateCompanion() error = %v", err)
	}

	if err := Global.DeletePC(pc.ID); err != nil {
		t.Fatalf("DeletePC() error = %v", err)
	}

	if got, _ := Global.GetPCByID(pc.ID); got != nil {
		t.Error("PC should have been deleted")
	}
	if got, _ := Global.GetPCByID(companion.ID); got != nil {
		t.Error("companion should have been cascade-deleted with its parent")
	}
}

func TestPCStore_GetCompanionsByParentID(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	if _, err := Global.CreateCompanion(pc.ID, "owner1", "Brego", 20, false); err != nil {
		t.Fatalf("CreateCompanion() error = %v", err)
	}

	companions, err := Global.GetCompanionsByParentID(pc.ID)
	if err != nil {
		t.Fatalf("GetCompanionsByParentID() error = %v", err)
	}
	if len(companions) != 1 || companions[0].Name != "Brego" {
		t.Errorf("GetCompanionsByParentID() = %+v, want [Brego]", companions)
	}
}

func TestPCStore_OwnsAnyPC(t *testing.T) {
	pc, err := Global.CreatePC("owner1", "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	owns, err := Global.OwnsAnyPC("owner1", []string{pc.ID, "some-other-id"})
	if err != nil {
		t.Fatalf("OwnsAnyPC() error = %v", err)
	}
	if !owns {
		t.Error("OwnsAnyPC() = false, want true (owner1 owns pc.ID)")
	}

	owns, err = Global.OwnsAnyPC("someone-else", []string{pc.ID})
	if err != nil {
		t.Fatalf("OwnsAnyPC() error = %v", err)
	}
	if owns {
		t.Error("OwnsAnyPC() = true, want false (someone-else doesn't own pc.ID)")
	}
}

func TestPCStore_ListPCsByOwner(t *testing.T) {
	if _, err := Global.CreatePC("owner-list-test", "Aragorn", 45); err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	if _, err := Global.CreatePC("owner-list-test", "Legolas", 30); err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	if _, err := Global.CreatePC("someone-else", "Gimli", 40); err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	pcs, err := Global.ListPCsByOwner("owner-list-test")
	if err != nil {
		t.Fatalf("ListPCsByOwner() error = %v", err)
	}
	if len(pcs) != 2 {
		t.Errorf("ListPCsByOwner() returned %d PCs, want 2", len(pcs))
	}
	for _, pc := range pcs {
		if pc.OwnerUserID != "owner-list-test" {
			t.Errorf("ListPCsByOwner() included a PC owned by %q", pc.OwnerUserID)
		}
	}
}
