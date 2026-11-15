package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"combatapp/mpmb/mpmbtest"
	"combatapp/store"
)

type importEnv struct {
	pcs     *fakePCStore
	sheets  *fakeSheetStore
	imports *fakeImportStore
}

func setupImport(t *testing.T, userID string, pcs ...store.PC) importEnv {
	t.Helper()
	env := importEnv{pcs: newFakePCStore(pcs...), sheets: newFakeSheetStore(), imports: newFakeImportStore()}
	withFakePCStore(t, env.pcs)
	withFakeSheetStore(t, env.sheets)
	withFakeImportStore(t, env.imports)
	withFakeAuth(t, userID)
	return env
}

func pdfFields(kv ...string) []mpmbtest.Field {
	var out []mpmbtest.Field
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, mpmbtest.Field{Name: kv[i], Value: kv[i+1]})
	}
	return out
}

func samplePDF(extra ...string) []byte {
	kv := append([]string{"PC Name", "Puck", "HP Max", "21", "Attack.1.Weapon Selection", "Club", "Attack.1.To Hit", "+4"}, extra...)
	return mpmbtest.Build(mpmbtest.Doc{Fields: pdfFields(kv...), Info: mpmbtest.MPMBInfo()})
}

func uploadRequest(t *testing.T, target string, file []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "puck.pdf")
	fw.Write(file)
	mw.Close()
	r := authedRequest(http.MethodPost, target, buf.Bytes())
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func serve(h http.HandlerFunc, r *http.Request, pathValues map[string]string) *httptest.ResponseRecorder {
	for k, v := range pathValues {
		r.SetPathValue(k, v)
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func uploadFor(t *testing.T, pcID string, file []byte) *httptest.ResponseRecorder {
	return serve(UploadSheetImport, uploadRequest(t, "/api/pcs/"+pcID+"/sheet/import", file), map[string]string{"id": pcID})
}

func uploadNew(t *testing.T, file []byte) *httptest.ResponseRecorder {
	return serve(UploadNewCharacterImport, uploadRequest(t, "/api/sheet-imports", file), nil)
}

func confirm(id string) *httptest.ResponseRecorder {
	return serve(ConfirmSheetImport, authedRequest(http.MethodPost, "/api/sheet-imports/"+id+"/confirm", nil), map[string]string{"id": id})
}

func discard(id string) *httptest.ResponseRecorder {
	return serve(DiscardSheetImport, authedRequest(http.MethodDelete, "/api/sheet-imports/"+id, nil), map[string]string{"id": id})
}

func previewOf(t *testing.T, w *httptest.ResponseRecorder) importPreview {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var p importPreview
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func ownedPC(id string) store.PC {
	return store.PC{ID: id, OwnerUserID: "u1", Type: "pc", Name: "Old", MaxHP: 10}
}

func TestUploadSheetImport_ReturnsPreviewAndChangesNothing(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))

	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))

	if p.ImportID == "" || p.NewCharacter || p.Name != "Puck" || p.MaxHP != 21 {
		t.Errorf("preview = %+v", p)
	}
	if len(p.Changes) == 0 {
		t.Error("expected differences")
	}
	if len(env.sheets.byPC) != 0 {
		t.Error("an upload must not store a sheet")
	}
	if pc, _ := env.pcs.GetPCByID("pc1"); pc.Name != "Old" || pc.MaxHP != 10 {
		t.Errorf("pc changed: %+v", pc)
	}
	if len(env.imports.byID) != 1 {
		t.Errorf("drafts = %d, want 1", len(env.imports.byID))
	}
}

