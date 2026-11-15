package mpmb

import (
	"strings"
	"testing"
)

// --- companions ---

func companionPage(p string, extra ...string) *Document {
	base := []string{
		p + "Comp.Desc.Name", "Robin Gutierre'", p + "Comp.Race", "Pseudodragon",
		p + "Comp.Desc.MonsterType", "Dragon", p + "Comp.Desc.Alignment", "Neutral Good",
		p + "Comp.Use.AC", "12+Dex", p + "Comp.Use.HP.Max", "10", p + "Comp.Use.Speed", "15 ft, Fly 60 ft",
		p + "Comp.Use.Senses", "Blindsight 10ft, Darkvision 60 ft", p + "Comp.Use.Proficiency Bonus", "2",
		p + "Comp.Use.Ability.Dex.Score", "15", p + "Comp.Use.Ability.Dex.Mod", "2", p + "Comp.Use.Ability.Dex.ST.Mod", "2",
		p + "Comp.Use.Features", "##◆ Languages##. Understands Common.\r##◆ Magic Resistance##\r   Advantage on saves.",
		p + "Comp.Use.Traits", "##◆ Multiattack##. Two Bite attacks.",
		p + "Comp.eqp.Gear Row 1", "Silla de montar", p + "Comp.eqp.Gear Amount 1", "1",
	}
	d := docOf(append([]string{"Template.extras.AScomp", "," + p}, base...)...)
	d.withDisplay(p+"Comp.Desc.Size", "0.5", "Tiny")
	return d.with(extra...)
}

