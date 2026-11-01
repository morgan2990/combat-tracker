package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"combatapp/room"
	"combatapp/store"
)

// saveSection drives SaveSheetSection as the signed-in user.
func saveSection(t *testing.T, id, section string, revision int, dataJSON string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"revision":` + strconv.Itoa(revision) + `,"data":` + dataJSON + `}`
	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/pcs/"+id+"/sheet/"+section, []byte(body))
	r.SetPathValue("id", id)
	r.SetPathValue("section", section)
	SaveSheetSection(w, r)
	return w
}

func getSheet(t *testing.T, id string) (*httptest.ResponseRecorder, sheetResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/pcs/"+id+"/sheet", nil)
	r.SetPathValue("id", id)
	GetSheet(w, r)
	var resp sheetResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decoding sheet response: %v", err)
		}
	}
	return w, resp
}

func revisionOf(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var out struct{ Revision int }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding revision: %v (body %q)", err, w.Body.String())
	}
	return out.Revision
}

// sheetFixture wires fakes for one owner with a PC (and optionally more PCs).
func sheetFixture(t *testing.T, pcs ...store.PC) (*fakePCStore, *fakeSheetStore) {
	t.Helper()
	pcFake := newFakePCStore(pcs...)
	sheetFake := newFakeSheetStore()
	withFakePCStore(t, pcFake)
	withFakeSheetStore(t, sheetFake)
	withFakeAuth(t, "owner1")
	return pcFake, sheetFake
}

var aria = store.PC{
	ID: "pc1", OwnerUserID: "owner1", Name: "Aria", Type: "pc", MaxHP: 16,
	Items: []store.Item{{Name: "Rope", Quantity: 2}}, Currency: store.Currency{GP: 25},
}

// --- reading ---

func TestGetSheet_EmptyWhenNoneStored(t *testing.T) {
	_, sheets := sheetFixture(t, aria)

	w, resp := getSheet(t, "pc1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if resp.Sheet.Revision != 0 || resp.PC.ID != "pc1" || len(resp.Companions) != 0 {
		t.Errorf("response = %+v, want an empty revision-0 sheet and no companions", resp)
	}
	if len(sheets.byPC) != 0 {
		t.Errorf("GET created a document: %+v", sheets.byPC)
	}
}

func TestGetSheet_ReturnsCompanionsWithTheirSheets(t *testing.T) {
	_, sheets := sheetFixture(t, aria, store.PC{ID: "c1", OwnerUserID: "owner1", Name: "Rex", Type: "companion", ParentPCID: "pc1", MaxHP: 8})
	sheets.byPC["c1"] = &store.CharacterSheet{PCID: "c1", Revision: 2, Creature: &store.Creature{AC: "13+Prof"}}

	_, resp := getSheet(t, "pc1")

	if len(resp.Companions) != 1 || resp.Companions[0].PC.ID != "c1" {
		t.Fatalf("companions = %+v", resp.Companions)
	}
	if got := resp.Companions[0].Sheet; got.Revision != 2 || got.Creature == nil || got.Creature.AC != "13+Prof" {
		t.Errorf("companion sheet = %+v", got)
	}
}

func TestGetSheet_NotOwnerIsNotFound(t *testing.T) {
	sheetFixture(t, store.PC{ID: "pc9", OwnerUserID: "someone-else", Type: "pc"})

	w, _ := getSheet(t, "pc9")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetSheet_RequiresAuth(t *testing.T) {
	withFakePCStore(t, newFakePCStore(aria))
	withFakeSheetStore(t, newFakeSheetStore())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/pcs/pc1/sheet", nil)
	r.SetPathValue("id", "pc1")

	GetSheet(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestSaveSheetSection_RequiresAuth(t *testing.T) {
	withFakePCStore(t, newFakePCStore(aria))
	sheets := newFakeSheetStore()
	withFakeSheetStore(t, sheets)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/pcs/pc1/sheet/notes", strings.NewReader(`{"revision":0,"data":{}}`))
	r.SetPathValue("id", "pc1")
	r.SetPathValue("section", "notes")

	SaveSheetSection(w, r)

	if w.Code != http.StatusUnauthorized || len(sheets.byPC) != 0 {
		t.Errorf("status = %d (sheets %d), want 401 and no change", w.Code, len(sheets.byPC))
	}
}

// --- saving ---

func TestSaveSheetSection_FirstSaveCreatesAtRevisionOne(t *testing.T) {
	sheetFixture(t, aria)

	w := saveSection(t, "pc1", "notes", 0, `{"personality":"Curious"}`)

	if w.Code != http.StatusOK || revisionOf(t, w) != 1 {
		t.Fatalf("status = %d, body %s; want 200 with revision 1", w.Code, w.Body.String())
	}
}

func TestSaveSheetSection_OtherSectionsStayIntact(t *testing.T) {
	sheetFixture(t, aria)
	rev := revisionOf(t, saveSection(t, "pc1", "notes", 0, `{"flaws":"Reckless"}`))

	w := saveSection(t, "pc1", "attacks", rev, `[{"name":"Club","to_hit":"+1"}]`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	_, resp := getSheet(t, "pc1")
	if resp.Sheet.Notes == nil || resp.Sheet.Notes.Flaws != "Reckless" || len(resp.Sheet.Attacks) != 1 {
		t.Errorf("sheet = %+v, want notes preserved and one attack", resp.Sheet)
	}
}

func TestSaveSheetSection_StaleRevisionConflicts(t *testing.T) {
	sheetFixture(t, aria)
	rev := revisionOf(t, saveSection(t, "pc1", "notes", 0, `{}`))
	saveSection(t, "pc1", "attacks", rev, `[{"name":"A"}]`)

	w := saveSection(t, "pc1", "attacks", rev, `[{"name":"B"}]`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	_, resp := getSheet(t, "pc1")
	if resp.Sheet.Attacks[0].Name != "A" {
		t.Errorf("stale save changed the sheet: %+v", resp.Sheet.Attacks)
	}
}

func TestSaveSheetSection_UnknownSectionRejected(t *testing.T) {
	_, sheets := sheetFixture(t, aria)

	w := saveSection(t, "pc1", "unknown", 0, `{}`)

	if w.Code != http.StatusBadRequest || len(sheets.byPC) != 0 {
		t.Errorf("status = %d (sheets %d), want 400 and no change", w.Code, len(sheets.byPC))
	}
}

func TestSaveSheetSection_NonOwnerForbidden(t *testing.T) {
	_, sheets := sheetFixture(t, store.PC{ID: "pc9", OwnerUserID: "someone-else", Type: "pc"})

	w := saveSection(t, "pc9", "notes", 0, `{}`)

	if w.Code != http.StatusForbidden || len(sheets.byPC) != 0 {
		t.Errorf("status = %d (sheets %d), want 403 and no change", w.Code, len(sheets.byPC))
	}
}

func TestSaveSheetSection_NullAndMalformedRejected(t *testing.T) {
	_, sheets := sheetFixture(t, aria)

	for _, data := range []string{`null`, `"text"`} {
		if w := saveSection(t, "pc1", "attacks", 0, data); w.Code != http.StatusBadRequest {
			t.Errorf("data %s: status = %d, want 400", data, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r := authedRequest(http.MethodPut, "/api/pcs/pc1/sheet/notes", []byte(`{not json`))
	r.SetPathValue("id", "pc1")
	r.SetPathValue("section", "notes")
	SaveSheetSection(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", w.Code)
	}
	if len(sheets.byPC) != 0 {
		t.Errorf("rejected saves created a sheet")
	}
}

func TestSaveSheetSection_OversizedBodyRejected(t *testing.T) {
	_, sheets := sheetFixture(t, aria)
	huge := `{"free_text":"` + strings.Repeat("x", maxSheetBody) + `"}`

	w := saveSection(t, "pc1", "notes", 0, huge)

	if w.Code != http.StatusRequestEntityTooLarge || len(sheets.byPC) != 0 {
		t.Errorf("status = %d (sheets %d), want 413 and no change", w.Code, len(sheets.byPC))
	}
}

func TestSaveSheetSection_StoreErrorIs500(t *testing.T) {
	_, sheets := sheetFixture(t, aria)
	sheets.forceErr = http.ErrAbortHandler

	if w := saveSection(t, "pc1", "notes", 0, `{}`); w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- identity and combat write through to the PC ---

func TestSaveSheetSection_IdentityUpdatesPCName(t *testing.T) {
	pcs, sheets := sheetFixture(t, aria)

	w := saveSection(t, "pc1", "identity", 0, `{"name":"Aria Sunsong","species":"Eladrin","classes":[{"class":"Warlock","subclass":"Archfey","level":1},{"class":"Druid","subclass":"Dreams","level":2}]}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got, _ := pcs.GetPCByID("pc1")
	if got.Name != "Aria Sunsong" || got.MaxHP != 16 || len(got.Items) != 1 || got.Currency.GP != 25 {
		t.Errorf("PC = %+v, want a new name and everything else untouched", got)
	}
	stored := sheets.byPC["pc1"].Identity
	if stored.Name != "" || len(stored.Classes) != 2 || stored.Classes[1].Subclass != "Dreams" {
		t.Errorf("stored identity = %+v, want no name and both classes in order", stored)
	}
}