func TestUploadSheetImport_Rejections(t *testing.T) {
	env := setupImport(t, "u1",
		ownedPC("pc1"),
		store.PC{ID: "other", OwnerUserID: "u2", Type: "pc", Name: "B", MaxHP: 5},
		store.PC{ID: "comp", OwnerUserID: "u1", Type: "companion", ParentPCID: "pc1", Name: "C", MaxHP: 5},
	)
	flat := mpmbtest.Build(mpmbtest.Doc{Fields: pdfFields("PC Name", "x"), Info: mpmbtest.MPMBInfo(), NoForm: true})
	notMPMB := mpmbtest.Build(mpmbtest.Doc{Fields: pdfFields("PC Name", "x"), Info: map[string]string{"Title": "other"}})

	tests := []struct {
		name string
		pc   string
		file []byte
		want int
	}{
		{"someone else's character", "other", samplePDF(), http.StatusForbidden},
		{"companion", "comp", samplePDF(), http.StatusBadRequest},
		{"missing character", "nope", samplePDF(), http.StatusNotFound},
		{"flattened", "pc1", flat, http.StatusBadRequest},
		{"not an MPMB sheet", "pc1", notMPMB, http.StatusBadRequest},
		{"corrupt", "pc1", []byte("not a pdf at all"), http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if w := uploadFor(t, tc.pc, tc.file); w.Code != tc.want {
				t.Errorf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
			if len(env.imports.byID) != 0 {
				t.Error("a rejected upload must store nothing")
			}
		})
	}
}

func TestUploadSheetImport_FlattenedMessage(t *testing.T) {
	setupImport(t, "u1", ownedPC("pc1"))
	flat := mpmbtest.Build(mpmbtest.Doc{Fields: pdfFields("PC Name", "x"), Info: mpmbtest.MPMBInfo(), NoForm: true})
	if w := uploadFor(t, "pc1", flat); !bytes.Contains(w.Body.Bytes(), []byte("flatten")) {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestUploadSheetImport_Unauthenticated(t *testing.T) {
	setupImport(t, "u1")
	r := uploadRequest(t, "/api/pcs/pc1/sheet/import", samplePDF())
	r.Header.Del("Cookie")
	if w := serve(UploadSheetImport, r, map[string]string{"id": "pc1"}); w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d", w.Code)
	}
}

func TestUploadSheetImport_TooLarge(t *testing.T) {
	setupImport(t, "u1", ownedPC("pc1"))
	old := maxImportUpload
	maxImportUpload = 100
	t.Cleanup(func() { maxImportUpload = old })
	if w := uploadFor(t, "pc1", samplePDF()); w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d", w.Code)
	}
}

func TestUploadSheetImport_ConcurrentImportGets429(t *testing.T) {
	setupImport(t, "u1", ownedPC("pc1"))
	importSlot <- struct{}{}
	t.Cleanup(func() { <-importSlot })
	if w := uploadFor(t, "pc1", samplePDF()); w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d", w.Code)
	}
}

func TestUploadNewCharacterImport_CreatesNothing(t *testing.T) {
	env := setupImport(t, "u1")
	p := previewOf(t, uploadNew(t, samplePDF()))
	if !p.NewCharacter || p.ImportID == "" || p.Name != "Puck" {
		t.Errorf("preview = %+v", p)
	}
	if len(env.pcs.byID) != 0 || len(env.sheets.byPC) != 0 {
		t.Error("an upload must not create a character")
	}
}

func TestConfirm_ReplacesSheetAndUpdatesPC(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))
	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))

	if w := confirm(p.ImportID); w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	sh := env.sheets.byPC["pc1"]
	if sh == nil || sh.Revision != 1 || len(sh.Attacks) != 1 {
		t.Fatalf("sheet = %+v", sh)
	}
	src := sh.Source
	if src == nil || src.Kind != "mpmb" || src.ImportID != p.ImportID || src.FileName != "puck.pdf" || src.FileSHA256 == "" || src.ImportedAt == nil {
		t.Errorf("source = %+v", src)
	}
	if pc, _ := env.pcs.GetPCByID("pc1"); pc.Name != "Puck" || pc.MaxHP != 21 {
		t.Errorf("pc = %+v", pc)
	}
	if env.imports.byID[p.ImportID].Status != store.ImportApplied {
		t.Error("import not marked applied")
	}
}

func TestConfirm_Twice409(t *testing.T) {
	setupImport(t, "u1", ownedPC("pc1"))
	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))
	confirm(p.ImportID)
	if w := confirm(p.ImportID); w.Code != http.StatusConflict {
		t.Errorf("status = %d", w.Code)
	}
}

