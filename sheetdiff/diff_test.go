package sheetdiff

import (
	"testing"

	"combatapp/store"
)

func ip(n int) *int { return &n }

func find(changes []Change, section, path string) *Change {
	for i := range changes {
		if changes[i].Section == section && changes[i].Path == path {
			return &changes[i]
		}
	}
	return nil
}

func TestSheets_NoDifferences(t *testing.T) {
	s := store.CharacterSheet{Combat: &store.SheetCombat{AC: ip(13)}, Attacks: []store.Attack{{Name: "Club", ToHit: "+1"}}}
	in := Input{Name: "Puck", MaxHP: 21, Sheet: s}

	if got := Sheets(in, in); len(got) != 0 {
		t.Errorf("changes = %+v, want none", got)
	}
}

func TestSheets_AnEmptyAndAnAbsentSectionAreEqual(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Defenses: &store.SheetDefenses{}, Attacks: []store.Attack{}}}
	if got := Sheets(old, Input{}); len(got) != 0 {
		t.Errorf("changes = %+v, want none", got)
	}
}

func TestSheets_ChangedValueShowsOldAndNew(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Combat: &store.SheetCombat{AC: ip(13), SpeedText: "30 ft"}}}
	new := Input{Sheet: store.CharacterSheet{Combat: &store.SheetCombat{AC: ip(15), SpeedText: "30 ft"}}}

	got := Sheets(old, new)
	if len(got) != 1 {
		t.Fatalf("changes = %+v", got)
	}
	c := got[0]
	if c.Section != "combat" || c.Path != "ac" || c.Kind != Changed || c.Old != float64(13) || c.New != float64(15) {
		t.Errorf("change = %+v", c)
	}
}

func TestSheets_NameAndMaxHPAreCompared(t *testing.T) {
	old := Input{Name: "Aria", MaxHP: 16}
	new := Input{Name: "Aria Sunsong", MaxHP: 20}

	got := Sheets(old, new)
	if c := find(got, "identity", "name"); c == nil || c.Old != "Aria" || c.New != "Aria Sunsong" {
		t.Errorf("name change = %+v", c)
	}
	if c := find(got, "combat", "max_hp"); c == nil || c.Old != float64(16) || c.New != float64(20) {
		t.Errorf("max_hp change = %+v", c)
	}
}

func TestSheets_AValueTheImportLacksIsShownAsRemoved(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Notes: &store.SheetNotes{Personality: "Curious", Flaws: "Reckless"}}}
	new := Input{Sheet: store.CharacterSheet{Notes: &store.SheetNotes{Personality: "Curious"}}}

	got := Sheets(old, new)
	if c := find(got, "notes", "flaws"); len(got) != 1 || c == nil || c.Kind != Removed || c.Old != "Reckless" {
		t.Errorf("changes = %+v, want the hand-written flaw shown as removed", got)
	}
}

func TestSheets_ListRowsPairByTitleSoReorderingIsNotAChange(t *testing.T) {
	a := store.CharacterSheet{Attacks: []store.Attack{{Name: "Club", ToHit: "+1"}, {Name: "Sting", ToHit: "DC 12"}}}
	b := store.CharacterSheet{Attacks: []store.Attack{{Name: "Sting", ToHit: "DC 12"}, {Name: "Club", ToHit: "+1"}}}

	if got := Sheets(Input{Sheet: a}, Input{Sheet: b}); len(got) != 0 {
		t.Errorf("a reordered list produced %+v", got)
	}
}

