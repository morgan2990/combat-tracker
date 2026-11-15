package mpmb

import (
	"regexp"
	"strconv"
	"strings"

	"combatapp/store"
)

// warlockPact is Pact Magic by Warlock level: {slots, slot level}. The sheet
// only stores the combined slot totals, so this table is how the Pact Magic
// part is separated back out. It is the one rules table the import carries.
var warlockPact = [21][2]int{
	{}, {1, 1}, {2, 1}, {2, 2}, {2, 2}, {2, 3}, {2, 3}, {2, 4}, {2, 4}, {2, 5}, {2, 5},
	{3, 5}, {3, 5}, {3, 5}, {3, 5}, {3, 5}, {3, 5}, {4, 5}, {4, 5}, {4, 5}, {4, 5},
}

// spellSource is what CurrentSpells records about one spell list.
type spellSource struct {
	Name  string
	Kind  string // class | feat | item | other
	Level int    // class level, 0 when absent
	Pact  bool   // Warlock-style Pact Magic
}

// spellSources parses CurrentSpells.Stringified. It never fails the import:
// on any problem it warns and returns nil, and the caller falls back to
// treating every list as a class list with combined slots.
func (m *mapper) spellSources() map[string]spellSource {
	raw := m.d.Get("CurrentSpells.Stringified")
	// The field holds several literals joined by a run of #: the spell sources
	// first, then unrelated state (spell lists to include or exclude).
	if i := strings.Index(raw, "##########"); i >= 0 {
		raw = raw[:i]
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	v, err := ParseLiteral(raw)
	if err != nil {
		m.warn("Spell source details could not be read (%v): spell lists were treated as class lists and Pact Magic was left combined with the other slots.", err)
		return nil
	}
	top, ok := v.(map[string]any)
	if !ok {
		m.warn("Spell source details had an unexpected shape: spell lists were treated as class lists.")
		return nil
	}
	out := make(map[string]spellSource, len(top))
	for key, entry := range top {
		s := spellSource{Name: litString(lit(entry, "name")), Kind: "class"}
		// A subclass-only name ("Beast Master") reads better with its class.
		if short := litString(lit(entry, "shortname")); short != "" && s.Name != "" && !strings.Contains(s.Name, short) && litString(lit(entry, "refType")) != "feat" && litString(lit(entry, "refType")) != "item" {
			s.Name = short + " (" + s.Name + ")"
		}
		switch litString(lit(entry, "refType")) {
		case "feat":
			s.Kind = "feat"
			s.Name = strings.TrimSuffix(s.Name, " (feat)")
		case "item":
			s.Kind = "item"
			s.Name = strings.TrimSuffix(s.Name, " (item)")
		case "class", "":
		default:
			s.Kind = "other"
		}
		s.Level, _ = litInt(lit(entry, "level"))
		if factor, ok := lit(entry, "factor").([]any); ok && len(factor) >= 2 && litString(factor[1]) == "warlock" {
			s.Pact = true
		}
		out[key] = s
	}
	return out
}

var levelLabelRE = regexp.MustCompile(`^(\d+)(?:st|nd|rd|th) Level`)

// levelFromLabel reads a section label: "Cantrips (0 Level)" is 0, "3rd Level" is 3.
func levelFromLabel(label string) (int, bool) {
	label = strings.TrimSpace(label)
	if strings.HasPrefix(strings.ToLower(label), "cantrip") {
		return 0, true
	}
	if g := levelLabelRE.FindStringSubmatch(label); g != nil {
		return parseInt(g[1])
	}
	return 0, false
}

// preparation maps a spell row's check field to a preparation state. An
// unknown value becomes "other", keeping the original as the mark.
func preparation(check string) (prep, mark string) {
	switch check {
	case "markedbox":
		return "always", ""
	case "checkedbox":
		return "prepared", ""
	case "checkbox":
		return "unprepared", ""
	case "":
		return "known", ""
	case "atwill":
		return "at_will", ""
	case "oncelr":
		return "limited", ""
	}
	return "other", check
}

// spellcasting maps the spell pages, slots and Pact Magic. It returns nil
// when the sheet has none of them.
func (m *mapper) spellcasting() *store.SheetSpellcasting {
	sources := m.spellSources()

	pages := append(m.prefixes("SSfront"), m.prefixes("SSmore")...)

	// List details (ability, DC, attack, prepare count) from the headers.
	type head struct {
		ability, dc, attack string
		prepare             *int
	}
	heads := map[string]head{}
	var order []string
	lists := map[string]*store.SpellList{}
	listFor := func(key string) *store.SpellList {
		if l, ok := lists[key]; ok {
			return l
		}
		src, known := sources[key]
		l := &store.SpellList{Kind: "class", Name: titleCase(key)}
		if known {
			l.Kind = src.Kind
			if src.Name != "" {
				l.Name = src.Name
			}
		}
		lists[key] = l
		order = append(order, key)
		return l
	}
	for _, p := range pages {
		for _, i := range m.indices(p+"spellshead.class.", "") {
			key := strings.ToLower(m.get(p + "spellshead.class." + strconv.Itoa(i)))
			if key == "" {
				continue
			}
			n := strconv.Itoa(i)
			h := head{
				ability: abilityName(m.get(p + "spellshead.ability." + n)),
				dc:      m.get(p + "spellshead.dc." + n),
				attack:  m.get(p + "spellshead.attack." + n),
			}
			if c, ok := m.num(p + "spellshead.prepare." + n); ok && c > 0 {
				h.prepare = &c
			}
			if _, dup := heads[key]; !dup {
				heads[key] = h
			}
			listFor(key)
		}
	}

	// Rows, in order, across pages: dividers set the level of what follows.
	// The first section has no divider row; its label is spellsdiv.Text.0
	// ("Cantrips (0 Level)", or "1st Level" for a class with no cantrips).
	level := 0
	if len(pages) > 0 {
		if lv, ok := levelFromLabel(m.get(pages[0] + "spellsdiv.Text.0")); ok {
			level = lv
		}
	}
	for _, p := range pages {
		for _, i := range m.indices(p+"spells.remember.", "") {
			n := strconv.Itoa(i)
			rem := m.get(p + "spells.remember." + n)
			switch {
			case rem == "" || rem == "hidethisline" || strings.HasPrefix(rem, "setcaptions") || strings.HasPrefix(rem, "setheader##"):
				continue
			case strings.HasPrefix(rem, "setdivider##"):
				parts := strings.Split(rem, "##")
				if len(parts) >= 2 {
					if lv, ok := parseInt(parts[1]); ok {
						level = lv
					}
				}
				continue
			}
			name := m.get(p + "spells.name." + n)
			if name == "" {
				continue
			}
			parts := strings.Split(rem, "##")
			key := ""
			if len(parts) >= 3 {
				key = strings.ToLower(parts[2])
			}
			if key == "" {
				m.warn("Spell %q has no spell list and was skipped.", name)
				continue
			}
			check := m.get(p + "spells.check." + n)
			prep, mark := preparation(check)
			if prep == "other" {
				m.warn("Spell %q has an unrecognized marker %q; it was kept as \"other\".", name, check)
			}
			l := listFor(key)
			l.Spells = append(l.Spells, store.Spell{
				Key:         parts[0],
				Name:        name,
				Level:       level,
				Preparation: prep,
				Mark:        mark,
				Save:        m.get(p + "spells.save." + n),
				School:      m.get(p + "spells.school." + n),
				CastingTime: m.get(p + "spells.time." + n),
				Range:       m.get(p + "spells.range." + n),
				Components:  m.get(p + "spells.components." + n),
				Duration:    m.get(p + "spells.duration." + n),
				Book:        m.get(p + "spells.book." + n),
				Page:        m.get(p + "spells.page." + n),
				Description: m.get(p + "spells.description." + n),
			})
		}
	}

	out := &store.SheetSpellcasting{}
	for _, key := range order {
		l := lists[key]
		h := heads[key]
		l.Ability, l.SaveDC, l.Attack, l.PrepareCount = h.ability, h.dc, h.attack, h.prepare
		if len(l.Spells) > 0 || h.dc != "" || h.attack != "" || h.ability != "" {
			out.SpellLists = append(out.SpellLists, *l)
		}
	}

	// Slots: the sheet shows combined totals, Pact Magic included.
	var totals [9]int
	for lv := 1; lv <= 9; lv++ {
		if n, ok := m.num("SpellSlots.CheckboxesSet.lvl" + strconv.Itoa(lv)); ok && n > 0 {
			totals[lv-1] = n
		}
	}
	for _, src := range sources {
		if !src.Pact {
			continue
		}
		if src.Level < 1 || src.Level > 20 {
			m.warn("Pact Magic was found but the Warlock level (%d) is not valid; slots were left combined.", src.Level)
			break
		}
		count, slotLevel := warlockPact[src.Level][0], warlockPact[src.Level][1]
		if totals[slotLevel-1] < count {
			m.warn("Pact Magic (%d level-%d slots for Warlock %d) does not fit in the sheet's combined slot total (%d); slots were left combined.", count, slotLevel, src.Level, totals[slotLevel-1])
			break
		}
		totals[slotLevel-1] -= count
		out.PactMagic = &store.PactMagic{SlotLevel: slotLevel, Total: count}
		break
	}
	last := -1
	for i, n := range totals {
		if n > 0 {
			last = i
		}
	}
	if last >= 0 {
		out.Slots = append([]int(nil), totals[:last+1]...)
	}

	if len(out.SpellLists) == 0 && out.Slots == nil && out.PactMagic == nil {
		return nil
	}
	return out
}
