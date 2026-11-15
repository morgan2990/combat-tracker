package mpmb

import (
	"strings"
	"testing"

	"combatapp/store"
)

const adultoSources = `({warlock:{shortname:"Warlock", level:1, name:"Warlock (Archfey Patron)", typeSp:"known", refType:"class", factor:[1, "warlock"]}, druid:{level:2, name:"Druid (Circle of Dreams)", typeSp:"list", refType:"class", factor:[1, "default"]}, 'magic initiate (druid)':{name:"Magic Initiate (Druid) [Origin] (feat)", typeSp:"feat", refType:"feat"}, 'ring of jaedear':{name:"Ring of Jaedear (Level 1) (item)", typeSp:"item", refType:"item"}})`

// adultoSpells is a sheet shaped like Puck Adulto's spell pages: a Warlock
// list, a Druid list, a feat list and an item list, with captions, headers
// and level dividers between them.
func adultoSpells() *Document {
	const p = "P7.SSfront."
	d := docOf(
		"Template.extras.SSfront", ",P7.SSfront.",
		"CurrentSpells.Stringified", adultoSources,
		"SpellSlots.CheckboxesSet.lvl1", "4", "SpellSlots.CheckboxesSet.lvl2", "0",
		p+"spellshead.class.0", "warlock", p+"spellshead.class.1", "druid",
		p+"spellshead.class.2", "magic initiate (druid)", p+"spellshead.class.3", "ring of jaedear",
		p+"spellshead.prepare.0", "0", p+"spellshead.prepare.1", "5",
		p+"spellshead.dc.0", "14", p+"spellshead.dc.1", "14",
		p+"spellshead.attack.0", "+6", p+"spellshead.attack.1", "+6",
		p+"spellshead.ability.0", "Cha Mod", p+"spellshead.ability.1", "Wis Mod", p+"spellshead.ability.3", "nothing",
	)
	rows := []struct{ remember, check, name string }{
		{"setcaptions##", "", "SPELL"},
		{"eldritch blast####warlock", "", "Faeric Arrow"},
		{"mage hand####warlock", "", "Mage Hand"},
		{"setdivider##1##1", "hide", ""},
		{"setcaptions##", "", "SPELL"},
		{"find familiar##atwill##warlock", "atwill", "Find Familiar ®"},
		{"hex####warlock", "", "Hex*"},
		{"setheader##druid##1", "hide", ""},
		{"hidethisline", "hide", ""},
		{"setdivider##0##2", "hide", ""},
		{"setcaptions##kn", "KN", "SPELL"},
		{"druidcraft##atwill##druid", "atwill", "Druidcraft"},
		{"setdivider##1##3", "hide", ""},
		{"setcaptions##pr", "PR", "SPELL"},
		{"absorb elements##checkbox##druid", "checkbox", "Absorb Elements"},
		{"find familiar##checkbox##druid", "markedbox", "Find Familiar ®"},
		{"healing word##checkbox##druid", "checkedbox", "Healing Word*"},
		{"setheader##magic initiate (druid)##2", "hide", ""},
		{"setdivider##0##4", "hide", ""},
		{"guidance##atwill##magic initiate (druid)", "atwill", "Guidance"},
		{"setdivider##1##5", "hide", ""},
		{"cure wounds##oncelr##magic initiate (druid)", "oncelr", "Cure Wounds"},
		{"setheader##ring of jaedear##3", "hide", ""},
		{"setdivider##1##6", "hide", ""},
		{"shield##oncelr##ring of jaedear", "oncelr", "Shield"},
	}
	for i, r := range rows {
		idx := itoa(i)
		d.with(p+"spells.remember."+idx, r.remember)
		if r.check != "" {
			d.with(p+"spells.check."+idx, r.check)
		}
		if r.name != "" {
			d.with(p+"spells.name."+idx, r.name, p+"spells.book."+idx, "P4", p+"spells.page."+idx, "272")
		}
	}
	return d
}

func listNamed(sc *store.SheetSpellcasting, name string) *store.SpellList {
	for i := range sc.SpellLists {
		if sc.SpellLists[i].Name == name {
			return &sc.SpellLists[i]
		}
	}
	return nil
}

func spellNamed(l *store.SpellList, name string) *store.Spell {
	for i := range l.Spells {
		if strings.HasPrefix(l.Spells[i].Name, name) {
			return &l.Spells[i]
		}
	}
	return nil
}

