//go:build integration

package store

import (
	"errors"
	"testing"
)

func TestCharacterSheetStore_GetByPCID_NotFound(t *testing.T) {
	got, err := GlobalCharacterSheets.GetByPCID("no-such-pc")
	if err != nil {
		t.Fatalf("GetByPCID() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetByPCID() = %+v, want nil", got)
	}
}

func TestCharacterSheetStore_FirstSaveCreatesAtRevisionOne(t *testing.T) {
	pcID := NewID(8)
	rev, err := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{Personality: "Curious"})
	if err != nil {
		t.Fatalf("SaveSection() error = %v", err)
	}
	if rev != 1 {
		t.Errorf("revision = %d, want 1", rev)
	}

	got, err := GlobalCharacterSheets.GetByPCID(pcID)
	if err != nil || got == nil {
		t.Fatalf("GetByPCID() = %+v, %v", got, err)
	}
	if got.Revision != 1 || got.SchemaVersion != SheetSchemaVersion || got.Notes == nil || got.Notes.Personality != "Curious" {
		t.Errorf("stored sheet = %+v", got)
	}
}

func TestCharacterSheetStore_SavingOneSectionLeavesOthersIntact(t *testing.T) {
	pcID := NewID(8)
	rev, err := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{Flaws: "Reckless"})
	if err != nil {
		t.Fatalf("first SaveSection() error = %v", err)
	}
	rev, err = GlobalCharacterSheets.SaveSection(pcID, "attacks", rev, []Attack{{Name: "Club", ToHit: "+1"}})
	if err != nil {
		t.Fatalf("second SaveSection() error = %v", err)
	}
	if rev != 2 {
		t.Errorf("revision = %d, want 2", rev)
	}

	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Notes == nil || got.Notes.Flaws != "Reckless" {
		t.Errorf("notes were not preserved: %+v", got.Notes)
	}
	if len(got.Attacks) != 1 || got.Attacks[0].Name != "Club" {
		t.Errorf("attacks = %+v", got.Attacks)
	}
}

func TestCharacterSheetStore_StaleRevisionConflicts(t *testing.T) {
	pcID := NewID(8)
	rev, _ := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{})
	if _, err := GlobalCharacterSheets.SaveSection(pcID, "attacks", rev, []Attack{{Name: "A"}}); err != nil {
		t.Fatalf("SaveSection() error = %v", err)
	}

	_, err := GlobalCharacterSheets.SaveSection(pcID, "attacks", rev, []Attack{{Name: "B"}})
	if !errors.Is(err, ErrSheetConflict) {
		t.Fatalf("SaveSection() with stale revision error = %v, want ErrSheetConflict", err)
	}
	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Attacks[0].Name != "A" {
		t.Errorf("stale save changed the sheet: %+v", got.Attacks)
	}
}

func TestCharacterSheetStore_ConcurrentFirstSaveConflicts(t *testing.T) {
	pcID := NewID(8)
	if _, err := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{}); err != nil {
		t.Fatalf("first SaveSection() error = %v", err)
	}
	_, err := GlobalCharacterSheets.SaveSection(pcID, "defenses", 0, &SheetDefenses{Vision: "Darkvision 60 ft"})
	if !errors.Is(err, ErrSheetConflict) {
		t.Fatalf("second revision-0 SaveSection() error = %v, want ErrSheetConflict", err)
	}
}

func TestCharacterSheetStore_RevisionOnMissingSheetConflicts(t *testing.T) {
	_, err := GlobalCharacterSheets.SaveSection(NewID(8), "notes", 3, &SheetNotes{})
	if !errors.Is(err, ErrSheetConflict) {
		t.Fatalf("SaveSection() on missing sheet error = %v, want ErrSheetConflict", err)
	}
}

func TestCharacterSheetStore_SavePreservesSource(t *testing.T) {
	pcID := NewID(8)
	rev, _ := GlobalCharacterSheets.SaveSection(pcID, "notes", 0, &SheetNotes{})
	if _, err := GlobalCharacterSheets.col.UpdateOne(t.Context(),
		map[string]any{"pc_id": pcID},
		map[string]any{"$set": map[string]any{"source": SheetSource{Kind: "mpmb", SheetVersion: "14.0.8"}}}); err != nil {
		t.Fatalf("seeding source: %v", err)
	}

	if _, err := GlobalCharacterSheets.SaveSection(pcID, "defenses", rev, &SheetDefenses{Vision: "Darkvision 60 ft"}); err != nil {
		t.Fatalf("SaveSection() error = %v", err)
	}

	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if got.Source == nil || got.Source.Kind != "mpmb" || got.Source.SheetVersion != "14.0.8" {
		t.Errorf("source was not preserved: %+v", got.Source)
	}
}

func TestCharacterSheetStore_RoundTripsFreeTextAndBlankValues(t *testing.T) {
	pcID := NewID(8)
	ac := 13
	if _, err := GlobalCharacterSheets.SaveSection(pcID, "limited_uses", 0, []LimitedUse{
		{Name: "Balm", Max: "2d6", Recovery: "long rest", Source: "Ring of Jaedear"},
	}); err != nil {
		t.Fatalf("SaveSection() error = %v", err)
	}
	got, _ := GlobalCharacterSheets.GetByPCID(pcID)
	if len(got.LimitedUses) != 1 || got.LimitedUses[0].Max != "2d6" || got.LimitedUses[0].Source != "Ring of Jaedear" {
		t.Errorf("limited uses = %+v", got.LimitedUses)
	}

	rev := got.Revision
	if _, err := GlobalCharacterSheets.SaveSection(pcID, "combat", rev, &SheetCombat{AC: &ac}); err != nil {
		t.Fatalf("SaveSection(combat) error = %v", err)
	}
	got, _ = GlobalCharacterSheets.GetByPCID(pcID)
	if got.Combat == nil || got.Combat.AC == nil || *got.Combat.AC != 13 || got.Combat.Initiative != nil {
		t.Errorf("combat = %+v, want AC=13 and a nil (blank) initiative", got.Combat)
	}
}

func TestCharacterSheetStore_GetByPCIDsAndDelete(t *testing.T) {
	a, b, c := NewID(8), NewID(8), NewID(8)
	for _, id := range []string{a, b, c} {
		if _, err := GlobalCharacterSheets.SaveSection(id, "notes", 0, &SheetNotes{}); err != nil {
			t.Fatalf("SaveSection(%s) error = %v", id, err)
		}
	}

	got, err := GlobalCharacterSheets.GetByPCIDs([]string{a, b, "missing"})
	if err != nil {
		t.Fatalf("GetByPCIDs() error = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("GetByPCIDs() returned %d sheets, want 2", len(got))
	}

	if err := GlobalCharacterSheets.DeleteByPCIDs([]string{a, b}); err != nil {
		t.Fatalf("DeleteByPCIDs() error = %v", err)
	}
	for _, id := range []string{a, b} {
		if sh, _ := GlobalCharacterSheets.GetByPCID(id); sh != nil {
			t.Errorf("sheet %s still exists after delete", id)
		}
	}
	if sh, _ := GlobalCharacterSheets.GetByPCID(c); sh == nil {
		t.Errorf("DeleteByPCIDs() removed an unrelated sheet")
	}

	if err := GlobalCharacterSheets.DeleteByPCIDs(nil); err != nil {
		t.Errorf("DeleteByPCIDs(nil) error = %v, want nil", err)
	}
}
