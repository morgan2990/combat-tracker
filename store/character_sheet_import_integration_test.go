//go:build integration

package store

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func importedSheet(pcID string) CharacterSheet {
	return CharacterSheet{
		PCID:   pcID,
		Source: &SheetSource{Kind: "mpmb", SheetVersion: "14.0.8-beta", FileName: "puck.pdf"},
		Notes:  &SheetNotes{Personality: "Curious"},
	}
}

// --- ReplaceSheet ---

func TestCharacterSheetStore_ReplaceSheetCreatesAtRevisionOne(t *testing.T) {
	pcID := NewID(8)
	rev, err := GlobalCharacterSheets.ReplaceSheet(importedSheet(pcID), 0)
	if err != nil || rev != 1 {
		t.Fatalf("ReplaceSheet() = %d, %v", rev, err)
	}

	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Revision != 1 || got.Source == nil || got.Source.FileName != "puck.pdf" || got.Notes.Personality != "Curious" {
		t.Errorf("stored = %+v", got)
	}
}

func TestCharacterSheetStore_ReplaceSheetClearsSectionsTheImportLacks(t *testing.T) {
	pcID := NewID(8)
	rev, _ := GlobalCharacterSheets.SaveSection(pcID, "attacks", 0, []Attack{{Name: "Club"}})
	rev, _ = GlobalCharacterSheets.SaveSection(pcID, "defenses", rev, &SheetDefenses{Vision: "Darkvision"})

	newRev, err := GlobalCharacterSheets.ReplaceSheet(importedSheet(pcID), rev)
	if err != nil || newRev != rev+1 {
		t.Fatalf("ReplaceSheet() = %d, %v", newRev, err)
	}

	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if len(got.Attacks) != 0 || got.Defenses != nil {
		t.Errorf("old sections survived the replace: attacks=%+v defenses=%+v", got.Attacks, got.Defenses)
	}
	if got.Notes == nil || got.Source == nil {
		t.Errorf("imported sections missing: %+v", got)
	}
}

func TestCharacterSheetStore_ReplaceSheetConflicts(t *testing.T) {
	pcID := NewID(8)
	rev, _ := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{})

	if _, err := GlobalCharacterSheets.ReplaceSheet(importedSheet(pcID), rev+5); !errors.Is(err, ErrSheetConflict) {
		t.Errorf("stale base revision: error = %v, want ErrSheetConflict", err)
	}
	if _, err := GlobalCharacterSheets.ReplaceSheet(importedSheet(pcID), 0); !errors.Is(err, ErrSheetConflict) {
		t.Errorf("create over an existing sheet: error = %v, want ErrSheetConflict", err)
	}
	if _, err := GlobalCharacterSheets.ReplaceSheet(importedSheet(NewID(8)), 3); !errors.Is(err, ErrSheetConflict) {
		t.Errorf("replace of a missing sheet: error = %v, want ErrSheetConflict", err)
	}
	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Source != nil || got.Revision != rev {
		t.Errorf("a conflicting replace changed the sheet: %+v", got)
	}
}

func TestCharacterSheetStore_SectionSavesAfterAReplaceKeepSource(t *testing.T) {
	pcID := NewID(8)
	rev, _ := GlobalCharacterSheets.ReplaceSheet(importedSheet(pcID), 0)

	if _, err := GlobalCharacterSheets.SaveSection(pcID, "defenses", rev, &SheetDefenses{Vision: "Darkvision"}); err != nil {
		t.Fatalf("SaveSection() error = %v", err)
	}

	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Source == nil || got.Source.FileName != "puck.pdf" || got.Notes == nil {
		t.Errorf("sheet = %+v, want source and imported sections kept", got)
	}
}

// --- import store ---

func sampleImport(owner, pcID string) CharacterSheetImport {
	return CharacterSheetImport{
		OwnerUserID:  owner,
		PCID:         pcID,
		BaseRevision: 2,
		Source:       SheetSource{Kind: "mpmb", SheetVersion: "14.0.8-beta", FileSHA256: "abc"},
		Parsed: ImportParsed{
			Name: "Puck", MaxHP: 21, Warnings: []string{"a warning"},
			Companions: []ImportedCompanion{{Name: "Robin", MaxHP: 10, Creature: Creature{AC: "12+Dex"}}},
			Inventory:  ImportedInventory{Present: true, Items: []Item{{Name: "Rope", Quantity: 2}}},
		},
		RawFields: []RawField{
			{Name: "Attack.1.To Hit", Value: "+6", Type: "/Tx"},
			{Name: "Action 9", Value: "Tomá wacho!", TM: "Tomá wacho!#!#Kid Tricks (feat)"},
			{Name: "Size Category", Value: "1", Display: "Small", Type: "/Ch"},
		},
	}
}

