package mpmb

import (
	"strings"

	"combatapp/store"
)

// MapperVersion identifies the mapping rules. It is stored with each import,
// so a sheet imported by an older mapper can be told apart and re-mapped.
const MapperVersion = "1"

// Map turns a read sheet into the app's model. It never fails: a value that
// does not fit is left out and reported in Warnings.
func Map(d *Document) store.ImportParsed {
	m := newMapper(d)
	out := store.ImportParsed{}

	if major := strings.SplitN(d.Info.Version, ".", 2)[0]; major != "14" {
		m.warn("This is MPMB sheet version %q; the importer was built for version 14, so some fields may be missing or misread.", d.Info.Version)
	}

	out.Name = m.get("PC Name")
	if out.Name == "" {
		m.warn("The sheet has no character name; the character's name was left as it is.")
	}
	if hp, ok := m.num("HP Max"); ok && hp > 0 {
		out.MaxHP = hp
	} else {
		m.warn("Max HP is missing or not a positive number; the character's max HP was left as it is.")
	}

	s := &out.Sheet
	s.SchemaVersion = store.SheetSchemaVersion
	s.Identity = m.identity()
	s.Abilities = m.abilities()
	s.Skills = m.skills()
	s.Combat = m.combat()
	s.Defenses = m.defenses()
	s.Attacks = m.attacks()
	s.Actions = m.actions()
	s.MagicItems = m.magicItems()
	s.LimitedUses = m.limitedUses(s.MagicItems)
	s.Proficiencies = m.proficiencies()
	s.Spellcasting = m.spellcasting()
	s.WildShapes = m.wildShapes()
	out.Companions = m.companions()
	out.Inventory = m.inventory()
	fs := m.featureSection()
	s.Features = fs.features
	s.WeaponMasteries = fs.masteries
	s.Notes = m.notes(fs.free)

	if d.Skipped > 0 {
		m.warn("%d oversized field value(s) in the PDF were skipped.", d.Skipped)
	}
	out.Warnings = m.warns
	return out
}

// --- identity ---

func (m *mapper) identity() *store.SheetIdentity {
	id := &store.SheetIdentity{
		Player:         m.get("Player Name"),
		ClassesText:    m.get("Class and Levels"),
		CharacterLevel: m.intPtr("Character Level"),
		Species:        m.get("Race"),
		Background:     m.get("Background"),
		Size:           strings.TrimSpace(m.d.Display("Size Category")),
		Alignment:      m.get("Alignment"),
		XP:             m.intPtr("Total Experience"),
		Classes:        classesFromFeatures(m.d.Get("Class Features")),
	}
	if id.Player == "" && id.ClassesText == "" && id.CharacterLevel == nil && id.Species == "" && id.Background == "" &&
		id.Size == "" && id.Alignment == "" && id.XP == nil && len(id.Classes) == 0 {
		return nil
	}
	return id
}

// --- abilities ---

var abilityKeys = []string{"Str", "Dex", "Con", "Int", "Wis", "Cha"}

func (m *mapper) abilityScore(prefix string) store.AbilityScore {
	return store.AbilityScore{
		Score:          m.intPtr(prefix),
		Mod:            m.intPtr(prefix + " Mod"),
		SaveBonus:      m.intPtr(prefix + " ST Mod"),
		SaveProficient: m.checked(prefix + " ST Prof"),
	}
}

func (m *mapper) abilities() *store.SheetAbilities {
	a := &store.SheetAbilities{
		Str: m.abilityScore("Str"), Dex: m.abilityScore("Dex"), Con: m.abilityScore("Con"),
		Int: m.abilityScore("Int"), Wis: m.abilityScore("Wis"), Cha: m.abilityScore("Cha"),
	}
	for _, s := range []store.AbilityScore{a.Str, a.Dex, a.Con, a.Int, a.Wis, a.Cha} {
		if s.Score != nil || s.Mod != nil || s.SaveBonus != nil {
			return a
		}
	}
	return nil
}

// --- skills ---

// skillTable is the sheet's 18 skills: field prefix, name, ability.
var skillTable = []struct{ prefix, name, ability string }{
	{"Acr", "Acrobatics", "Dex"}, {"Ani", "Animal Handling", "Wis"}, {"Arc", "Arcana", "Int"},
	{"Ath", "Athletics", "Str"}, {"Dec", "Deception", "Cha"}, {"His", "History", "Int"},
	{"Ins", "Insight", "Wis"}, {"Inti", "Intimidation", "Cha"}, {"Inv", "Investigation", "Int"},
	{"Med", "Medicine", "Wis"}, {"Nat", "Nature", "Int"}, {"Perc", "Perception", "Wis"},
	{"Perf", "Performance", "Cha"}, {"Pers", "Persuasion", "Cha"}, {"Rel", "Religion", "Int"},
	{"Sle", "Sleight of Hand", "Dex"}, {"Ste", "Stealth", "Dex"}, {"Sur", "Survival", "Wis"},
}