func TestMap_CompanionFields(t *testing.T) {
	d := companionPage("P4.AScomp.",
		"P4.AScomp.Comp.Use.Attack.1.Weapon", "Bite", "P4.AScomp.Comp.Use.Attack.1.Mod", "Comp.Use.Ability.Dex.Mod",
		"P4.AScomp.Comp.Use.Attack.1.To Hit", "+4", "P4.AScomp.Comp.Use.Attack.1.Damage", "1d4+2", "P4.AScomp.Comp.Use.Attack.1.Damage Type", "Piercing",
		"P4.AScomp.Comp.Use.Attack.2.Mod", "empty",
		"P4.AScomp.Comp.Use.Skills.Perc.Mod", "5", "P4.AScomp.Comp.Use.Skills.Acr.Mod", "2",
		"P4.AScomp.Text.Comp.Use.Skills.Perc.Prof", "expertise", "P4.AScomp.Text.Comp.Use.Skills.Acr.Prof", "nothing",
		"P4.AScomp.Cnote.Left", "**Pseudodragon** notes.",
		"PC Name", "Puck", "HP Max", "21",
	)

	res := Map(d)
	if len(res.Companions) != 1 {
		t.Fatalf("companions = %+v", res.Companions)
	}
	c := res.Companions[0]
	cr := c.Creature
	if c.Name != "Robin Gutierre'" || c.MaxHP != 10 {
		t.Errorf("name=%q maxHP=%d", c.Name, c.MaxHP)
	}
	if cr.Species != "Pseudodragon" || cr.Type != "Dragon" || cr.Size != "Tiny" || cr.Alignment != "Neutral Good" ||
		cr.AC != "12+Dex" || cr.HP != "10" || cr.SpeedText != "15 ft, Fly 60 ft" || cr.Senses != "Blindsight 10ft, Darkvision 60 ft" {
		t.Errorf("creature = %+v", cr)
	}
	if ptrInt(cr.Abilities.Dex.Score) != 15 || ptrInt(cr.Abilities.Dex.SaveBonus) != 2 || ptrInt(cr.ProficiencyBonus) != 2 {
		t.Errorf("abilities = %+v", cr.Abilities.Dex)
	}
	if len(cr.Attacks) != 1 || cr.Attacks[0].Name != "Bite" || cr.Attacks[0].Ability != "Dex" || cr.Attacks[0].ToHit != "+4" {
		t.Errorf("attacks = %+v, want one (the empty row skipped) with the ability read from the field path", cr.Attacks)
	}
	if len(cr.Skills) != 1 || cr.Skills[0].Key != "Perception" || !cr.Skills[0].Expertise || ptrInt(cr.Skills[0].Bonus) != 5 {
		t.Errorf("skills = %+v, want only the proficient one", cr.Skills)
	}
	if !strings.Contains(cr.Traits, "Magic Resistance") || !strings.Contains(cr.Traits, "Multiattack") || strings.ContainsAny(cr.Traits, "#◆") {
		t.Errorf("traits = %q", cr.Traits)
	}
	if len(cr.Gear) != 1 || cr.Gear[0].Name != "Silla de montar" || cr.Gear[0].Amount != "1" {
		t.Errorf("gear = %+v", cr.Gear)
	}
	if cr.Notes != "Pseudodragon notes." {
		t.Errorf("notes = %q", cr.Notes)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestMap_CompanionWithoutAName(t *testing.T) {
	d := docOf(
		"Template.extras.AScomp", ",P4.AScomp.",
		"P4.AScomp.Comp.Race", "Beast of the Burrow", "P4.AScomp.Comp.Use.HP.Max", "20",
	)

	c := Map(d).Companions
	if len(c) != 1 || c[0].Name != "Beast of the Burrow" || c[0].MaxHP != 20 {
		t.Errorf("companions = %+v, want the species as the name", c)
	}

	d = docOf("Template.extras.AScomp", ",P4.AScomp.", "P4.AScomp.Comp.Use.HP.Max", "8")
	if c := Map(d).Companions; len(c) != 1 || c[0].Name != "Companion" {
		t.Errorf("companions = %+v, want a placeholder name", c)
	}
}

func TestMap_CompanionWithoutNumericHPGetsAWarning(t *testing.T) {
	for _, hp := range []string{"", "lots", "0", "-3"} {
		d := docOf("Template.extras.AScomp", ",P4.AScomp.", "P4.AScomp.Comp.Race", "Wolf")
		if hp != "" {
			d.with("P4.AScomp.Comp.Use.HP.Max", hp)
		}
		res := Map(d)
		if len(res.Companions) != 1 || res.Companions[0].MaxHP != 0 {
			t.Errorf("hp %q: companions = %+v, want MaxHP 0 for the caller to default", hp, res.Companions)
		}
		found := false
		for _, w := range res.Warnings {
			found = found || (strings.Contains(w, "Wolf") && strings.Contains(w, "max HP"))
		}
		if !found {
			t.Errorf("hp %q: warnings = %v", hp, res.Warnings)
		}
	}
}

func TestMap_SeveralCompanionPagesAndUnusedOnesSkipped(t *testing.T) {
	d := docOf(
		"Template.extras.AScomp", ",P4.AScomp.,P9.AScomp.,P10.AScomp.",
		"P4.AScomp.Comp.Desc.Name", "Rex", "P4.AScomp.Comp.Use.HP.Max", "11",
		"P9.AScomp.Comp.Desc.Name", "Mia", "P9.AScomp.Comp.Use.HP.Max", "7",
		"P10.AScomp.Comp.Desc.Alignment", "Neutral", // a page with nothing that identifies a companion
	)

	c := Map(d).Companions
	if len(c) != 2 || c[0].Name != "Rex" || c[1].Name != "Mia" {
		t.Errorf("companions = %+v", c)
	}
}

func TestMap_NoCompanionPagesMeansNoCompanions(t *testing.T) {
	if c := Map(docOf("PC Name", "Puck")).Companions; len(c) != 0 {
		t.Errorf("companions = %+v", c)
	}
}

// --- wild shapes ---

func TestMap_WildShapesInfoAndForms(t *testing.T) {
	const p = "P6.WSfront."
	d := docOf(
		"Template.extras.WSfront", ","+p,
		p+"Wildshapes.Info.Uses", "2", p+"Wildshapes.Info.Duration", "1h", p+"Wildshapes.Info.Limitations", "No Fly Speed",
		p+"Wildshape.Race.1", "Rat", p+"Wildshape.Race.2", "Cat",
		p+"Wildshape.1.MonsterType", "Beast", p+"Wildshape.1.AC", "10", p+"Wildshape.1.HP Max", "1",
		p+"Wildshape.1.Speed", "20 ft, Climb 20 ft", p+"Wildshape.1.CR", "0", p+"Wildshape.1.HD", "1d4",
		p+"Wildshape.1.Ability.Str.Score", "2", p+"Wildshape.1.Ability.Str.Mod", "-4", p+"Wildshape.1.Ability.Wis.ST.Mod", "6", p+"Wildshape.1.Ability.Wis.ST.Prof", "/True",
		p+"Wildshape.1.Attack.1.Weapon", "Bite", p+"Wildshape.1.Attack.1.To Hit", "+2", p+"Wildshape.1.Attack.1.Damage", "1", p+"Wildshape.1.Attack.1.Damage Type", "Piercing",
		p+"Wildshape.1.Traits", "Agile: The rat doesn't provoke.",
		p+"Wildshape.1.Skills.Perc", "6", p+"Text.Wildshape.1.Skills.Perc.Prof", "proficient",
		p+"Wildshape.1.Skills.Ste", "2", p+"Text.Wildshape.1.Skills.Ste.Prof", "nothing",
		p+"Wildshape.2.AC", "12",
	).withDisplay(p+"Wildshape.1.Size", "0.5", "Tiny")

	ws := Map(d).Sheet.WildShapes
	if ws.Uses != "2" || ws.Duration != "1h" || ws.Limitations != "No Fly Speed" || len(ws.Forms) != 2 {
		t.Fatalf("wild shapes = %+v", ws)
	}
	rat := ws.Forms[0]
	if rat.Name != "Rat" || rat.Type != "Beast" || rat.Size != "Tiny" || rat.AC != "10" || rat.HP != "1" || rat.SpeedText != "20 ft, Climb 20 ft" {
		t.Errorf("rat = %+v", rat)
	}
	if ptrInt(rat.Abilities.Str.Score) != 2 || ptrInt(rat.Abilities.Wis.SaveBonus) != 6 || !rat.Abilities.Wis.SaveProficient {
		t.Errorf("rat abilities = %+v", rat.Abilities)
	}
	if len(rat.Attacks) != 1 || rat.Attacks[0].Name != "Bite" || len(rat.Skills) != 1 || rat.Skills[0].Key != "Perception" || ptrInt(rat.Skills[0].Bonus) != 6 {
		t.Errorf("rat attacks=%+v skills=%+v", rat.Attacks, rat.Skills)
	}
	if rat.Traits != "Agile: The rat doesn't provoke." || rat.Notes != "CR 0\nHit dice 1d4" {
		t.Errorf("rat traits=%q notes=%q", rat.Traits, rat.Notes)
	}
	if ws.Forms[1].Name != "Cat" || ws.Forms[1].AC != "12" {
		t.Errorf("cat = %+v", ws.Forms[1])
	}
}

func TestMap_NoWildShapePageMeansNone(t *testing.T) {
	if ws := Map(docOf("PC Name", "Puck")).Sheet.WildShapes; ws != nil {
		t.Errorf("wild shapes = %+v", ws)
	}
}

// --- inventory ---

func TestMap_InventoryFromBothGearListsAndCoins(t *testing.T) {
	inv := Map(docOf(
		"Adventuring Gear Row 1", "Rope (50 ft)", "Adventuring Gear Amount 1", "2",
		"Adventuring Gear Row 2", "Torch", "Adventuring Gear Amount 2", "many",
		"Adventuring Gear Row 3", "Healing Potion",
		"Extra.Gear Row 1", "Spare boots", "Extra.Gear Amount 1", "1",
		"Platinum Pieces", "1", "Gold Pieces", "25", "Electrum Pieces", "0", "Silver Pieces", "4", "Copper Pieces", "-7",
	)).Inventory

	if !inv.Present || len(inv.Items) != 4 {
		t.Fatalf("inventory = %+v", inv)
	}
	want := map[string]int{"Rope (50 ft)": 2, "Torch": 1, "Healing Potion": 1, "Spare boots": 1}
	for _, it := range inv.Items {
		if want[it.Name] != it.Quantity {
			t.Errorf("%s quantity = %d, want %d (a non-numeric or missing amount is 1)", it.Name, it.Quantity, want[it.Name])
		}
	}
	if c := inv.Currency; c.PP != 1 || c.GP != 25 || c.EP != 0 || c.SP != 4 || c.CP != 0 {
		t.Errorf("currency = %+v (a negative coin counts as 0)", c)
	}
}

func TestMap_InventoryPresence(t *testing.T) {
	if inv := Map(docOf("PC Name", "Puck")).Inventory; inv.Present || len(inv.Items) != 0 {
		t.Errorf("empty sheet: %+v, want not present", inv)
	}
	if inv := Map(docOf("Gold Pieces", "5")).Inventory; !inv.Present || len(inv.Items) != 0 {
		t.Errorf("coins only: %+v, want present", inv)
	}
	if inv := Map(docOf("Extra.Gear Row 1", "Dagger")).Inventory; !inv.Present {
		t.Errorf("one item: %+v, want present", inv)
	}
	if inv := Map(docOf("Gold Pieces", "0", "Copper Pieces", "0")).Inventory; inv.Present {
		t.Errorf("zero coins: %+v, want not present", inv)
	}
}
