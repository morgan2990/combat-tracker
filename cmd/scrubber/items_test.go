package main

import (
	"errors"
	"testing"

	"combatapp/store"
)

func withFakeItemStore(t *testing.T, fake *fakeItemUpserter) {
	t.Helper()
	original := itemDB
	itemDB = fake
	t.Cleanup(func() { itemDB = original })
}

func TestItemEdition(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{"XPHB", "5.5e"},
		{"XDMG", "5.5e"},
		{"XMM", "5.5e"},
		{"PHB", "5e"},
		{"DMG", "5e"},
		{"XGE", "5e"}, // X-prefixed but not a 5.5e core book — see design.md Decision 7
		{"", "5e"},
	}
	for _, c := range cases {
		if got := itemEdition(c.source); got != c.want {
			t.Errorf("itemEdition(%q) = %q, want %q", c.source, got, c.want)
		}
	}
}

func TestStripTypeSourceSuffix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"AF|DMG", "AF"},
		{"AF", "AF"},
		{"", ""},
		{"AF|DMG|EXTRA", "AF"},
	}
	for _, c := range cases {
		if got := stripTypeSourceSuffix(c.in); got != c.want {
			t.Errorf("stripTypeSourceSuffix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildCategoryLookup(t *testing.T) {
	entries := []fiveEToolsItemTypeEntry{
		{Name: "Ammunition", Abbreviation: "A"},
		{Name: "Armor", Abbreviation: "LA"},
		{Name: "No Abbreviation", Abbreviation: ""},
	}
	lookup := buildCategoryLookup(entries)

	if got := lookup["A"]; got != "Ammunition" {
		t.Errorf(`lookup["A"] = %q, want "Ammunition"`, got)
	}
	if got := lookup["LA"]; got != "Armor" {
		t.Errorf(`lookup["LA"] = %q, want "Armor"`, got)
	}
	if _, ok := lookup[""]; ok {
		t.Error("entries with an empty abbreviation should not be added to the lookup")
	}
	if len(lookup) != 2 {
		t.Errorf("len(lookup) = %d, want 2", len(lookup))
	}
}

func TestNormalizeRarity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"none", ""},
		{"rare", "rare"},
		{"", ""},
		{"very rare", "very rare"},
	}
	for _, c := range cases {
		if got := normalizeRarity(c.in); got != c.want {
			t.Errorf("normalizeRarity(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsUnreservedURIComponent(t *testing.T) {
	unreserved := "ABCXYZabcxyz0129-_.!~*'()"
	for i := 0; i < len(unreserved); i++ {
		c := unreserved[i]
		if !isUnreservedURIComponent(c) {
			t.Errorf("isUnreservedURIComponent(%q) = false, want true", c)
		}
	}

	reserved := " /?#[]@:&=+$,%\"<>{}|\\^`"
	for i := 0; i < len(reserved); i++ {
		c := reserved[i]
		if isUnreservedURIComponent(c) {
			t.Errorf("isUnreservedURIComponent(%q) = true, want false", c)
		}
	}
}

func TestEncodeURIComponent(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"abc", "abc"},
		{"a b", "a%20b"},
		{"100%", "100%25"},
		{"a+b", "a%2Bb"},
		{"", ""},
	}
	for _, c := range cases {
		if got := encodeURIComponent(c.in); got != c.want {
			t.Errorf("encodeURIComponent(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToURLHashPart(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Bag of Holding", "bag%20of%20holding"},
		{"PHB", "phb"},
		{"Boots of Elvenkind", "boots%20of%20elvenkind"},
	}
	for _, c := range cases {
		if got := toURLHashPart(c.in); got != c.want {
			t.Errorf("toURLHashPart(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestItemReferenceURL(t *testing.T) {
	got := itemReferenceURL("Bag of Holding", "DMG")
	want := "https://5e.tools/items.html#bag%20of%20holding_dmg"
	if got != want {
		t.Errorf("itemReferenceURL() = %q, want %q", got, want)
	}
}

func TestScrubItemEntries_BaseItemDirect(t *testing.T) {
	fake := newFakeItemUpserter()
	withFakeItemStore(t, fake)

	itemTypes := []fiveEToolsItemTypeEntry{{Name: "Armor", Abbreviation: "LA"}}
	baseItems := []fiveEToolsItemEntry{
		{Name: "Leather Armor", Source: "PHB", Type: "LA", Rarity: "none"},
	}

	processed, inserted, updated := scrubItemEntries(baseItems, itemTypes, nil)

	if processed != 1 || inserted != 1 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (1, 1, 0)", processed, inserted, updated)
	}
	got := fake.calls[0]
	if got.Name != "Leather Armor" || got.Edition != "5e" || got.Category != "Armor" || got.Rarity != "" {
		t.Errorf("got item = %+v, want Name=Leather Armor Edition=5e Category=Armor Rarity=\"\"", got)
	}
	if got.FiveEToolsID != "leather-armor-phb" || got.SourceBook != "PHB" {
		t.Errorf("got item = %+v", got)
	}
}

func TestScrubItemEntries_5DotFiveEdition(t *testing.T) {
	fake := newFakeItemUpserter()
	withFakeItemStore(t, fake)

	items := []fiveEToolsItemEntry{
		{Name: "Potion of Healing", Source: "XPHB", Type: "P"},
	}

	scrubItemEntries(nil, nil, items)

	if fake.calls[0].Edition != "5.5e" {
		t.Errorf("Edition = %q, want 5.5e", fake.calls[0].Edition)
	}
}

func TestScrubItemEntries_CopyInheritsCategoryAndRarity(t *testing.T) {
	fake := newFakeItemUpserter()
	withFakeItemStore(t, fake)

	itemTypes := []fiveEToolsItemTypeEntry{{Name: "Wondrous Item", Abbreviation: "W"}}
	items := []fiveEToolsItemEntry{
		{Name: "Bag of Holding", Source: "DMG", Type: "W", Rarity: "uncommon"},
		{Name: "Bag of Holding Variant", Source: "DMG", Copy: &fiveEToolsCopy{Name: "Bag of Holding", Source: "DMG"}},
	}

	processed, inserted, updated := scrubItemEntries(nil, itemTypes, items)

	if processed != 2 || inserted != 2 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (2, 2, 0)", processed, inserted, updated)
	}
	variant := fake.calls[1]
	if variant.Category != "Wondrous Item" || variant.Rarity != "uncommon" {
		t.Errorf("variant = %+v, want Category=Wondrous Item Rarity=uncommon (both inherited)", variant)
	}
}

func TestScrubItemEntries_CopyOverridesRarity(t *testing.T) {
	fake := newFakeItemUpserter()
	withFakeItemStore(t, fake)

	items := []fiveEToolsItemEntry{
		{Name: "Bag of Holding", Source: "DMG", Rarity: "uncommon"},
		{Name: "Bag of Holding+", Source: "DMG", Rarity: "rare", Copy: &fiveEToolsCopy{Name: "Bag of Holding", Source: "DMG"}},
	}

	scrubItemEntries(nil, nil, items)

	variant := fake.calls[1]
	if variant.Rarity != "rare" {
		t.Errorf("variant.Rarity = %q, want rare (own value, not inherited)", variant.Rarity)
	}
}

func TestScrubItemEntries_UnresolvableCopySkipped(t *testing.T) {
	fake := newFakeItemUpserter()
	withFakeItemStore(t, fake)

	items := []fiveEToolsItemEntry{
		{Name: "Orphan", Source: "DMG", Copy: &fiveEToolsCopy{Name: "Does Not Exist", Source: "DMG"}},
	}

	processed, inserted, updated := scrubItemEntries(nil, nil, items)

	if processed != 0 || inserted != 0 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want all 0", processed, inserted, updated)
	}
}

func TestScrubItemEntries_UpsertErrorSkipsButContinues(t *testing.T) {
	fake := newFakeItemUpserter()
	fake.forceErr = errors.New("boom")
	withFakeItemStore(t, fake)

	baseItems := []fiveEToolsItemEntry{{Name: "Longsword", Source: "PHB"}}

	processed, inserted, updated := scrubItemEntries(baseItems, nil, nil)

	if processed != 0 || inserted != 0 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want all 0 on upsert error", processed, inserted, updated)
	}
}

func TestScrubItemEntries_ReportsUpdated(t *testing.T) {
	fake := newFakeItemUpserter()
	fake.byKey["Longsword|5e"] = store.CatalogItem{Name: "Longsword", Edition: "5e"}
	withFakeItemStore(t, fake)

	baseItems := []fiveEToolsItemEntry{{Name: "Longsword", Source: "PHB"}}

	processed, inserted, updated := scrubItemEntries(baseItems, nil, nil)

	if processed != 1 || inserted != 0 || updated != 1 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (1, 0, 1)", processed, inserted, updated)
	}
}

func TestItemFiveEToolsID(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{"Bag of Holding", "DMG", "bag-of-holding-dmg"},
		{"Longsword", "PHB", "longsword-phb"},
		{"Potion of Healing", "XPHB", "potion-of-healing-xphb"},
	}
	for _, c := range cases {
		if got := itemFiveEToolsID(c.name, c.source); got != c.want {
			t.Errorf("itemFiveEToolsID(%q, %q) = %q, want %q", c.name, c.source, got, c.want)
		}
	}
}