func TestSpellcasting_ListsCarryTheirSourceKind(t *testing.T) {
	m := newMapper(adultoSpells())
	sc := m.spellcasting()

	var got []string
	for _, l := range sc.SpellLists {
		got = append(got, l.Kind+":"+l.Name)
	}
	want := "class:Warlock (Archfey Patron),class:Druid (Circle of Dreams),feat:Magic Initiate (Druid) [Origin],item:Ring of Jaedear (Level 1)"
	if strings.Join(got, ",") != want {
		t.Errorf("lists = %v\nwant     %v", got, want)
	}
	if len(m.warns) != 0 {
		t.Errorf("unexpected warnings: %v", m.warns)
	}
}

func TestSpellcasting_ReadsOnlyTheFirstOfSeveralJoinedLiterals(t *testing.T) {
	// The real field is "({sources})##########({excl:[], incl:[...]})", and
	// writes undefined as (void 0).
	joined := strings.TrimSuffix(adultoSources, "})") + `, other:{calcSpellScores:(void 0)}})##########({excl:[], incl:["warlock", "druid"]})`
	d := adultoSpells().with("CurrentSpells.Stringified", joined)
	d.byName["CurrentSpells.Stringified"] = len(d.Fields) - 1
	m := newMapper(d)
	sc := m.spellcasting()

	if len(m.warns) != 0 {
		t.Fatalf("warnings = %v, want the sources to parse", m.warns)
	}
	if sc.PactMagic == nil || listNamed(sc, "Ring of Jaedear (Level 1)") == nil {
		t.Errorf("pact=%+v lists=%+v", sc.PactMagic, sc.SpellLists)
	}
}

func TestSpellcasting_FirstSectionLevelComesFromItsLabel(t *testing.T) {
	// A class with no cantrips (a Ranger) has no divider rows at all.
	d := docOf(
		"Template.extras.SSfront", ",P6.SSfront.",
		"CurrentSpells.Stringified", `({ranger:{shortname:"Ranger", level:3, name:"Beast Master", refType:"class", factor:[2, "default"]}})`,
		"P6.SSfront.spellsdiv.Text.0", "1st Level",
		"P6.SSfront.spellshead.class.0", "ranger",
		"P6.SSfront.spells.remember.0", "setcaptions##pr",
		"P6.SSfront.spells.remember.1", "absorb elements##checkbox##ranger", "P6.SSfront.spells.name.1", "Absorb Elements", "P6.SSfront.spells.check.1", "checkbox",
		"P6.SSfront.spells.remember.2", "hunter's mark##checkbox##ranger", "P6.SSfront.spells.name.2", "Hunter's Mark*", "P6.SSfront.spells.check.2", "markedbox",
	)
	sc := newMapper(d).spellcasting()

	l := sc.SpellLists[0]
	if l.Name != "Ranger (Beast Master)" {
		t.Errorf("list name = %q, want the class kept with its subclass", l.Name)
	}
	for _, s := range l.Spells {
		if s.Level != 1 {
			t.Errorf("%s has level %d, want 1", s.Name, s.Level)
		}
	}
}

func TestLevelFromLabel(t *testing.T) {
	for label, want := range map[string]int{"Cantrips (0 Level)": 0, "1st Level": 1, "2nd Level": 2, "3rd Level": 3, "9th Level": 9} {
		if got, ok := levelFromLabel(label); !ok || got != want {
			t.Errorf("levelFromLabel(%q) = %d, %v, want %d", label, got, ok, want)
		}
	}
	if _, ok := levelFromLabel("Spells"); ok {
		t.Error("an unrelated label was read as a level")
	}
}

func TestSpellcasting_ListHeaderDetails(t *testing.T) {
	sc := newMapper(adultoSpells()).spellcasting()

	druid := listNamed(sc, "Druid (Circle of Dreams)")
	if druid.Ability != "Wis" || druid.SaveDC != "14" || druid.Attack != "+6" || druid.PrepareCount == nil || *druid.PrepareCount != 5 {
		t.Errorf("druid header = %+v", druid)
	}
	warlock := listNamed(sc, "Warlock (Archfey Patron)")
	if warlock.Ability != "Cha" || warlock.PrepareCount != nil {
		t.Errorf("warlock: ability=%q prepare=%v (a prepare count of 0 means none)", warlock.Ability, warlock.PrepareCount)
	}
	if item := listNamed(sc, "Ring of Jaedear (Level 1)"); item.Ability != "" {
		t.Errorf("an ability of \"nothing\" must map to none, got %q", item.Ability)
	}
}

