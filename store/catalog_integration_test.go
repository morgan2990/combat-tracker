//go:build integration

package store

import "testing"

func TestMonsterStore_UpsertInsertsThenUpdates(t *testing.T) {
	m := Monster{Name: "Goblin", Edition: "5e", MaxHP: 7}
	result, outcome, err := GlobalMonsters.UpsertMonster(m)
	if err != nil {
		t.Fatalf("UpsertMonster() error = %v", err)
	}
	if outcome != UpsertInserted {
		t.Errorf("outcome = %v, want UpsertInserted", outcome)
	}
	if result.MaxHP != 7 {
		t.Errorf("MaxHP = %d, want 7", result.MaxHP)
	}

	m2 := Monster{Name: "Goblin", Edition: "5e", MaxHP: 9}
	result2, outcome2, err := GlobalMonsters.UpsertMonster(m2)
	if err != nil {
		t.Fatalf("UpsertMonster() (second call) error = %v", err)
	}
	if outcome2 != UpsertUpdated {
		t.Errorf("outcome = %v, want UpsertUpdated", outcome2)
	}
	if result2.MaxHP != 9 {
		t.Errorf("MaxHP = %d, want 9 (updated)", result2.MaxHP)
	}
	if result2.ID != result.ID {
		t.Error("upserting the same {name, edition} should replace the same document, not create a new one")
	}
}

func TestMonsterStore_GetMonsterByName(t *testing.T) {
	if _, _, err := GlobalMonsters.UpsertMonster(Monster{Name: "Orc", Edition: "5e", MaxHP: 15}); err != nil {
		t.Fatalf("UpsertMonster() error = %v", err)
	}

	got, err := GlobalMonsters.GetMonsterByName("Orc")
	if err != nil {
		t.Fatalf("GetMonsterByName() error = %v", err)
	}
	if got == nil || got.MaxHP != 15 {
		t.Errorf("GetMonsterByName() = %+v, want MaxHP=15", got)
	}

	if got, err := GlobalMonsters.GetMonsterByName("does-not-exist"); err != nil || got != nil {
		t.Errorf("GetMonsterByName(unknown) = (%+v, %v), want (nil, nil)", got, err)
	}
}

func TestMonsterStore_SearchMonsters_NoTypesenseReturnsEmpty(t *testing.T) {
	hits, err := GlobalMonsters.SearchMonsters("Goblin", "5e", "user1")
	if err != nil {
		t.Fatalf("SearchMonsters() error = %v, want nil (best-effort, no Typesense configured)", err)
	}
	if len(hits) != 0 {
		t.Errorf("SearchMonsters() = %v, want empty (no Typesense configured)", hits)
	}
}

func TestCatalogItemStore_UpsertInsertsThenUpdates(t *testing.T) {
	ci := CatalogItem{Name: "Longsword", Edition: "5e", Category: "Weapon"}
	result, outcome, err := GlobalCatalogItems.UpsertItem(ci)
	if err != nil {
		t.Fatalf("UpsertItem() error = %v", err)
	}
	if outcome != UpsertInserted {
		t.Errorf("outcome = %v, want UpsertInserted", outcome)
	}

	ci2 := CatalogItem{Name: "Longsword", Edition: "5e", Category: "Martial Weapon"}
	result2, outcome2, err := GlobalCatalogItems.UpsertItem(ci2)
	if err != nil {
		t.Fatalf("UpsertItem() (second call) error = %v", err)
	}
	if outcome2 != UpsertUpdated || result2.Category != "Martial Weapon" {
		t.Errorf("second upsert = (%+v, %v), want UpsertUpdated with Category=Martial Weapon", result2, outcome2)
	}
	if result2.ID != result.ID {
		t.Error("upserting the same {name, edition} should replace the same document")
	}
}

func TestCatalogItemStore_GetItemByID(t *testing.T) {
	created, _, err := GlobalCatalogItems.UpsertItem(CatalogItem{Name: "Shield", Edition: "5e", Category: "Armor"})
	if err != nil {
		t.Fatalf("UpsertItem() error = %v", err)
	}

	got, err := GlobalCatalogItems.GetItemByID(created.ID.Hex())
	if err != nil {
		t.Fatalf("GetItemByID() error = %v", err)
	}
	if got == nil || got.Name != "Shield" {
		t.Errorf("GetItemByID() = %+v, want Name=Shield", got)
	}
}

func TestCatalogItemStore_GetItemByID_InvalidHexReturnsNilNoError(t *testing.T) {
	got, err := GlobalCatalogItems.GetItemByID("not-a-valid-hex-id")
	if err != nil {
		t.Fatalf("GetItemByID() error = %v, want nil for a malformed id", err)
	}
	if got != nil {
		t.Errorf("GetItemByID() = %+v, want nil", got)
	}
}