func TestSaveSheetSection_ClassTextWithoutParsedClasses(t *testing.T) {
	sheetFixture(t, aria)

	w := saveSection(t, "pc1", "identity", 0, `{"name":"Aria","classes_text":"Beast Master"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	_, resp := getSheet(t, "pc1")
	if resp.Sheet.Identity.ClassesText != "Beast Master" || len(resp.Sheet.Identity.Classes) != 0 {
		t.Errorf("identity = %+v", resp.Sheet.Identity)
	}
}

func TestSaveSheetSection_CombatUpdatesPCMaxHP(t *testing.T) {
	pcs, _ := sheetFixture(t, aria)

	w := saveSection(t, "pc1", "combat", 0, `{"max_hp":21,"ac":13,"speed_text":"30 ft"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got, _ := pcs.GetPCByID("pc1")
	if got.MaxHP != 21 || got.Name != "Aria" || len(got.Items) != 1 || got.Currency.GP != 25 {
		t.Errorf("PC = %+v, want a new max_hp and everything else untouched", got)
	}
}

func TestSaveSheetSection_InvalidNameOrMaxHPRejected(t *testing.T) {
	pcs, sheets := sheetFixture(t, aria)

	for _, c := range []struct{ section, data string }{
		{"identity", `{"name":""}`},
		{"identity", `{"name":"   "}`},
		{"identity", `{"species":"Elf"}`},
		{"combat", `{"max_hp":0}`},
		{"combat", `{"max_hp":-3}`},
		{"combat", `{"ac":13}`},
	} {
		if w := saveSection(t, "pc1", c.section, 0, c.data); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400", c.section, c.data, w.Code)
		}
	}
	got, _ := pcs.GetPCByID("pc1")
	if got.Name != "Aria" || got.MaxHP != 16 || len(sheets.byPC) != 0 {
		t.Errorf("rejected saves changed state: pc=%+v sheets=%d", got, len(sheets.byPC))
	}
}

// --- free text, blanks, and per-section content ---

func TestSaveSheetSection_FreeTextAndBlankValuesRoundTrip(t *testing.T) {
	sheetFixture(t, aria)
	rev := revisionOf(t, saveSection(t, "pc1", "limited_uses", 0, `[{"name":"Balm","max":"2d6","recovery":"LR/SS","source":"Ring of Jaedear"}]`))
	rev = revisionOf(t, saveSection(t, "pc1", "attacks", rev, `[{"name":"Sting","to_hit":"DC 12","damage":"2d4+2"}]`))
	saveSection(t, "pc1", "combat", rev, `{"max_hp":16,"ac":null}`)

	_, resp := getSheet(t, "pc1")

	if resp.Sheet.LimitedUses[0].Max != "2d6" || resp.Sheet.LimitedUses[0].Recovery != "LR/SS" || resp.Sheet.Attacks[0].ToHit != "DC 12" {
		t.Errorf("free-text values changed: %+v %+v", resp.Sheet.LimitedUses, resp.Sheet.Attacks)
	}
	if resp.Sheet.Combat.AC != nil {
		t.Errorf("AC = %v, want nil (blank, not zero)", *resp.Sheet.Combat.AC)
	}
}

func TestSaveSheetSection_ModifierIsStoredAsSent(t *testing.T) {
	sheetFixture(t, aria)

	saveSection(t, "pc1", "abilities", 0, `{"str":{"score":18,"mod":-1,"save_bonus":5,"save_proficient":true}}`)

	_, resp := getSheet(t, "pc1")
	str := resp.Sheet.Abilities.Str
	if *str.Score != 18 || *str.Mod != -1 || *str.SaveBonus != 5 || !str.SaveProficient {
		t.Errorf("str = %+v, want values exactly as sent (no recalculation)", str)
	}
}

func TestSaveSheetSection_SectionContentRoundTrips(t *testing.T) {
	sheetFixture(t, aria)
	rev := 0
	save := func(section, data string) {
		t.Helper()
		w := saveSection(t, "pc1", section, rev, data)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body %s", section, w.Code, w.Body.String())
		}
		rev = revisionOf(t, w)
	}
	save("actions", `{"bonus_action":[{"name":"Tomá wacho!","source":"Kid Tricks (feat)","is_default":false}]}`)
	save("limited_uses", `[{"name":"Shield","max":"1","recovery":"dawn","source":"Ring of Jaedear"}]`)
	save("magic_items", `[{"name":"Ring of Jaedear","stage":"Level 1","attunement_required":true,"attuned":true}]`)
	save("proficiencies", `{"languages":[{"name":"Sylvan","pending":false},{"name":"from Human","pending":true}]}`)
	save("notes", `{"personality":"Curious","ideals":"Kindness","bonds":"The party","flaws":"Reckless"}`)
	save("wild_shapes", `{"uses":"2","limitations":"No Fly Speed","forms":[{"name":"Rat","ac":"10"},{"name":"Wolf","ac":"12"}]}`)

	_, resp := getSheet(t, "pc1")
	s := resp.Sheet

	if a := s.Actions.BonusAction[0]; a.Source != "Kid Tricks (feat)" {
		t.Errorf("action source = %q", a.Source)
	}
	if u := s.LimitedUses[0]; u.Recovery != "dawn" || u.Source != "Ring of Jaedear" {
		t.Errorf("limited use = %+v", u)
	}
	if m := s.MagicItems[0]; m.Name != "Ring of Jaedear" || m.Stage != "Level 1" {
		t.Errorf("magic item = %+v", m)
	}
	if l := s.Proficiencies.Languages; len(l) != 2 || l[0].Pending || !l[1].Pending || l[1].Name != "from Human" {
		t.Errorf("languages = %+v", l)
	}
	if n := s.Notes; n.Personality == "" || n.Ideals == "" || n.Bonds == "" || n.Flaws == "" {
		t.Errorf("notes = %+v, want all four personality fields", n)
	}
	if w := s.WildShapes; len(w.Forms) != 2 || w.Forms[0].Name != "Rat" || w.Limitations != "No Fly Speed" {
		t.Errorf("wild shapes = %+v", w)
	}
}

