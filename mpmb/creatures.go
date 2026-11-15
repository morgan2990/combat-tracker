package mpmb

import (
	"strings"

	"combatapp/store"
)

// attackRows reads attack rows "<prefix>N.Weapon" (and "...Weapon Selection").
// Rows with no name are blank slots and are skipped.
func (m *mapper) attackRows(prefix string) []store.Attack {
	seen := map[int]bool{}
	var rows []int
	for _, n := range append(m.indices(prefix, ".Weapon"), m.indices(prefix, ".Weapon Selection")...) {
		if !seen[n] {
			seen[n] = true
			rows = append(rows, n)
		}
	}
	sortInts(rows)
	var out []store.Attack
	for _, n := range rows {
		p := prefix + itoa(n) + "."
		name := strings.TrimSpace(m.d.Display(p + "Weapon Selection"))
		if name == "" {
			name = m.get(p + "Weapon")
		}
		if name == "" {
			continue
		}
		out = append(out, store.Attack{
			Name:       name,
			Ability:    attackAbility(m.get(p + "Mod")),
			Range:      m.get(p + "Range"),
			ToHit:      m.get(p + "To Hit"),
			Damage:     m.get(p + "Damage"),
			DamageType: m.get(p + "Damage Type"),
			Notes:      m.text(p + "Description"),
		})
	}
	return out
}

// attackAbility reads an attack's ability: "Cha Mod" on the main sheet, but
// "Comp.Use.Ability.Dex.Mod" (a field path) on a companion's.
func attackAbility(s string) string {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "Comp.Use.Ability."); ok {
		s, _, _ = strings.Cut(rest, ".")
		return s
	}
	return abilityName(s)
}

// creatureAbilities reads "<prefix>Ability.<Str>.Score" and its Mod, ST.Mod, ST.Prof.
func (m *mapper) creatureAbilities(prefix string) store.SheetAbilities {
	get := func(key string) store.AbilityScore {
		p := prefix + "Ability." + key + "."
		return store.AbilityScore{
			Score:          m.intPtr(p + "Score"),
			Mod:            m.intPtr(p + "Mod"),
			SaveBonus:      m.intPtr(p + "ST.Mod"),
			SaveProficient: m.checked(p + "ST.Prof"),
		}
	}
	return store.SheetAbilities{
		Str: get("Str"), Dex: get("Dex"), Con: get("Con"), Int: get("Int"), Wis: get("Wis"), Cha: get("Cha"),
	}
}

// creatureSkills lists the skills a creature is proficient in. Its other
// skills are the plain ability modifiers, which the abilities already carry.
// modPrefix holds each skill's bonus ("<modPrefix><abbr><modSuffix>") and
// profPrefix its competence text: "nothing", "proficient" or "expertise".
func (m *mapper) creatureSkills(modPrefix, modSuffix, profPrefix string) []store.SkillEntry {
	var out []store.SkillEntry
	for _, t := range skillTable {
		prof := strings.ToLower(m.get(profPrefix + t.prefix + ".Prof"))
		if prof != "proficient" && prof != "expertise" {
			continue
		}
		out = append(out, store.SkillEntry{
			Key: t.name, Ability: t.ability, Bonus: m.intPtr(modPrefix + t.prefix + modSuffix),
			Proficient: true, Expertise: prof == "expertise",
		})
	}
	return out
}

// --- companions ---