func (m *mapper) skills() *store.SheetSkills {
	sk := &store.SheetSkills{
		PassivePerception:    m.intPtr("Passive Perception"),
		PassiveInsight:       m.intPtr("Passive Insight"),
		PassiveInvestigation: m.intPtr("Passive Investigation"),
	}
	for _, t := range skillTable {
		bonus := m.intPtr(t.prefix)
		if bonus == nil {
			continue
		}
		sk.Entries = append(sk.Entries, store.SkillEntry{
			Key: t.name, Ability: t.ability, Bonus: bonus,
			Proficient: m.checked(t.prefix + " Prof"), Expertise: m.checked(t.prefix + " Exp"),
		})
	}
	if len(sk.Entries) == 0 && sk.PassivePerception == nil && sk.PassiveInsight == nil && sk.PassiveInvestigation == nil {
		return nil
	}
	return sk
}

// --- combat ---

func (m *mapper) combat() *store.SheetCombat {
	c := &store.SheetCombat{
		AC:               m.intPtr("AC"),
		Initiative:       m.intPtr("Initiative bonus"),
		ProficiencyBonus: m.intPtr("Proficiency Bonus"),
		SpeedText:        m.get("Speed"),
	}
	if u := strings.ToLower(m.get("Unit System")); u == "" || u == "imperial" {
		c.Speeds = parseSpeeds(c.SpeedText)
	}
	for _, n := range []string{"1", "2", "3"} {
		die, okDie := m.num("HD" + n + " Die")
		total, okTotal := m.num("HD" + n + " Level")
		if okDie && okTotal && total > 0 {
			c.HitDice = append(c.HitDice, store.HitDie{Die: die, Total: total})
		}
	}
	if c.AC == nil && c.Initiative == nil && c.ProficiencyBonus == nil && c.SpeedText == "" && len(c.HitDice) == 0 {
		return nil
	}
	return c
}

// parseSpeeds reads imperial speed text such as "40 ft, climb 30 ft, burrow 15 ft".
// An unlabeled number is the walking speed.
func parseSpeeds(text string) store.Speeds {
	var sp store.Speeds
	for part := range strings.SplitSeq(text, ",") {
		fields := strings.Fields(strings.ToLower(part))
		mode, num := "walk", 0
		found := false
		for _, f := range fields {
			if n, ok := parseInt(f); ok && !found {
				num, found = n, true
				continue
			}
			switch f {
			case "walk", "climb", "swim", "fly", "burrow":
				mode = f
			}
		}
		if !found {
			continue
		}
		n := num
		switch mode {
		case "walk":
			sp.Walk = &n
		case "climb":
			sp.Climb = &n
		case "swim":
			sp.Swim = &n
		case "fly":
			sp.Fly = &n
		case "burrow":
			sp.Burrow = &n
		}
	}
	return sp
}

// --- defenses ---

func (m *mapper) defenses() *store.SheetDefenses {
	d := &store.SheetDefenses{
		Vision:    m.get("Vision"),
		SaveNotes: m.text("Saving Throw advantages / disadvantages"),
	}
	for _, n := range m.indices("Resistance Damage Type ", "") {
		if r := m.get("Resistance Damage Type " + itoa(n)); r != "" {
			d.Resistances = append(d.Resistances, r)
		}
	}
	if d.Vision == "" && d.SaveNotes == "" && len(d.Resistances) == 0 {
		return nil
	}
	return d
}

// --- attacks and actions ---

func (m *mapper) attacks() []store.Attack { return m.attackRows("Attack.") }

// actionList reads "<prefix> 1", "<prefix> 2", ... A field's /TM holds
// "name#!#source"; the source "Default action" marks the standard actions.
func (m *mapper) actionList(prefix string) []store.ActionEntry {
	var out []store.ActionEntry
	for _, n := range m.indices(prefix+" ", "") {
		name := m.get(prefix + " " + itoa(n))
		if name == "" {
			continue
		}
		e := store.ActionEntry{Name: name}
		if _, src, ok := strings.Cut(m.d.TM(prefix+" "+itoa(n)), "#!#"); ok {
			// "Default action", "Default reaction": the standard rules actions.
			if src = strings.TrimSpace(src); strings.HasPrefix(strings.ToLower(src), "default ") {
				e.IsDefault = true
			} else {
				e.Source = src
			}
		}
		out = append(out, e)
	}
	return out
}

func (m *mapper) actions() *store.SheetActions {
	a := &store.SheetActions{
		Action:      m.actionList("Action"),
		BonusAction: m.actionList("Bonus Action"),
		Reaction:    m.actionList("Reaction"),
	}
	if len(a.Action) == 0 && len(a.BonusAction) == 0 && len(a.Reaction) == 0 {
		return nil
	}
	return a
}

// --- limited uses and magic items ---