// --- spellcasting ---

func TestSaveSheetSection_SpellcastingKeepsListsPreparationAndPactMagicApart(t *testing.T) {
	sheetFixture(t, aria)
	data := `{
		"spell_lists":[
			{"kind":"class","name":"Druid","ability":"Wis","save_dc":"14","attack":"+6","spells":[
				{"name":"Find Familiar","level":1,"preparation":"always"},
				{"name":"Healing Word","level":1,"preparation":"prepared"}]},
			{"kind":"feat","name":"Magic Initiate (Druid)","spells":[{"name":"Cure Wounds","level":1,"preparation":"limited"}]},
			{"kind":"item","name":"Ring of Jaedear","spells":[{"name":"Shield","level":1,"preparation":"limited"}]}],
		"slots":[3],
		"pact_magic":{"slot_level":1,"total":1}}`

	w := saveSection(t, "pc1", "spellcasting", 0, data)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	_, resp := getSheet(t, "pc1")
	sc := resp.Sheet.Spellcasting
	if len(sc.SpellLists) != 3 || sc.SpellLists[0].Kind != "class" || sc.SpellLists[1].Kind != "feat" || sc.SpellLists[2].Kind != "item" {
		t.Fatalf("spell lists = %+v", sc.SpellLists)
	}
	if sc.SpellLists[0].SaveDC != "14" || sc.SpellLists[0].Attack != "+6" {
		t.Errorf("first list = %+v", sc.SpellLists[0])
	}
	if sp := sc.SpellLists[0].Spells; sp[0].Preparation != "always" || sp[1].Preparation != "prepared" {
		t.Errorf("preparation = %q / %q, want always / prepared", sp[0].Preparation, sp[1].Preparation)
	}
	if sc.Slots[0] != 3 || sc.PactMagic == nil || sc.PactMagic.Total != 1 {
		t.Errorf("slots = %v, pact = %+v, want 3 regular and 1 pact slot kept separate", sc.Slots, sc.PactMagic)
	}
}