func TestSheets_ListRowsAddedRemovedAndChangedByTitle(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Attacks: []store.Attack{{Name: "Club", ToHit: "+1"}, {Name: "Dagger"}}}}
	new := Input{Sheet: store.CharacterSheet{Attacks: []store.Attack{{Name: "Club", ToHit: "+3"}, {Name: "Sting"}}}}

	got := Sheets(old, new)
	if c := find(got, "attacks", "[Club].to_hit"); c == nil || c.Kind != Changed || c.Old != "+1" || c.New != "+3" {
		t.Errorf("club change = %+v", c)
	}
	if c := find(got, "attacks", "[Dagger]"); c == nil || c.Kind != Removed {
		t.Errorf("dagger = %+v, want removed", c)
	}
	if c := find(got, "attacks", "[Sting]"); c == nil || c.Kind != Added {
		t.Errorf("sting = %+v, want added", c)
	}
	if len(got) != 3 {
		t.Errorf("changes = %+v, want exactly 3", got)
	}
}

func TestSheets_AWholeListRemovedIsReportedRowByRow(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Attacks: []store.Attack{{Name: "Club"}, {Name: "Dagger"}}}}

	got := Sheets(old, Input{})
	if len(got) != 2 || find(got, "attacks", "[Club]") == nil || find(got, "attacks", "[Dagger]") == nil {
		t.Errorf("changes = %+v, want each attack named as removed", got)
	}
	for _, c := range got {
		if c.Kind != Removed {
			t.Errorf("change = %+v", c)
		}
	}
}

func TestSheets_DuplicateTitlesStayDistinct(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Attacks: []store.Attack{{Name: "Bite", ToHit: "+2"}, {Name: "Bite", ToHit: "+4"}}}}
	new := Input{Sheet: store.CharacterSheet{Attacks: []store.Attack{{Name: "Bite", ToHit: "+2"}, {Name: "Bite", ToHit: "+5"}}}}

	got := Sheets(old, new)
	if len(got) != 1 || got[0].Path != "[Bite#2].to_hit" || got[0].New != "+5" {
		t.Errorf("changes = %+v", got)
	}
}

func TestSheets_RowsPairByWeaponKeyAndClass(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{
		WeaponMasteries: []store.WeaponMastery{{Weapon: "Sling", Mastery: "Slow"}},
		Skills:          &store.SheetSkills{Entries: []store.SkillEntry{{Key: "Stealth", Bonus: ip(4)}}},
		Identity:        &store.SheetIdentity{Classes: []store.SheetClass{{Class: "Druid", Level: ip(2)}}},
	}}
	new := Input{Sheet: store.CharacterSheet{
		WeaponMasteries: []store.WeaponMastery{{Weapon: "Sling", Mastery: "Nick"}},
		Skills:          &store.SheetSkills{Entries: []store.SkillEntry{{Key: "Stealth", Bonus: ip(5)}}},
		Identity:        &store.SheetIdentity{Classes: []store.SheetClass{{Class: "Druid", Level: ip(3)}}},
	}}

	got := Sheets(old, new)
	for _, want := range []struct{ sec, path string }{
		{"weapon_masteries", "[Sling].mastery"}, {"skills", "entries[Stealth].bonus"}, {"identity", "classes[Druid].level"},
	} {
		if find(got, want.sec, want.path) == nil {
			t.Errorf("missing change %s %s in %+v", want.sec, want.path, got)
		}
	}
}

func TestSheets_PlainValueListsAreComparedWhole(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{Defenses: &store.SheetDefenses{Resistances: []string{"Fire"}}}}
	new := Input{Sheet: store.CharacterSheet{Defenses: &store.SheetDefenses{Resistances: []string{"Fire", "Cold"}}}}

	got := Sheets(old, new)
	if len(got) != 1 || got[0].Path != "resistances" || got[0].Kind != Changed {
		t.Errorf("changes = %+v", got)
	}
}

func TestSheets_NewCharacterIsAllAdditions(t *testing.T) {
	new := Input{Name: "Puck", MaxHP: 21, Sheet: store.CharacterSheet{
		Identity: &store.SheetIdentity{Species: "Eladrin"},
		Attacks:  []store.Attack{{Name: "Club"}},
	}}

	got := Sheets(Input{}, new)
	for _, c := range got {
		if c.Kind != Added {
			t.Errorf("change = %+v, want only additions", c)
		}
	}
	if find(got, "identity", "name") == nil || find(got, "identity", "species") == nil || find(got, "attacks", "[Club]") == nil || find(got, "combat", "max_hp") == nil {
		t.Errorf("changes = %+v", got)
	}
}