func TestSpellcasting_LevelComesFromDividersAndCaptionsAreSkipped(t *testing.T) {
	sc := newMapper(adultoSpells()).spellcasting()

	warlock := listNamed(sc, "Warlock (Archfey Patron)")
	for name, level := range map[string]int{"Faeric Arrow": 0, "Mage Hand": 0, "Find Familiar": 1, "Hex": 1} {
		if s := spellNamed(warlock, name); s == nil || s.Level != level {
			t.Errorf("warlock %s = %+v, want level %d", name, s, level)
		}
	}
	druid := listNamed(sc, "Druid (Circle of Dreams)")
	if s := spellNamed(druid, "Druidcraft"); s.Level != 0 {
		t.Errorf("druid cantrip level = %d", s.Level)
	}
	if s := spellNamed(druid, "Healing Word"); s.Level != 1 {
		t.Errorf("druid level-1 spell level = %d", s.Level)
	}
	for _, l := range sc.SpellLists {
		for _, s := range l.Spells {
			if s.Name == "SPELL" {
				t.Errorf("a caption row became a spell in %s", l.Name)
			}
		}
	}
	if len(warlock.Spells) != 4 {
		t.Errorf("warlock has %d spells, want 4", len(warlock.Spells))
	}
}

func TestSpellcasting_PreparationFromCheckValue(t *testing.T) {
	sc := newMapper(adultoSpells()).spellcasting()
	druid := listNamed(sc, "Druid (Circle of Dreams)")
	warlock := listNamed(sc, "Warlock (Archfey Patron)")
	mi := listNamed(sc, "Magic Initiate (Druid) [Origin]")

	for _, c := range []struct {
		list *store.SpellList
		name string
		want string
	}{
		{druid, "Find Familiar", "always"},
		{druid, "Healing Word", "prepared"},
		{druid, "Absorb Elements", "unprepared"},
		{warlock, "Hex", "known"},
		{warlock, "Find Familiar", "at_will"},
		{mi, "Cure Wounds", "limited"},
	} {
		if s := spellNamed(c.list, c.name); s == nil || s.Preparation != c.want {
			t.Errorf("%s in %s: %+v, want %s", c.name, c.list.Name, s, c.want)
		}
	}
}

func TestSpellcasting_KeepsDisplayedNameAndIdentifier(t *testing.T) {
	sc := newMapper(adultoSpells()).spellcasting()
	s := spellNamed(listNamed(sc, "Warlock (Archfey Patron)"), "Faeric")

	if s.Name != "Faeric Arrow" || s.Key != "eldritch blast" {
		t.Errorf("name=%q key=%q, want the displayed name and the identifier", s.Name, s.Key)
	}
	if s.Book != "P4" || s.Page != "272" {
		t.Errorf("book/page = %q/%q", s.Book, s.Page)
	}
}

func TestSpellcasting_UnknownMarkerBecomesOtherWithAWarning(t *testing.T) {
	d := adultoSpells().with("P7.SSfront.spells.check.14", "circle")
	m := newMapper(d)
	sc := m.spellcasting()

	s := spellNamed(listNamed(sc, "Druid (Circle of Dreams)"), "Absorb")
	if s.Preparation != "other" || s.Mark != "circle" {
		t.Errorf("spell = %+v", s)
	}
	if len(m.warns) != 1 || !strings.Contains(m.warns[0], "circle") {
		t.Errorf("warnings = %v", m.warns)
	}
}

func TestSpellcasting_PactMagicIsSeparatedFromCombinedSlots(t *testing.T) {
	m := newMapper(adultoSpells()) // Warlock 1 + Druid 2: the sheet shows 4 first-level slots
	sc := m.spellcasting()

	if sc.PactMagic == nil || sc.PactMagic.Total != 1 || sc.PactMagic.SlotLevel != 1 {
		t.Fatalf("pact magic = %+v, want 1 slot of level 1", sc.PactMagic)
	}
	if len(sc.Slots) != 1 || sc.Slots[0] != 3 {
		t.Errorf("slots = %v, want [3] (4 combined minus 1 pact)", sc.Slots)
	}
}