func TestSaveSheetSection_SpellcastingValidation(t *testing.T) {
	_, sheets := sheetFixture(t, aria)

	for name, data := range map[string]string{
		"unlisted preparation":     `{"spell_lists":[{"kind":"class","name":"X","spells":[{"name":"S","level":1,"preparation":"sometimes"}]}]}`,
		"empty preparation":        `{"spell_lists":[{"kind":"class","name":"X","spells":[{"name":"S","level":1}]}]}`,
		"level 10":                 `{"spell_lists":[{"kind":"class","name":"X","spells":[{"name":"S","level":10,"preparation":"known"}]}]}`,
		"negative level":           `{"spell_lists":[{"kind":"class","name":"X","spells":[{"name":"S","level":-1,"preparation":"known"}]}]}`,
		"unlisted list kind":       `{"spell_lists":[{"kind":"book","name":"X"}]}`,
		"ten slot levels":          `{"slots":[1,1,1,1,1,1,1,1,1,1]}`,
		"negative slot total":      `{"slots":[-1]}`,
		"pact slot level too high": `{"pact_magic":{"slot_level":10,"total":1}}`,
	} {
		if w := saveSection(t, "pc1", "spellcasting", 0, data); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, w.Code)
		}
	}
	if len(sheets.byPC) != 0 {
		t.Errorf("rejected saves created a sheet")
	}
}

