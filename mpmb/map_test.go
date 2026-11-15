package mpmb

import (
	"reflect"
	"strings"
	"testing"
)

func ptrInt(p *int) int {
	if p == nil {
		return -999
	}
	return *p
}

// --- identity ---

func TestMap_MulticlassComesFromTheClassFeaturesHeaders(t *testing.T) {
	res := Map(docOf(
		"Class and Levels", "Warlock (Archfey Patron) 1, Circle of Dreams Druid 2",
		"Class Features", "**Warlock (Archfey Patron), level 1:**\r#◆ Eldritch Invocations# (Warlock 1, P24 153)\r   Use the button.\r**Druid (Circle of Dreams), level 2:**\r#◆ Wild Shape# (Druid 2, P24 90)\r   I can shapechange.",
	))

	id := res.Sheet.Identity
	if id.ClassesText != "Warlock (Archfey Patron) 1, Circle of Dreams Druid 2" {
		t.Errorf("classes_text = %q", id.ClassesText)
	}
	if len(id.Classes) != 2 {
		t.Fatalf("classes = %+v", id.Classes)
	}
	for i, want := range []struct {
		class, sub string
		level      int
	}{{"Warlock", "Archfey Patron", 1}, {"Druid", "Circle of Dreams", 2}} {
		c := id.Classes[i]
		if c.Class != want.class || c.Subclass != want.sub || ptrInt(c.Level) != want.level {
			t.Errorf("class %d = %+v, want %+v", i, c, want)
		}
	}
}

func TestMap_ClassWithoutABaseNameTakesItFromItsFeatures(t *testing.T) {
	res := Map(docOf(
		"Class and Levels", "Beast Master",
		"Class Features", "**Beast Master, level 3:**\r#◆ Favored Enemy# (Ranger 1, P24 119) #[2× per LR or SS]#\r   I always have the spell.\r#◆ Ranger's Companion# (Beast Master 3, PHB 93)\r   Text.",
	))

	got := res.Sheet.Identity.Classes
	if len(got) != 1 || got[0].Class != "Ranger" || got[0].Subclass != "Beast Master" || ptrInt(got[0].Level) != 3 {
		t.Errorf("classes = %+v, want Ranger / Beast Master 3", got)
	}
}

func TestMap_AHeaderThatIsItsOwnBaseClassHasNoSubclass(t *testing.T) {
	res := Map(docOf("Class Features", "**Fighter, level 2:**\r#◆ Second Wind# (Fighter 1, P24 90)\r   Heal."))

	got := res.Sheet.Identity.Classes
	if len(got) != 1 || got[0].Class != "Fighter" || got[0].Subclass != "" {
		t.Errorf("classes = %+v", got)
	}
}

func TestMap_SizeIsTheSelectedLabelNotTheSharedValue(t *testing.T) {
	d := docOf("PC Name", "Puck").withDisplay("Size Category", "1", "Small")

	if got := Map(d).Sheet.Identity.Size; got != "Small" {
		t.Errorf("size = %q, want Small", got)
	}
}