func TestSpellcasting_PactMagicUsesTheWarlockTable(t *testing.T) {
	for level, want := range map[int][2]int{1: {1, 1}, 2: {2, 1}, 3: {2, 2}, 5: {2, 3}, 9: {2, 5}, 11: {3, 5}, 17: {4, 5}, 20: {4, 5}} {
		src := `({warlock:{level:` + itoa(level) + `, name:"Warlock", refType:"class", factor:[1, "warlock"]}})`
		d := docOf("CurrentSpells.Stringified", src, "SpellSlots.CheckboxesSet.lvl"+itoa(want[1]), "5")
		sc := newMapper(d).spellcasting()
		if sc.PactMagic == nil || sc.PactMagic.Total != want[0] || sc.PactMagic.SlotLevel != want[1] {
			t.Errorf("Warlock %d: pact = %+v, want %d slots of level %d", level, sc.PactMagic, want[0], want[1])
			continue
		}
		if got := sc.Slots[want[1]-1]; got != 5-want[0] {
			t.Errorf("Warlock %d: remaining slots of level %d = %d, want %d", level, want[1], got, 5-want[0])
		}
	}
}

func TestSpellcasting_PactMagicThatDoesNotFitStaysCombined(t *testing.T) {
	d := adultoSpells().with("SpellSlots.CheckboxesSet.lvl1", "0")
	d.byName["SpellSlots.CheckboxesSet.lvl1"] = len(d.Fields) - 1 // the later value wins
	m := newMapper(d)
	sc := m.spellcasting()

	if sc.PactMagic != nil {
		t.Errorf("pact magic = %+v, want none when it does not fit", sc.PactMagic)
	}
	if len(m.warns) == 0 || !strings.Contains(m.warns[0], "Pact Magic") {
		t.Errorf("warnings = %v, want a Pact Magic warning", m.warns)
	}
}

func TestSpellcasting_UnreadableSourceDetailsDefaultToClassLists(t *testing.T) {
	d := adultoSpells().with("CurrentSpells.Stringified", "({atkAdd:(function (f) { return 1 })})")
	d.byName["CurrentSpells.Stringified"] = len(d.Fields) - 1
	m := newMapper(d)
	sc := m.spellcasting()

	for _, l := range sc.SpellLists {
		if l.Kind != "class" {
			t.Errorf("list %q has kind %q, want class", l.Name, l.Kind)
		}
	}
	if listNamed(sc, "Magic Initiate (Druid)") == nil {
		t.Errorf("lists = %+v, want names derived from the list keys", sc.SpellLists)
	}
	if sc.PactMagic != nil || len(sc.Slots) != 1 || sc.Slots[0] != 4 {
		t.Errorf("pact=%+v slots=%v, want slots left combined at [4]", sc.PactMagic, sc.Slots)
	}
	if len(m.warns) == 0 {
		t.Error("no warning for unreadable source details")
	}
}

func TestSpellcasting_RowsContinueOnTheNextPage(t *testing.T) {
	d := docOf(
		"Template.extras.SSfront", ",P7.SSfront.",
		"Template.extras.SSmore", ",P8.SSmore.",
		"CurrentSpells.Stringified", `({druid:{level:5, name:"Druid", refType:"class", factor:[1, "default"]}})`,
		"P7.SSfront.spellshead.class.0", "druid",
		"P7.SSfront.spells.remember.0", "setdivider##1##1",
		"P7.SSfront.spells.remember.1", "bless####druid", "P7.SSfront.spells.name.1", "Bless",
		"P8.SSmore.spells.remember.0", "cure wounds####druid", "P8.SSmore.spells.name.0", "Cure Wounds",
		"P8.SSmore.spells.remember.1", "setdivider##2##2",
		"P8.SSmore.spells.remember.2", "hold person####druid", "P8.SSmore.spells.name.2", "Hold Person",
	)
	sc := newMapper(d).spellcasting()

	spells := sc.SpellLists[0].Spells
	if len(spells) != 3 || spells[0].Level != 1 || spells[1].Name != "Cure Wounds" || spells[1].Level != 1 || spells[2].Level != 2 {
		t.Errorf("spells = %+v, want the level to carry onto the second page", spells)
	}
}

func TestSpellcasting_NoneWhenTheSheetHasNoSpells(t *testing.T) {
	if sc := newMapper(docOf("PC Name", "Fighter")).spellcasting(); sc != nil {
		t.Errorf("spellcasting = %+v, want nil", sc)
	}
}

func TestPreparation_Table(t *testing.T) {
	for check, want := range map[string]string{
		"markedbox": "always", "checkedbox": "prepared", "checkbox": "unprepared", "": "known",
		"atwill": "at_will", "oncelr": "limited", "weird": "other",
	} {
		if got, _ := preparation(check); got != want {
			t.Errorf("preparation(%q) = %q, want %q", check, got, want)
		}
	}
}