func TestSaveSheetSection_FeatureKindValidated(t *testing.T) {
	sheetFixture(t, aria)

	if w := saveSection(t, "pc1", "features", 0, `[{"kind":"feat","name":"Alert","source":"PHB","level":1}]`); w.Code != http.StatusOK {
		t.Errorf("valid feature: status = %d, want 200", w.Code)
	}
	if w := saveSection(t, "pc1", "features", 1, `[{"kind":"hobby","name":"Chess"}]`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid kind: status = %d, want 400", w.Code)
	}
}

// --- companions ---

var rex = store.PC{ID: "c1", OwnerUserID: "owner1", Name: "Rex", Type: "companion", ParentPCID: "pc1", MaxHP: 8, SharesInitiative: true}

func TestSaveSheetSection_CompanionCreatureKeepsFreeTextValues(t *testing.T) {
	pcs, _ := sheetFixture(t, aria, rex)

	w := saveSection(t, "c1", "creature", 0, `{"ac":"13+Prof","speed_text":"40 ft, climb 30 ft, burrow 15 ft","species":"Beast of the Burrow"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	_, resp := getSheet(t, "pc1")
	c := resp.Companions[0].Sheet.Creature
	if c.AC != "13+Prof" || c.SpeedText != "40 ft, climb 30 ft, burrow 15 ft" {
		t.Errorf("creature = %+v", c)
	}
	if got, _ := pcs.GetPCByID("c1"); !got.SharesInitiative || got.MaxHP != 8 {
		t.Errorf("companion PC changed: %+v", got)
	}
}

func TestSaveSheetSection_SectionRulesByCharacterType(t *testing.T) {
	_, sheets := sheetFixture(t, aria, rex)

	if w := saveSection(t, "c1", "spellcasting", 0, `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("spellcasting on a companion: status = %d, want 400", w.Code)
	}
	if w := saveSection(t, "c1", "identity", 0, `{"name":"Rex"}`); w.Code != http.StatusBadRequest {
		t.Errorf("identity on a companion: status = %d, want 400", w.Code)
	}
	if w := saveSection(t, "pc1", "creature", 0, `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("creature on a PC: status = %d, want 400", w.Code)
	}
	if len(sheets.byPC) != 0 {
		t.Errorf("rejected saves created a sheet")
	}
}

// --- source ---

func TestSaveSheetSection_SourceIsPreserved(t *testing.T) {
	_, sheets := sheetFixture(t, aria)
	sheets.byPC["pc1"] = &store.CharacterSheet{PCID: "pc1", Revision: 1, SchemaVersion: store.SheetSchemaVersion,
		Source: &store.SheetSource{Kind: "mpmb", SheetVersion: "14.0.8"}}

	if w := saveSection(t, "pc1", "notes", 1, `{"free_text":"x"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	_, resp := getSheet(t, "pc1")
	if resp.Sheet.Source == nil || resp.Sheet.Source.Kind != "mpmb" || resp.Sheet.Source.SheetVersion != "14.0.8" {
		t.Errorf("source = %+v, want it preserved", resp.Sheet.Source)
	}
}

func TestSaveSheetSection_ManualSheetHasSchemaVersionAndNoSource(t *testing.T) {
	sheetFixture(t, aria)
	saveSection(t, "pc1", "notes", 0, `{}`)

	_, resp := getSheet(t, "pc1")

	if resp.Sheet.SchemaVersion != store.SheetSchemaVersion || resp.Sheet.Source != nil {
		t.Errorf("sheet = %+v, want schema_version set and no source", resp.Sheet)
	}
}

// --- deletion ---

func deleteAs(t *testing.T, id string) {
	t.Helper()
	w := httptest.NewRecorder()
	r := authedRequest(http.MethodDelete, "/api/pcs/"+id, nil)
	r.SetPathValue("id", id)
	DeletePC(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("DeletePC status = %d", w.Code)
	}
}

func TestDeletePC_RemovesItsAndItsCompanionsSheets(t *testing.T) {
	_, sheets := sheetFixture(t, aria, rex)
	sheets.byPC["pc1"] = &store.CharacterSheet{PCID: "pc1", Revision: 1}
	sheets.byPC["c1"] = &store.CharacterSheet{PCID: "c1", Revision: 1}
	sheets.byPC["other"] = &store.CharacterSheet{PCID: "other", Revision: 1}

	deleteAs(t, "pc1")

	if len(sheets.byPC) != 1 || sheets.byPC["other"] == nil {
		t.Errorf("sheets after delete = %v, want only the unrelated one", sheets.byPC)
	}
}

func TestDeleteCompanion_KeepsParentSheet(t *testing.T) {
	_, sheets := sheetFixture(t, aria, rex)
	sheets.byPC["pc1"] = &store.CharacterSheet{PCID: "pc1", Revision: 1}
	sheets.byPC["c1"] = &store.CharacterSheet{PCID: "c1", Revision: 1}

	deleteAs(t, "c1")

	if sheets.byPC["c1"] != nil || sheets.byPC["pc1"] == nil {
		t.Errorf("sheets after deleting the companion = %v, want only the PC's", sheets.byPC)
	}
}

func TestDeletePC_SheetCleanupFailureStillSucceeds(t *testing.T) {
	_, sheets := sheetFixture(t, aria)
	sheets.forceErr = http.ErrAbortHandler

	deleteAs(t, "pc1") // fails the test itself unless the status is 200
}

// --- the DM route ---

func dmRequest(t *testing.T, roomID, pcID string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := authedRequest(http.MethodGet, "/api/rooms/"+roomID+"/pcs/"+pcID+"/sheet", nil)
	r.SetPathValue("room_id", roomID)
	r.SetPathValue("pc_id", pcID)
	GetRoomPCSheet(w, r)
	return w
}

// dmFixture creates a room owned by "dm1" whose linked party contains pc1
// (owned by "owner1"), with a companion and one stored sheet.
func dmFixture(t *testing.T) string {
	t.Helper()
	roomID := room.Global.CreateRoom("5e", "dm1")
	rm, _ := room.Global.GetRoom(roomID)
	rm.ClaimParty("party1")
	withFakePartyStore(t, newFakePartyStore(store.Party{ID: "party1", MemberPCIDs: []string{"pc1"}}))
	withFakePCStore(t, newFakePCStore(aria, rex, store.PC{ID: "pc2", OwnerUserID: "owner2", Name: "Outsider", Type: "pc", MaxHP: 9}))
	withFakeSheetStore(t, newFakeSheetStore(
		store.CharacterSheet{PCID: "pc1", Revision: 3, Notes: &store.SheetNotes{Flaws: "Reckless"}},
		store.CharacterSheet{PCID: "c1", Revision: 1, Creature: &store.Creature{AC: "12"}},
	))
	withFakeAuth(t, "dm1")
	return roomID
}

func TestGetRoomPCSheet_DMReadsPartyMemberAndCompanions(t *testing.T) {
	roomID := dmFixture(t)

	w := dmRequest(t, roomID, "pc1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp sheetResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Sheet.Notes == nil || resp.Sheet.Notes.Flaws != "Reckless" {
		t.Errorf("sheet = %+v", resp.Sheet)
	}
	if len(resp.Companions) != 1 || resp.Companions[0].Sheet.Creature.AC != "12" {
		t.Errorf("companions = %+v", resp.Companions)
	}
}

func TestGetRoomPCSheet_PCOutsideThePartyForbidden(t *testing.T) {
	roomID := dmFixture(t)

	if w := dmRequest(t, roomID, "pc2"); w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestGetRoomPCSheet_RoomWithoutPartyForbidden(t *testing.T) {
	dmFixture(t)
	roomID := room.Global.CreateRoom("5e", "dm1") // never claims a party

	if w := dmRequest(t, roomID, "pc1"); w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestGetRoomPCSheet_NonOwnerOfRoomForbidden(t *testing.T) {
	roomID := dmFixture(t)
	withFakeAuth(t, "owner1") // the PC's own player, but not the room's DM

	if w := dmRequest(t, roomID, "pc1"); w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestGetRoomPCSheet_RequiresAuth(t *testing.T) {
	roomID := dmFixture(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/pcs/pc1/sheet", nil)
	r.SetPathValue("room_id", roomID)
	r.SetPathValue("pc_id", "pc1")

	GetRoomPCSheet(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