func (m *mapper) companions() []store.ImportedCompanion {
	var out []store.ImportedCompanion
	for _, page := range m.prefixes("AScomp") {
		b := page + "Comp."
		species := m.get(b + "Race")
		name := m.get(b + "Desc.Name")
		hpText := m.get(b + "Use.HP.Max")
		if name == "" && species == "" && hpText == "" {
			continue // an unused companion page
		}
		if name == "" {
			name = species
		}
		if name == "" {
			name = "Companion"
		}

		c := store.ImportedCompanion{Name: name}
		if hp, ok := parseInt(hpText); ok && hp > 0 {
			c.MaxHP = hp
		} else {
			m.warn("Companion %q has no numeric max HP, so it will be created with 1.", name)
		}

		var traits []string
		for _, f := range []string{"Use.Features", "Use.Traits"} {
			if t := m.text(b + f); t != "" {
				traits = append(traits, t)
			}
		}
		var gear []store.CreatureGear
		for _, n := range m.indices(b+"eqp.Gear Row ", "") {
			if g := m.get(b + "eqp.Gear Row " + itoa(n)); g != "" {
				gear = append(gear, store.CreatureGear{Name: g, Amount: m.get(b + "eqp.Gear Amount " + itoa(n))})
			}
		}

		c.Creature = store.Creature{
			Name:             name,
			Species:          species,
			Type:             m.get(b + "Desc.MonsterType"),
			Size:             strings.TrimSpace(m.d.Display(b + "Desc.Size")),
			Alignment:        m.get(b + "Desc.Alignment"),
			AC:               m.get(b + "Use.AC"),
			HP:               hpText,
			SpeedText:        m.get(b + "Use.Speed"),
			Senses:           m.get(b + "Use.Senses"),
			ProficiencyBonus: m.intPtr(b + "Use.Proficiency Bonus"),
			Abilities:        m.creatureAbilities(b + "Use."),
			Skills:           m.creatureSkills(b+"Use.Skills.", ".Mod", page+"Text.Comp.Use.Skills."),
			Attacks:          m.attackRows(b + "Use.Attack."),
			Traits:           strings.Join(traits, "\n\n"),
			Gear:             gear,
			Notes:            strings.TrimSpace(m.text(page+"Cnote.Left") + "\n\n" + m.text(page+"Cnote.Right")),
		}
		out = append(out, c)
	}
	return out
}

// --- wild shapes ---

func (m *mapper) wildShapes() *store.SheetWildShapes {
	ws := &store.SheetWildShapes{}
	for _, page := range m.prefixes("WSfront") {
		if ws.Uses == "" {
			ws.Uses = m.get(page + "Wildshapes.Info.Uses")
			ws.Duration = m.get(page + "Wildshapes.Info.Duration")
			ws.Limitations = m.get(page + "Wildshapes.Info.Limitations")
		}
		for _, n := range m.indices(page+"Wildshape.Race.", "") {
			name := m.get(page + "Wildshape.Race." + itoa(n))
			if name == "" {
				continue
			}
			b := page + "Wildshape." + itoa(n) + "."
			var notes []string
			if cr := m.get(b + "CR"); cr != "" {
				notes = append(notes, "CR "+cr)
			}
			if hd := m.get(b + "HD"); hd != "" {
				notes = append(notes, "Hit dice "+hd)
			}
			ws.Forms = append(ws.Forms, store.Creature{
				Name:             name,
				Type:             m.get(b + "MonsterType"),
				Size:             strings.TrimSpace(m.d.Display(b + "Size")),
				AC:               m.get(b + "AC"),
				HP:               m.get(b + "HP Max"),
				SpeedText:        m.get(b + "Speed"),
				ProficiencyBonus: m.intPtr(b + "Proficiency Bonus"),
				Abilities:        m.creatureAbilities(b),
				Skills:           m.creatureSkills(b+"Skills.", "", page+"Text.Wildshape."+itoa(n)+".Skills."),
				Attacks:          m.attackRows(b + "Attack."),
				Traits:           m.text(b + "Traits"),
				Notes:            strings.Join(notes, "\n"),
			})
		}
	}
	if ws.Uses == "" && ws.Duration == "" && ws.Limitations == "" && len(ws.Forms) == 0 {
		return nil
	}
	return ws
}

// --- inventory ---

// coinFields maps the sheet's coin fields onto the app's denominations.
func (m *mapper) inventory() store.ImportedInventory {
	inv := store.ImportedInventory{Items: []store.Item{}}
	for _, prefix := range []string{"Adventuring Gear", "Extra.Gear"} {
		for _, n := range m.indices(prefix+" Row ", "") {
			name := m.get(prefix + " Row " + itoa(n))
			if name == "" {
				continue
			}
			qty := 1
			if q, ok := m.num(prefix + " Amount " + itoa(n)); ok && q > 0 {
				qty = q
			}
			inv.Items = append(inv.Items, store.Item{Name: name, Quantity: qty})
		}
	}
	coin := func(field string) int {
		if n, ok := m.num(field); ok && n > 0 {
			return n
		}
		return 0
	}
	inv.Currency = store.Currency{
		PP: coin("Platinum Pieces"), GP: coin("Gold Pieces"), EP: coin("Electrum Pieces"),
		SP: coin("Silver Pieces"), CP: coin("Copper Pieces"),
	}
	inv.Present = len(inv.Items) > 0 || inv.Currency != store.Currency{}
	return inv
}