func (m *mapper) limitedUses(items []store.MagicItem) []store.LimitedUse {
	var out []store.LimitedUse
	for _, n := range m.indices("Limited Feature ", "") {
		i := itoa(n)
		name := m.get("Limited Feature " + i)
		if name == "" {
			continue
		}
		out = append(out, store.LimitedUse{
			Name:     name,
			Max:      m.get("Limited Feature Max Usages " + i),
			Recovery: m.get("Limited Feature Recovery " + i),
			Source:   itemSource(name, m.printedItemNames(), items),
		})
	}
	return out
}

// printedItemNames are the magic item names exactly as the sheet prints them
// (stage included), by slot, so a use named "Ring of Jaedear (Level 1) (Shield)"
// can be traced back to the item it belongs to.
func (m *mapper) printedItemNames() []string {
	var out []string
	for _, n := range m.indices("Extra.Magic Item ", "") {
		if name := m.get("Extra.Magic Item " + itoa(n)); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// itemSource returns the stored name of the magic item whose printed name
// starts a limited use's name, or "". printed and items are in the same order.
func itemSource(use string, printed []string, items []store.MagicItem) string {
	best, bestLen := "", 0
	for i, p := range printed {
		if i < len(items) && strings.HasPrefix(use, p) && len(p) > bestLen {
			best, bestLen = items[i].Name, len(p)
		}
	}
	return best
}

func (m *mapper) magicItems() []store.MagicItem {
	var out []store.MagicItem
	for _, n := range m.indices("Extra.Magic Item ", "") {
		i := itoa(n)
		printed := m.get("Extra.Magic Item " + i)
		if printed == "" {
			continue
		}
		name, stage := splitStage(printed)
		it := store.MagicItem{
			Name:    name,
			Stage:   stage,
			Attuned: m.checked("Extra.Magic Item Attuned " + i),
			Text:    m.text("Extra.Magic Item Description " + i),
			Source:  strings.Trim(m.get("Extra.Magic Item Note "+i), "[] "),
		}
		out = append(out, it)
	}
	return out
}

// splitStage separates a progressing item's stage: "Ring of Jaedear (Level 1)"
// is the item "Ring of Jaedear" at stage "Level 1".
func splitStage(printed string) (name, stage string) {
	if i := strings.LastIndex(printed, " ("); i > 0 && strings.HasSuffix(printed, ")") {
		inner := printed[i+2 : len(printed)-1]
		if strings.HasPrefix(strings.ToLower(inner), "level ") {
			if _, ok := parseInt(inner[len("level "):]); ok {
				return strings.TrimSpace(printed[:i]), inner
			}
		}
	}
	return printed, ""
}

// --- proficiencies ---

func isPending(name string) bool {
	l := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(l, "from ") {
		return true
	}
	return strings.HasSuffix(l, ")") && pendingRE.MatchString(l)
}

func (m *mapper) proficiencies() *store.SheetProficiencies {
	list := func(names ...string) []store.Proficiency {
		var out []store.Proficiency
		for _, n := range names {
			if n != "" {
				out = append(out, store.Proficiency{Name: n, Pending: isPending(n)})
			}
		}
		return out
	}
	numbered := func(prefix string) []string {
		var out []string
		for _, n := range m.indices(prefix, "") {
			out = append(out, m.get(prefix+itoa(n)))
		}
		return out
	}
	other := func(field string) []string {
		var out []string
		for part := range strings.SplitSeq(m.get(field), ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
		return out
	}
	var armor, weapons []string
	for _, a := range []struct{ field, name string }{
		{"Proficiency Armor Light", "Light armor"}, {"Proficiency Armor Medium", "Medium armor"},
		{"Proficiency Armor Heavy", "Heavy armor"}, {"Proficiency Shields", "Shields"},
	} {
		if m.checked(a.field) {
			armor = append(armor, a.name)
		}
	}
	armor = append(armor, other("Proficiency Armor Other")...)
	for _, w := range []struct{ field, name string }{
		{"Proficiency Weapon Simple", "Simple weapons"}, {"Proficiency Weapon Martial", "Martial weapons"},
	} {
		if m.checked(w.field) {
			weapons = append(weapons, w.name)
		}
	}
	weapons = append(weapons, other("Proficiency Weapon Other Description")...)

	p := &store.SheetProficiencies{
		Armor:     list(armor...),
		Weapons:   list(weapons...),
		Tools:     list(numbered("Tool ")...),
		Languages: list(numbered("Language ")...),
	}
	if len(p.Armor)+len(p.Weapons)+len(p.Tools)+len(p.Languages) == 0 {
		return nil
	}
	return p
}

// --- notes ---

// notes maps the description fields; free is the text of the notes pages that
// is not a feature row.
func (m *mapper) notes(free []string) *store.SheetNotes {
	n := &store.SheetNotes{
		Appearance:  m.text("Background_Appearance"),
		Personality: m.text("Personality Trait"),
		Ideals:      m.text("Ideal"),
		Bonds:       m.text("Bond"),
		Flaws:       m.text("Flaw"),
		Backstory:   m.text("Background_History"),
		FreeText:    strings.TrimSpace(strings.Join(free, "\n\n")),
	}
	if *n == (store.SheetNotes{}) {
		return nil
	}
	return n
}