func TestMap_IdentityBasics(t *testing.T) {
	res := Map(docOf(
		"PC Name", "Puck (Adulto)", "Player Name", "Rodrigo", "Character Level", "3",
		"Race", "Elf, Fey (Eladrin), Multiverse", "Background", "Guide", "Alignment", "Chaotic Good",
		"Total Experience", "2700", "HP Max", "21",
	))

	id := res.Sheet.Identity
	if res.Name != "Puck (Adulto)" || res.MaxHP != 21 {
		t.Errorf("name=%q maxHP=%d", res.Name, res.MaxHP)
	}
	if id.Player != "Rodrigo" || id.Species != "Elf, Fey (Eladrin), Multiverse" || id.Background != "Guide" ||
		id.Alignment != "Chaotic Good" || ptrInt(id.XP) != 2700 || ptrInt(id.CharacterLevel) != 3 {
		t.Errorf("identity = %+v", id)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

// --- abilities, skills, combat, defenses ---

func TestMap_AbilitiesWithSavesAsPrinted(t *testing.T) {
	res := Map(docOf(
		"Str", "8", "Str Mod", "-1", "Str ST Mod", "-1",
		"Wis", "18", "Wis Mod", "4", "Wis ST Mod", "6", "Wis ST Prof", "/True",
	))

	a := res.Sheet.Abilities
	if ptrInt(a.Str.Score) != 8 || ptrInt(a.Str.Mod) != -1 || a.Str.SaveProficient {
		t.Errorf("str = %+v", a.Str)
	}
	if ptrInt(a.Wis.Score) != 18 || ptrInt(a.Wis.SaveBonus) != 6 || !a.Wis.SaveProficient {
		t.Errorf("wis = %+v", a.Wis)
	}
	if a.Dex.Score != nil {
		t.Errorf("an absent ability must stay blank, got %v", *a.Dex.Score)
	}
}

func TestMap_SkillsWithProficiencyAndExpertise(t *testing.T) {
	res := Map(docOf("Ste", "5", "Ste Prof", "/True", "Ste Exp", "/True", "Acr", "1", "Passive Perception", "15"))

	sk := res.Sheet.Skills
	if len(sk.Entries) != 2 || ptrInt(sk.PassivePerception) != 15 {
		t.Fatalf("skills = %+v", sk)
	}
	byName := map[string]bool{}
	for _, e := range sk.Entries {
		byName[e.Key] = e.Proficient && e.Expertise
		if e.Key == "Stealth" && (e.Ability != "Dex" || ptrInt(e.Bonus) != 5) {
			t.Errorf("stealth = %+v", e)
		}
	}
	if !byName["Stealth"] || byName["Acrobatics"] {
		t.Errorf("proficiency flags = %v", byName)
	}
}

func TestMap_CombatSpeedsHitDiceAndInitiative(t *testing.T) {
	res := Map(docOf(
		"AC", "13", "Initiative bonus", "2", "Proficiency Bonus", "2",
		"Speed", "40 ft, climb 30 ft, burrow 15 ft",
		"HD1 Level", "3", "HD1 Die", "8",
	))

	c := res.Sheet.Combat
	if ptrInt(c.AC) != 13 || ptrInt(c.Initiative) != 2 || ptrInt(c.ProficiencyBonus) != 2 {
		t.Errorf("combat = %+v", c)
	}
	if c.SpeedText != "40 ft, climb 30 ft, burrow 15 ft" {
		t.Errorf("speed text = %q", c.SpeedText)
	}
	if ptrInt(c.Speeds.Walk) != 40 || ptrInt(c.Speeds.Climb) != 30 || ptrInt(c.Speeds.Burrow) != 15 || c.Speeds.Fly != nil {
		t.Errorf("speeds = %+v", c.Speeds)
	}
	if len(c.HitDice) != 1 || c.HitDice[0].Die != 8 || c.HitDice[0].Total != 3 {
		t.Errorf("hit dice = %+v", c.HitDice)
	}
}

func TestMap_MetricSheetsKeepSpeedTextOnly(t *testing.T) {
	res := Map(docOf("Unit System", "metric", "Speed", "9 m"))

	c := res.Sheet.Combat
	if c.SpeedText != "9 m" || c.Speeds.Walk != nil {
		t.Errorf("combat = %+v, want the text kept and no numeric speed", c)
	}
}

func TestMap_DefensesVisionResistancesAndSaveNotes(t *testing.T) {
	res := Map(docOf(
		"Vision", "Darkvision 60 ft", "Resistance Damage Type 1", "Fire", "Resistance Damage Type 2", " ", "Resistance Damage Type 3", "Cold",
		"Saving Throw advantages / disadvantages", "Magic can't put me to sleep; **Adv. vs.** charmed",
	))

	d := res.Sheet.Defenses
	if d.Vision != "Darkvision 60 ft" || !reflect.DeepEqual(d.Resistances, []string{"Fire", "Cold"}) {
		t.Errorf("defenses = %+v", d)
	}
	if strings.Contains(d.SaveNotes, "*") || !strings.Contains(d.SaveNotes, "Adv. vs. charmed") {
		t.Errorf("save notes = %q", d.SaveNotes)
	}
}

// --- attacks, actions, limited uses ---

func TestMap_AttacksUseTheSelectionAndSkipEmptyRows(t *testing.T) {
	d := docOf(
		"Attack.1.Weapon", "fey arrow (eldritch blast)", "Attack.1.Mod", "Cha Mod", "Attack.1.Range", "120 ft",
		"Attack.1.To Hit", "+6", "Attack.1.Damage", "1×1d10", "Attack.1.Damage Type", "Force", "Attack.1.Description", "Each d10 is a separate beam",
		"Attack.2.Mod", "empty", "Attack.2.Damage Type", " ",
		"Attack.3.Weapon", "club", "Attack.3.Mod", "Str Mod", "Attack.3.To Hit", "+1",
	).withDisplay("Attack.1.Weapon Selection", "fey arrow (eldritch blast)", "Fey Arrow (Eldritch Blast)")

	got := Map(d).Sheet.Attacks
	if len(got) != 2 {
		t.Fatalf("attacks = %+v, want 2 (the empty row is skipped)", got)
	}
	if a := got[0]; a.Name != "Fey Arrow (Eldritch Blast)" || a.Ability != "Cha" || a.ToHit != "+6" || a.Damage != "1×1d10" || a.DamageType != "Force" || a.Range != "120 ft" || a.Notes == "" {
		t.Errorf("attack 1 = %+v", a)
	}
	if got[1].Name != "club" || got[1].Ability != "Str" {
		t.Errorf("attack 3 = %+v", got[1])
	}
}

func TestMap_ActionSourceComesFromTheMappingName(t *testing.T) {
	d := docOf("Action 1", "Attack / Cast a Spell", "Bonus Action 1", "Fey Step", "Reaction 1", "Shield").
		withTM("Action 9", "Tomá wacho!", "Tomá wacho!#!#Kid Tricks (feat)")
	d.withTM("Action 1", "Attack / Cast a Spell", "Attack / Cast a Spell#!#Default action")

	a := Map(d).Sheet.Actions
	byName := map[string]struct {
		source  string
		isStd   bool
		present bool
	}{}
	for _, e := range a.Action {
		byName[e.Name] = struct {
			source  string
			isStd   bool
			present bool
		}{e.Source, e.IsDefault, true}
	}
	if e := byName["Tomá wacho!"]; !e.present || e.source != "Kid Tricks (feat)" || e.isStd {
		t.Errorf("Tomá wacho! = %+v", e)
	}
	if e := byName["Attack / Cast a Spell"]; !e.present || e.source != "" || !e.isStd {
		t.Errorf("default action = %+v, want is_default and no source", e)
	}
	if len(a.BonusAction) != 1 || a.BonusAction[0].Name != "Fey Step" || len(a.Reaction) != 1 {
		t.Errorf("bonus=%+v reaction=%+v", a.BonusAction, a.Reaction)
	}
}

func TestMap_DefaultReactionsAreDefaultsToo(t *testing.T) {
	d := docOf("PC Name", "Puck").withTM("Reaction 1", "Opportunity Attack", "Opportunity Attack#!#Default reaction")

	r := Map(d).Sheet.Actions.Reaction
	if len(r) != 1 || !r[0].IsDefault || r[0].Source != "" {
		t.Errorf("reaction = %+v, want is_default and no source", r)
	}
}

func TestMap_LimitedUsesKeepTextAndTraceTheirMagicItem(t *testing.T) {
	res := Map(docOf(
		"Extra.Magic Item 1", "Ring of Jaedear (Level 1)",
		"Limited Feature 1", "Balm of the Summer Court", "Limited Feature Max Usages 1", "2d6", "Limited Feature Recovery 1", "long rest",
		"Limited Feature 2", "Ring of Jaedear (Level 1) (Shield)", "Limited Feature Max Usages 2", "1", "Limited Feature Recovery 2", "dawn",
		"Limited Feature Used 2", "1",
	))

	uses := res.Sheet.LimitedUses
	if len(uses) != 2 {
		t.Fatalf("uses = %+v", uses)
	}
	if uses[0].Max != "2d6" || uses[0].Recovery != "long rest" || uses[0].Source != "" {
		t.Errorf("use 1 = %+v", uses[0])
	}
	if uses[1].Source != "Ring of Jaedear" || uses[1].Recovery != "dawn" {
		t.Errorf("use 2 = %+v, want the source to be the item", uses[1])
	}
}

// --- proficiencies, magic items, notes ---

func TestMap_ProficienciesMarkUnresolvedChoices(t *testing.T) {
	res := Map(docOf(
		"Proficiency Armor Light", "/True", "Proficiency Shields", "/True", "Proficiency Weapon Simple", "/True",
		"Tool 1", "Trance: tool or weapon (1/2)", "Tool 2", "Cartographer's Tools",
		"Language 1", "Common", "Language 2", "Sylvan", "Language 3", "from Human",
	))

	p := res.Sheet.Proficiencies
	if len(p.Armor) != 2 || p.Armor[0].Name != "Light armor" || p.Armor[1].Name != "Shields" || len(p.Weapons) != 1 {
		t.Errorf("armor=%+v weapons=%+v", p.Armor, p.Weapons)
	}
	pending := map[string]bool{}
	for _, x := range append(p.Tools, p.Languages...) {
		pending[x.Name] = x.Pending
	}
	if !pending["Trance: tool or weapon (1/2)"] || !pending["from Human"] || pending["Cartographer's Tools"] || pending["Common"] || pending["Sylvan"] {
		t.Errorf("pending flags = %v", pending)
	}
}

func TestMap_MagicItemStageSourceAndAttunement(t *testing.T) {
	res := Map(docOf(
		"Extra.Magic Item 1", "Ring of Jaedear (Level 1)", "Extra.Magic Item Note 1", "[HB-RoJ 1]",
		"Extra.Magic Item Description 1", "Rough stone surface.", "Extra.Magic Item Attuned 1", "/True",
		"Extra.Magic Item 2", "Cloak of Protection (+1)",
	))

	items := res.Sheet.MagicItems
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if it := items[0]; it.Name != "Ring of Jaedear" || it.Stage != "Level 1" || it.Source != "HB-RoJ 1" || it.Text != "Rough stone surface." || !it.Attuned || it.AttunementRequired {
		t.Errorf("item 1 = %+v", it)
	}
	if it := items[1]; it.Name != "Cloak of Protection (+1)" || it.Stage != "" || it.Attuned {
		t.Errorf("item 2 = %+v, want a non-level parenthesis to stay in the name", it)
	}
}

func TestMap_NotesCarryPersonalityAndTheNotesPagesFreeText(t *testing.T) {
	res := Map(docOf(
		"Template.extras.ASnotes", ",P5.ASnotes.",
		"Personality Trait", "Esta es la personalidad de Puck.", "Ideal", "Ideales", "Bond", "Un bond", "Flaw", "Una falla",
		"Background_History", "Criado por dríades.",
		"P5.ASnotes.Notes.Left", "Puck's Moss-Whispered Respite: \rOnce per Long Rest, I can touch a creature.",
	))

	n := res.Sheet.Notes
	if n.Personality != "Esta es la personalidad de Puck." || n.Ideals != "Ideales" || n.Bonds != "Un bond" || n.Flaws != "Una falla" || n.Backstory != "Criado por dríades." {
		t.Errorf("notes = %+v", n)
	}
	if !strings.Contains(n.FreeText, "Once per Long Rest") {
		t.Errorf("free text = %q, want the notes page text that is not a feature row", n.FreeText)
	}
}

// --- state and warnings ---

func TestMap_SpentStateIsNotImported(t *testing.T) {
	base := docOf("PC Name", "Puck", "HP Max", "21", "Limited Feature 1", "Fey Step", "Limited Feature Max Usages 1", "2")
	spent := docOf("PC Name", "Puck", "HP Max", "21", "Limited Feature 1", "Fey Step", "Limited Feature Max Usages 1", "2",
		"HP Current", "7", "HP Temp", "5", "HD1 Used", "2", "Limited Feature Used 1", "2",
		"Death Save Fail1", "/True", "Death Save Success1", "/True", "Inspiration", "/True")

	if !reflect.DeepEqual(Map(base).Sheet, Map(spent).Sheet) {
		t.Error("spent state changed the mapped sheet")
	}
}

func TestMap_InvalidMaxHPAndMissingNameWarn(t *testing.T) {
	for name, d := range map[string]*Document{
		"missing": docOf("PC Name", "Puck"),
		"zero":    docOf("PC Name", "Puck", "HP Max", "0"),
		"text":    docOf("PC Name", "Puck", "HP Max", "many"),
	} {
		res := Map(d)
		if res.MaxHP != 0 || len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "max HP") {
			t.Errorf("%s: maxHP=%d warnings=%v", name, res.MaxHP, res.Warnings)
		}
	}
	if res := Map(docOf("HP Max", "10")); res.Name != "" || len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "name") {
		t.Errorf("no name: %+v", res.Warnings)
	}
}