func TestSheets_SectionsAreReportedInSheetOrder(t *testing.T) {
	new := Input{Sheet: store.CharacterSheet{
		Notes:    &store.SheetNotes{Flaws: "x"},
		Identity: &store.SheetIdentity{Species: "y"},
		Attacks:  []store.Attack{{Name: "z"}},
	}}

	var order []string
	for _, c := range Sheets(Input{}, new) {
		if len(order) == 0 || order[len(order)-1] != c.Section {
			order = append(order, c.Section)
		}
	}
	want := []string{"identity", "attacks", "notes"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestSheets_SourceRevisionAndPCIDAreNotDifferences(t *testing.T) {
	old := Input{Sheet: store.CharacterSheet{PCID: "a", Revision: 3, Source: &store.SheetSource{Kind: "mpmb"}}}
	new := Input{Sheet: store.CharacterSheet{PCID: "b", Revision: 9}}

	if got := Sheets(old, new); len(got) != 0 {
		t.Errorf("changes = %+v", got)
	}
}

// --- companions and inventory ---

func TestCompanions_ListsRemovedAndAddedWithoutMatching(t *testing.T) {
	got := Companions(
		[]store.PC{{Name: "Rex"}, {Name: "Mia"}},
		[]store.ImportedCompanion{{Name: "Rex"}, {Name: "Beast of the Burrow"}},
	)
	if len(got.Removed) != 2 || got.Removed[0] != "Rex" || len(got.Added) != 2 || got.Added[0] != "Rex" {
		t.Errorf("companions = %+v, want every old one removed and every imported one added, even a same-named one", got)
	}
	empty := Companions(nil, nil)
	if empty.Removed == nil || empty.Added == nil {
		t.Error("empty lists must be non-nil so they encode as []")
	}
}

func TestInventory_KeptWhenThePDFHasNone(t *testing.T) {
	got := Inventory([]store.Item{{Name: "Rope", Quantity: 2}}, store.Currency{GP: 25}, store.ImportedInventory{Present: false})

	if got.Action != "keep" || got.Changed || got.OldItems != 1 || got.NewItems != 1 {
		t.Errorf("inventory = %+v", got)
	}
}

func TestInventory_ReplacedWhenThePDFHasOne(t *testing.T) {
	items := []store.Item{{Name: "Rope", Quantity: 2}}
	got := Inventory(items, store.Currency{GP: 25}, store.ImportedInventory{
		Present: true, Items: []store.Item{{Name: "Torch", Quantity: 3}, {Name: "Rope", Quantity: 1}}, Currency: store.Currency{GP: 25},
	})
	if got.Action != "replace" || !got.Changed || got.OldItems != 1 || got.NewItems != 2 {
		t.Errorf("inventory = %+v", got)
	}
}

func TestInventory_ReplaceThatChangesNothingIsNotChanged(t *testing.T) {
	items := []store.Item{{Name: "Rope", Quantity: 2}, {Name: "Torch", Quantity: 1}}
	got := Inventory(items, store.Currency{GP: 5}, store.ImportedInventory{
		Present: true, Items: []store.Item{{Name: "torch", Quantity: 1}, {Name: "Rope", Quantity: 2}}, Currency: store.Currency{GP: 5},
	})
	if got.Action != "replace" || got.Changed {
		t.Errorf("inventory = %+v, want an order- and case-insensitive match to be unchanged", got)
	}
}

func TestInventory_CoinsAloneCountAsAChange(t *testing.T) {
	got := Inventory(nil, store.Currency{}, store.ImportedInventory{Present: true, Currency: store.Currency{GP: 10}})
	if got.Action != "replace" || !got.Changed {
		t.Errorf("inventory = %+v", got)
	}
}
