package main

import "testing"

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