func TestMap_AnotherMajorVersionWarns(t *testing.T) {
	d := docOf("PC Name", "Puck", "HP Max", "10")
	d.Info.Version = "13.2.3"

	res := Map(d)
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "13.2.3") {
		t.Errorf("warnings = %v", res.Warnings)
	}
	d.Info.Version = "14.1.0"
	if res := Map(d); len(res.Warnings) != 0 {
		t.Errorf("a 14.x version warned: %v", res.Warnings)
	}
}

func TestMap_SkippedValuesAreReported(t *testing.T) {
	d := docOf("PC Name", "Puck", "HP Max", "10")
	d.Skipped = 2

	if res := Map(d); len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "2 oversized") {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestMap_AnEmptySheetMapsToNoSections(t *testing.T) {
	res := Map(docOf("PC Name", "Puck", "HP Max", "10"))

	s := res.Sheet
	if s.Abilities != nil || s.Skills != nil || s.Combat != nil || s.Defenses != nil || s.Actions != nil ||
		s.Proficiencies != nil || s.Spellcasting != nil || s.Notes != nil || len(s.Attacks) != 0 || len(s.Features) != 0 {
		t.Errorf("sheet = %+v, want no sections", s)
	}
}

// --- features, feats, masteries ---

func TestMap_FeatureRowIsSplitIntoItsFields(t *testing.T) {
	res := Map(docOf("Class Features", "**Beast Master, level 3:**\r#◆ Favored Enemy# (Ranger 1, P24 119) #[2× per LR or SS]#\r   I always have the Hunter's Mark spell prepared.\r   I can cast it without a slot.\r#◆ Spellcasting# (Ranger 1, P24 119) #[4 spells prepared]#\r   I can cast ranger spells."))

	if len(res.Sheet.Features) != 2 {
		t.Fatalf("features = %+v", res.Sheet.Features)
	}
	f := res.Sheet.Features[0]
	if f.Kind != "class" || f.Name != "Favored Enemy" || f.Source != "Ranger" || ptrInt(f.Level) != 1 || f.Book != "P24" || f.Page != "119" || f.UsesNote != "2× per LR or SS" {
		t.Errorf("feature = %+v", f)
	}
	if f.Text != "I always have the Hunter's Mark spell prepared.\nI can cast it without a slot." {
		t.Errorf("text = %q", f.Text)
	}
	for _, s := range []string{f.Name, f.Text, f.UsesNote, res.Sheet.Features[1].Text} {
		if strings.ContainsAny(s, "#◆") || strings.Contains(s, "**") {
			t.Errorf("marker characters left in %q", s)
		}
	}
}

func TestMap_RowTextOnTheSameLineAndRowsWithoutALevel(t *testing.T) {
	res := Map(docOf("Extra.Notes", "#◆ Fighting Style [Archery]# (Ranger 2, P24 120) I gain a +2 bonus to ranged attacks.\r#◆ Kid Tricks# (Kid Tricks, HB-KT 1)\r   ME LA RE BANCO."))

	fs := res.Sheet.Features
	if len(fs) != 2 {
		t.Fatalf("features = %+v", fs)
	}
	if fs[0].Name != "Fighting Style [Archery]" || fs[0].Text != "I gain a +2 bonus to ranged attacks." {
		t.Errorf("inline-text row = %+v", fs[0])
	}
	if fs[1].Source != "Kid Tricks" || fs[1].Level != nil || fs[1].Book != "HB-KT" || fs[1].Page != "1" {
		t.Errorf("row without a level = %+v", fs[1])
	}
}

func TestMap_AFeatInTwoPlacesBecomesOneWithTheLongerText(t *testing.T) {
	res := Map(docOf(
		"Template.extras.ASnotes", ",P5.ASnotes.",
		"Feat Name 1", "Kid Tricks", "Feat Description 1", "Short text.", "Feat Note 1", "[HB-KT 1]",
		"Feat Name 2", "Alert [Origin]", "Feat Description 2", "I add my Prof Bonus to Initiative.", "Feat Note 2", "[P24 201]",
		"P5.ASnotes.Notes.Right", "#◆ Kid Tricks# (Kid Tricks, HB-KT 1)\r   The full, much longer description of the kid tricks feat.",
	))

	var kids []string
	feats := 0
	for _, f := range res.Sheet.Features {
		if f.Kind == "feat" {
			feats++
		}
		if f.Name == "Kid Tricks" {
			kids = append(kids, f.Text)
			if f.Kind != "feat" {
				t.Errorf("kind = %q, want feat", f.Kind)
			}
		}
	}
	if len(kids) != 1 || kids[0] != "The full, much longer description of the kid tricks feat." {
		t.Errorf("Kid Tricks = %q", kids)
	}
	if feats != 2 {
		t.Errorf("%d feats, want 2", feats)
	}
	for _, f := range res.Sheet.Features {
		if f.Name == "Alert [Origin]" && (f.Book != "P24" || f.Page != "201") {
			t.Errorf("feat note = %+v", f)
		}
	}
}

func TestParseNote_UsesTheLastBracketedSource(t *testing.T) {
	for in, want := range map[string][2]string{
		"[P24 201]":                    {"P24", "201"},
		"[HB-KT 1]":                    {"HB-KT", "1"},
		"Human bonus feat [P24 200]":   {"P24", "200"},
		"[Human bonus feat [P24 200]]": {"P24", "200"},
		"[Homebrew]":                   {"Homebrew", ""},
	} {
		if book, page := parseNote(in); book != want[0] || page != want[1] {
			t.Errorf("parseNote(%q) = %q, %q, want %q, %q", in, book, page, want[0], want[1])
		}
	}
}

func TestMap_WeaponMasteriesAreNotFeatures(t *testing.T) {
	res := Map(docOf("Extra.Notes", "#◆ Stealth Expertise# (Ranger Expertise, P24 120)\r#◆ Sling# (Weapon Mastery)\r   I gain access to the Sling's 'Slow' Mastery feature\r   Slow : reduce its Speed by 10 feet.\r#◆ Dagger# (Weapon Mastery)\r   I gain access to the Dagger's 'Nick' Mastery feature"))

	wm := res.Sheet.WeaponMasteries
	if len(wm) != 2 || wm[0].Weapon != "Sling" || wm[0].Mastery != "Slow" || wm[1].Weapon != "Dagger" || wm[1].Mastery != "Nick" {
		t.Errorf("masteries = %+v", wm)
	}
	for _, f := range res.Sheet.Features {
		if f.Name == "Sling" || f.Name == "Dagger" {
			t.Errorf("a weapon mastery became a feature: %+v", f)
		}
	}
	if len(res.Sheet.Features) != 1 || res.Sheet.Features[0].Name != "Stealth Expertise" {
		t.Errorf("features = %+v", res.Sheet.Features)
	}
}

func TestMap_RacialTraitsSplitOnTheirBullets(t *testing.T) {
	res := Map(docOf("Racial Traits", "Eladrin (my creature type is humanoid, elf)\r • Trance: I don't need to sleep.\r   Meditating for 4 hours is enough.\r • Fey Ancestry: Advantage vs charmed."))

	var sp []string
	for _, f := range res.Sheet.Features {
		if f.Kind == "species" {
			sp = append(sp, f.Name+"|"+f.Text)
		}
	}
	want := []string{"Trance|I don't need to sleep.\nMeditating for 4 hours is enough.", "Fey Ancestry|Advantage vs charmed."}
	if !reflect.DeepEqual(sp, want) {
		t.Errorf("species features = %q, want %q", sp, want)
	}
}

func TestCleanText(t *testing.T) {
	for in, want := range map[string]string{
		"**Bold** and #◆ Row#":     "Bold and Row",
		"a\r\n\r\n\r\n\r\nb":       "a\n\nb",
		"  x  y  ":                 "x y",
		"##◆ Slippery##\r   Text.": "Slippery\nText.",
	} {
		if got := cleanText(in); got != want {
			t.Errorf("cleanText(%q) = %q, want %q", in, got, want)
		}
	}
}