func TestSheetImportStore_CreateAndGetRoundTrip(t *testing.T) {
	created, err := GlobalSheetImports.Create(sampleImport("u1", "pc1"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" || created.Status != ImportDraft || created.ExpiresAt == nil || created.ExpiresAt.Before(created.CreatedAt) {
		t.Errorf("created = %+v", created)
	}

	got, err := GlobalSheetImports.Get(created.ID)
	if err != nil || got == nil {
		t.Fatalf("Get() = %+v, %v", got, err)
	}
	if got.BaseRevision != 2 || got.Parsed.Name != "Puck" || len(got.Parsed.Companions) != 1 || got.Parsed.Companions[0].Creature.AC != "12+Dex" {
		t.Errorf("parsed data = %+v", got.Parsed)
	}
	if len(got.RawFields) != 3 || got.RawFields[0].Name != "Attack.1.To Hit" || got.RawFields[1].TM != "Tomá wacho!#!#Kid Tricks (feat)" || got.RawFields[2].Display != "Small" {
		t.Errorf("raw fields = %+v, want names with dots kept as stored, in order", got.RawFields)
	}
	if got.Source.FileSHA256 != "abc" || got.ExpiresAt == nil {
		t.Errorf("source=%+v expires=%v", got.Source, got.ExpiresAt)
	}
}

func TestSheetImportStore_GetMissing(t *testing.T) {
	got, err := GlobalSheetImports.Get("nope")
	if err != nil || got != nil {
		t.Errorf("Get() = %+v, %v, want nil, nil", got, err)
	}
}

func TestSheetImportStore_ClaimWorksOnceAndReleaseRestores(t *testing.T) {
	created, _ := GlobalSheetImports.Create(sampleImport("u1", "pc1"))

	first, err := GlobalSheetImports.Claim(created.ID)
	if err != nil || first == nil || first.Status != ImportApplying {
		t.Fatalf("first Claim() = %+v, %v", first, err)
	}
	if second, err := GlobalSheetImports.Claim(created.ID); err != nil || second != nil {
		t.Errorf("second Claim() = %+v, %v, want nil (already claimed)", second, err)
	}

	if err := GlobalSheetImports.Release(created.ID); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if again, _ := GlobalSheetImports.Claim(created.ID); again == nil {
		t.Error("a released import could not be claimed again")
	}
}

func TestSheetImportStore_MarkAppliedStopsExpiryAndRecordsThePC(t *testing.T) {
	created, _ := GlobalSheetImports.Create(sampleImport("u1", ""))
	GlobalSheetImports.Claim(created.ID)

	if err := GlobalSheetImports.MarkApplied(created.ID, "newpc"); err != nil {
		t.Fatalf("MarkApplied() error = %v", err)
	}

	got, _ := GlobalSheetImports.Get(created.ID)
	if got.Status != ImportApplied || got.PCID != "newpc" || got.AppliedAt == nil || got.ExpiresAt != nil {
		t.Errorf("applied import = %+v, want status applied, the PC recorded and no expiry", got)
	}
	if claimed, _ := GlobalSheetImports.Claim(created.ID); claimed != nil {
		t.Error("an applied import could be claimed")
	}
}

func TestSheetImportStore_DraftsExpireThroughATTLIndex(t *testing.T) {
	cur, err := GlobalSheetImports.col.Indexes().List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var indexes []struct {
		Key                bson.D `bson:"key"`
		ExpireAfterSeconds *int32 `bson:"expireAfterSeconds"`
	}
	if err := cur.All(t.Context(), &indexes); err != nil {
		t.Fatal(err)
	}
	for _, idx := range indexes {
		if len(idx.Key) == 1 && idx.Key[0].Key == "expires_at" {
			if idx.ExpireAfterSeconds == nil || *idx.ExpireAfterSeconds != 0 {
				t.Errorf("the expires_at index is not a TTL index: %+v", idx)
			}
			return
		}
	}
	t.Error("no index on expires_at")
}

func TestSheetImportStore_DeleteAndKeepOnlyTheLatestPerPC(t *testing.T) {
	pcID := NewID(8)
	a, _ := GlobalSheetImports.Create(sampleImport("u1", pcID))
	b, _ := GlobalSheetImports.Create(sampleImport("u1", pcID))
	other, _ := GlobalSheetImports.Create(sampleImport("u1", NewID(8)))

	if err := GlobalSheetImports.DeleteOthersForPC(pcID, b.ID); err != nil {
		t.Fatalf("DeleteOthersForPC() error = %v", err)
	}
	if got, _ := GlobalSheetImports.Get(a.ID); got != nil {
		t.Error("the older import was kept")
	}
	if got, _ := GlobalSheetImports.Get(b.ID); got == nil {
		t.Error("the kept import was deleted")
	}
	if got, _ := GlobalSheetImports.Get(other.ID); got == nil {
		t.Error("another PC's import was deleted")
	}

	if err := GlobalSheetImports.Delete(b.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got, _ := GlobalSheetImports.Get(b.ID); got != nil {
		t.Error("Delete() did not delete")
	}
	if err := GlobalSheetImports.DeleteByPCIDs(nil); err != nil {
		t.Errorf("DeleteByPCIDs(nil) error = %v", err)
	}
}