func TestConfirm_SheetChangedAfterUploadIs409AndChangesNothing(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))
	env.sheets.byPC["pc1"] = &store.CharacterSheet{PCID: "pc1", Revision: 1}
	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))
	env.sheets.byPC["pc1"].Revision = 2 // saved after the upload

	if w := confirm(p.ImportID); w.Code != http.StatusConflict {
		t.Fatalf("status = %d", w.Code)
	}
	if sh := env.sheets.byPC["pc1"]; sh.Revision != 2 || len(sh.Attacks) != 0 {
		t.Errorf("sheet changed: %+v", sh)
	}
	if pc, _ := env.pcs.GetPCByID("pc1"); pc.Name != "Old" {
		t.Errorf("pc changed: %+v", pc)
	}
	if env.imports.byID[p.ImportID].Status != store.ImportDraft {
		t.Error("a failed confirm should leave the draft")
	}
}

func TestConfirm_InvalidHPKeepsExistingMaxHP(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))
	pdf := mpmbtest.Build(mpmbtest.Doc{Fields: pdfFields("PC Name", "Puck", "HP Max", "lots"), Info: mpmbtest.MPMBInfo()})
	p := previewOf(t, uploadFor(t, "pc1", pdf))
	if len(p.Warnings) == 0 {
		t.Error("expected a warning")
	}
	confirm(p.ImportID)
	if pc, _ := env.pcs.GetPCByID("pc1"); pc.MaxHP != 10 {
		t.Errorf("max_hp = %d", pc.MaxHP)
	}
}

func TestConfirm_InventoryKeptWhenPDFHasNone(t *testing.T) {
	pc := ownedPC("pc1")
	pc.Items = []store.Item{{Name: "Rope", Quantity: 1}}
	pc.Currency = store.Currency{GP: 5}
	env := setupImport(t, "u1", pc)

	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))
	if p.Inventory.Action != "keep" {
		t.Errorf("inventory = %+v", p.Inventory)
	}
	confirm(p.ImportID)
	got, _ := env.pcs.GetPCByID("pc1")
	if len(got.Items) != 1 || got.Currency.GP != 5 {
		t.Errorf("pc = %+v", got)
	}
}

func TestConfirm_InventoryReplacedKeepingCatalogLinks(t *testing.T) {
	pc := ownedPC("pc1")
	pc.Items = []store.Item{{Name: "Rope", Quantity: 1, CatalogItemID: "cat1", ReferenceURL: "http://x"}, {Name: "Gone", Quantity: 1}}
	env := setupImport(t, "u1", pc)

	pdf := samplePDF("Adventuring Gear Row 1", "Rope", "Adventuring Gear Amount 1", "two",
		"Adventuring Gear Row 2", "Torch", "Adventuring Gear Amount 2", "3", "Gold Pieces", "7")
	p := previewOf(t, uploadFor(t, "pc1", pdf))
	if p.Inventory.Action != "replace" {
		t.Fatalf("inventory = %+v", p.Inventory)
	}
	confirm(p.ImportID)
	got, _ := env.pcs.GetPCByID("pc1")
	if len(got.Items) != 2 || got.Items[0].Name != "Rope" || got.Items[0].CatalogItemID != "cat1" ||
		got.Items[0].Quantity != 1 || got.Items[1].Quantity != 3 {
		t.Errorf("items = %+v", got.Items)
	}
	if got.Currency.GP != 7 {
		t.Errorf("currency = %+v", got.Currency)
	}
}

func companionPDF() []byte {
	return samplePDF("Template.extras.AScomp", ",P4.AScomp.",
		"P4.AScomp.Comp.Race", "Beast of the Burrow", "P4.AScomp.Comp.Use.HP.Max", "n/a")
}