func TestCustomMonsterStore_CreateGetUpdateDelete(t *testing.T) {
	m := CustomMonster{Name: "Bandit Chief", Edition: "5e", MaxHP: 25, OwnerID: "owner1"}
	created, err := GlobalCustomMonsters.CreateCustomMonster(m)
	if err != nil {
		t.Fatalf("CreateCustomMonster() error = %v", err)
	}
	if created.ID == "" || !created.IsCustom {
		t.Fatalf("created = %+v, want a generated ID and IsCustom=true", created)
	}

	got, err := GlobalCustomMonsters.GetCustomMonsterByID(created.ID)
	if err != nil {
		t.Fatalf("GetCustomMonsterByID() error = %v", err)
	}
	if got == nil || got.Name != "Bandit Chief" {
		t.Errorf("GetCustomMonsterByID() = %+v, want Name=Bandit Chief", got)
	}

	updated, err := GlobalCustomMonsters.UpdateCustomMonster(created.ID, CustomMonster{
		Name: "Bandit Captain", Edition: "5e", MaxHP: 30, OwnerID: "owner1",
	})
	if err != nil {
		t.Fatalf("UpdateCustomMonster() error = %v", err)
	}
	if updated.Name != "Bandit Captain" || updated.MaxHP != 30 {
		t.Errorf("updated = %+v, want Name=Bandit Captain MaxHP=30", updated)
	}

	if err := GlobalCustomMonsters.DeleteCustomMonster(created.ID); err != nil {
		t.Fatalf("DeleteCustomMonster() error = %v", err)
	}
	if got, _ := GlobalCustomMonsters.GetCustomMonsterByID(created.ID); got != nil {
		t.Error("custom monster should have been deleted")
	}
}

func TestCustomMonsterStore_ListCustomMonstersByOwner(t *testing.T) {
	if _, err := GlobalCustomMonsters.CreateCustomMonster(CustomMonster{Name: "A", Edition: "5e", OwnerID: "owner-list-test"}); err != nil {
		t.Fatalf("CreateCustomMonster() error = %v", err)
	}
	if _, err := GlobalCustomMonsters.CreateCustomMonster(CustomMonster{Name: "B", Edition: "5.5e", OwnerID: "owner-list-test"}); err != nil {
		t.Fatalf("CreateCustomMonster() error = %v", err)
	}
	if _, err := GlobalCustomMonsters.CreateCustomMonster(CustomMonster{Name: "C", Edition: "5e", OwnerID: "someone-else"}); err != nil {
		t.Fatalf("CreateCustomMonster() error = %v", err)
	}

	results, err := GlobalCustomMonsters.ListCustomMonstersByOwner("owner-list-test", "5e")
	if err != nil {
		t.Fatalf("ListCustomMonstersByOwner() error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "A" {
		t.Errorf("ListCustomMonstersByOwner(owner-list-test, 5e) = %+v, want exactly [A]", results)
	}
}

func TestEncounterStore_CreateGetUpdateDelete(t *testing.T) {
	e := Encounter{Name: "Goblin Ambush", Edition: "5e", OwnerID: "owner1"}
	created, err := GlobalEncounters.CreateEncounter(e)
	if err != nil {
		t.Fatalf("CreateEncounter() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created encounter should have a generated ID")
	}

	got, err := GlobalEncounters.GetEncounterByID(created.ID)
	if err != nil {
		t.Fatalf("GetEncounterByID() error = %v", err)
	}
	if got == nil || got.Name != "Goblin Ambush" {
		t.Errorf("GetEncounterByID() = %+v, want Name=Goblin Ambush", got)
	}

	updated, err := GlobalEncounters.UpdateEncounter(created.ID, Encounter{Name: "Bigger Ambush", Edition: "5e", OwnerID: "owner1"})
	if err != nil {
		t.Fatalf("UpdateEncounter() error = %v", err)
	}
	if updated.Name != "Bigger Ambush" {
		t.Errorf("updated.Name = %q, want Bigger Ambush", updated.Name)
	}

	if err := GlobalEncounters.DeleteEncounter(created.ID); err != nil {
		t.Fatalf("DeleteEncounter() error = %v", err)
	}
	if got, _ := GlobalEncounters.GetEncounterByID(created.ID); got != nil {
		t.Error("encounter should have been deleted")
	}
}

func TestEncounterStore_ListEncountersByOwner(t *testing.T) {
	if _, err := GlobalEncounters.CreateEncounter(Encounter{Name: "A", Edition: "5e", OwnerID: "owner-enc-list"}); err != nil {
		t.Fatalf("CreateEncounter() error = %v", err)
	}
	if _, err := GlobalEncounters.CreateEncounter(Encounter{Name: "B", Edition: "5.5e", OwnerID: "owner-enc-list"}); err != nil {
		t.Fatalf("CreateEncounter() error = %v", err)
	}

	results, err := GlobalEncounters.ListEncountersByOwner("owner-enc-list", "5e")
	if err != nil {
		t.Fatalf("ListEncountersByOwner() error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "A" {
		t.Errorf("ListEncountersByOwner(owner-enc-list, 5e) = %+v, want exactly [A]", results)
	}
}