func TestConfirm_CompanionsAreReplaced(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"),
		store.PC{ID: "oldc", OwnerUserID: "u1", Type: "companion", ParentPCID: "pc1", Name: "Old Pet", MaxHP: 4, SharesInitiative: true})
	env.sheets.byPC["oldc"] = &store.CharacterSheet{PCID: "oldc", Revision: 1}

	p := previewOf(t, uploadFor(t, "pc1", companionPDF()))
	if len(p.Companions.Removed) != 1 || p.Companions.Removed[0] != "Old Pet" ||
		len(p.Companions.Added) != 1 || p.Companions.Added[0] != "Beast of the Burrow" {
		t.Errorf("companions = %+v", p.Companions)
	}
	if len(p.Warnings) == 0 {
		t.Error("expected a warning about the companion's HP")
	}
	confirm(p.ImportID)

	comps, _ := env.pcs.GetCompanionsByParentID("pc1")
	if len(comps) != 1 || comps[0].Name != "Beast of the Burrow" || comps[0].MaxHP != 1 || comps[0].SharesInitiative {
		t.Fatalf("companions = %+v", comps)
	}
	if sh := env.sheets.byPC[comps[0].ID]; sh == nil || sh.Creature == nil {
		t.Error("new companion has no creature sheet")
	}
	if _, ok := env.sheets.byPC["oldc"]; ok {
		t.Error("old companion's sheet should be deleted")
	}
}

func TestConfirm_NewCharacter(t *testing.T) {
	env := setupImport(t, "u1")
	p := previewOf(t, uploadNew(t, samplePDF()))
	w := confirm(p.ImportID)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		PCID     string `json:"pc_id"`
		Revision int    `json:"revision"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	pc, _ := env.pcs.GetPCByID(resp.PCID)
	if pc == nil || pc.OwnerUserID != "u1" || pc.Name != "Puck" || pc.MaxHP != 21 || resp.Revision != 1 {
		t.Errorf("pc = %+v, resp = %+v", pc, resp)
	}
	if env.sheets.byPC[resp.PCID] == nil {
		t.Error("no sheet")
	}
}

func TestConfirm_NewCharacterWithoutNameOrHPIs400(t *testing.T) {
	env := setupImport(t, "u1")
	for _, f := range [][]mpmbtest.Field{
		pdfFields("HP Max", "9"),
		pdfFields("PC Name", "X", "HP Max", "none"),
	} {
		p := previewOf(t, uploadNew(t, mpmbtest.Build(mpmbtest.Doc{Fields: f, Info: mpmbtest.MPMBInfo()})))
		if w := confirm(p.ImportID); w.Code != http.StatusBadRequest {
			t.Errorf("status = %d", w.Code)
		}
	}
	if len(env.pcs.byID) != 0 {
		t.Error("no PC should be created")
	}
}

func TestConfirm_OlderImportsOfThePCAreRemoved(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))
	first := previewOf(t, uploadFor(t, "pc1", samplePDF()))
	confirm(first.ImportID)
	second := previewOf(t, uploadFor(t, "pc1", samplePDF()))
	confirm(second.ImportID)
	if len(env.imports.byID) != 1 || env.imports.byID[second.ImportID] == nil {
		t.Errorf("imports = %v", env.imports.byID)
	}
}

func TestDiscardAndConfirm_AccessRules(t *testing.T) {
	env := setupImport(t, "u1", ownedPC("pc1"))
	p := previewOf(t, uploadFor(t, "pc1", samplePDF()))

	withFakeAuth(t, "u2")
	if w := confirm(p.ImportID); w.Code != http.StatusNotFound {
		t.Errorf("other user confirm = %d", w.Code)
	}
	if w := discard(p.ImportID); w.Code != http.StatusNotFound {
		t.Errorf("other user discard = %d", w.Code)
	}
	if len(env.imports.byID) != 1 {
		t.Fatal("another user's import must survive")
	}

	withFakeAuth(t, "u1")
	if w := discard(p.ImportID); w.Code != http.StatusNoContent {
		t.Errorf("discard = %d", w.Code)
	}
	if len(env.imports.byID) != 0 {
		t.Error("draft not deleted")
	}
	if w := discard(p.ImportID); w.Code != http.StatusNotFound {
		t.Errorf("expired discard = %d", w.Code)
	}
	if w := confirm(p.ImportID); w.Code != http.StatusNotFound {
		t.Errorf("expired confirm = %d", w.Code)
	}
}
