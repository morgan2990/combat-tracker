package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"combatapp/mpmb"
	"combatapp/sheetdiff"
	"combatapp/store"
)

// maxImportUpload is the largest PDF accepted. Real MPMB sheets are ~30 MB.
// A variable only so tests can lower it.
var maxImportUpload int64 = 64 << 20

// multipartSlack leaves room for the multipart framing around the file.
const multipartSlack = 1 << 20

// importSlot lets one import run at a time per server: parsing a sheet takes
// ~100 MB, and a small group will rarely collide.
var importSlot = make(chan struct{}, 1)

// importPreview is the response to an upload.
type importPreview struct {
	ImportID     string                    `json:"import_id"`
	NewCharacter bool                      `json:"new_character"`
	Name         string                    `json:"name"`
	MaxHP        int                       `json:"max_hp"`
	Source       store.SheetSource         `json:"source"`
	Warnings     []string                  `json:"warnings"`
	Changes      []sheetdiff.Change        `json:"changes"`
	Companions   sheetdiff.CompanionChange `json:"companions"`
	Inventory    sheetdiff.InventoryChange `json:"inventory"`
}

// uploadedSheet is a PDF read and mapped, ready to store as a draft.
type uploadedSheet struct {
	fileName string
	hash     string
	doc      *mpmb.Document
	parsed   store.ImportParsed
}

// receiveUpload takes the "file" part of a multipart request, streams it to a
// temp file while hashing it, and reads and maps it. On failure it has already
// written the response. The temp file is removed before it returns.
func receiveUpload(w http.ResponseWriter, r *http.Request) (*uploadedSheet, bool) {
	select {
	case importSlot <- struct{}{}:
		defer func() { <-importSlot }()
	default:
		http.Error(w, "another import is being processed; try again in a moment", http.StatusTooManyRequests)
		return nil, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportUpload+multipartSlack)
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "expected a multipart upload with a file field", http.StatusBadRequest)
		return nil, false
	}
	var part *multipart.Part
	for {
		p, err := mr.NextPart()
		if err != nil {
			if tooLarge(err) {
				http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
				return nil, false
			}
			http.Error(w, "no file uploaded", http.StatusBadRequest)
			return nil, false
		}
		if p.FormName() == "file" {
			part = p
			break
		}
	}

	tmp, err := os.CreateTemp("", "mpmb-upload-*.pdf")
	if err != nil {
		http.Error(w, "could not store the upload", http.StatusInternalServerError)
		return nil, false
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(part, maxImportUpload+1))
	if err != nil {
		if tooLarge(err) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return nil, false
		}
		http.Error(w, "upload failed", http.StatusBadRequest)
		return nil, false
	}
	if n > maxImportUpload {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return nil, false
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "could not read the upload", http.StatusInternalServerError)
		return nil, false
	}

	doc, err := mpmb.Read(tmp)
	if err != nil {
		if isReaderError(err) {
			http.Error(w, readerMessage(err), http.StatusBadRequest)
		} else {
			log.Printf("sheet import: reading upload: %v", err)
			http.Error(w, mpmb.ErrUnreadable.Error(), http.StatusBadRequest)
		}
		return nil, false
	}
	return &uploadedSheet{
		fileName: part.FileName(),
		hash:     hex.EncodeToString(h.Sum(nil)),
		doc:      doc,
		parsed:   mpmb.Map(doc),
	}, true
}

func tooLarge(err error) bool {
	_, ok := errors.AsType[*http.MaxBytesError](err)
	return ok
}

func isReaderError(err error) bool {
	return errors.Is(err, mpmb.ErrNoForm) || errors.Is(err, mpmb.ErrNotMPMB) ||
		errors.Is(err, mpmb.ErrUnreadable) || errors.Is(err, mpmb.ErrTooComplex)
}

// readerMessage is the reader's own message without the wrapped library detail.
func readerMessage(err error) string {
	for _, e := range []error{mpmb.ErrNoForm, mpmb.ErrNotMPMB, mpmb.ErrTooComplex, mpmb.ErrUnreadable} {
		if errors.Is(err, e) {
			return e.Error()
		}
	}
	return err.Error()
}

// storeDraft saves the upload as a draft owned by userID.
func storeDraft(u *uploadedSheet, userID, pcID string, baseRevision int) (*store.CharacterSheetImport, error) {
	return importDB.Create(store.CharacterSheetImport{
		OwnerUserID:  userID,
		PCID:         pcID,
		BaseRevision: baseRevision,
		Source: store.SheetSource{
			Kind:          "mpmb",
			SheetVersion:  u.doc.Info.Version,
			SheetBuild:    u.doc.Info.Build,
			FileName:      u.fileName,
			FileSHA256:    u.hash,
			MapperVersion: mpmb.MapperVersion,
		},
		Parsed:    u.parsed,
		RawFields: u.doc.Fields,
	})
}

// UploadSheetImport reads an MPMB PDF for an existing PC the requester owns,
// stores a draft, and returns the differences. It changes nothing else.
func UploadSheetImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	pc, ok := requireOwnedPC(w, r, r.PathValue("id"), userID)
	if !ok {
		return
	}
	if pc.Type == "companion" {
		http.Error(w, "a companion's sheet cannot be imported from a PDF", http.StatusBadRequest)
		return
	}
	u, ok := receiveUpload(w, r)
	if !ok {
		return
	}

	current, err := sheetDB.GetByPCID(pc.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	cur := store.EmptyCharacterSheet(pc.ID)
	if current != nil {
		cur = *current
	}
	companions, err := pcDB.GetCompanionsByParentID(pc.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	draft, err := storeDraft(u, userID, pc.ID, cur.Revision)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	newName, newMaxHP := pc.Name, pc.MaxHP
	if u.parsed.Name != "" {
		newName = u.parsed.Name
	}
	if u.parsed.MaxHP > 0 {
		newMaxHP = u.parsed.MaxHP
	}
	writeJSON(w, http.StatusOK, importPreview{
		ImportID: draft.ID,
		Name:     newName,
		MaxHP:    newMaxHP,
		Source:   draft.Source,
		Warnings: nonNil(u.parsed.Warnings),
		Changes: nonNilChanges(sheetdiff.Sheets(
			sheetdiff.Input{Name: pc.Name, MaxHP: pc.MaxHP, Sheet: cur},
			sheetdiff.Input{Name: newName, MaxHP: newMaxHP, Sheet: u.parsed.Sheet},
		)),
		Companions: sheetdiff.Companions(companions, u.parsed.Companions),
		Inventory:  sheetdiff.Inventory(pc.Items, pc.Currency, u.parsed.Inventory),
	})
}

// UploadNewCharacterImport reads an MPMB PDF for a character that does not
// exist yet. Nothing is created until the draft is confirmed.
func UploadNewCharacterImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	u, ok := receiveUpload(w, r)
	if !ok {
		return
	}
	draft, err := storeDraft(u, userID, "", 0)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, importPreview{
		ImportID:     draft.ID,
		NewCharacter: true,
		Name:         u.parsed.Name,
		MaxHP:        u.parsed.MaxHP,
		Source:       draft.Source,
		Warnings:     nonNil(u.parsed.Warnings),
		Changes: nonNilChanges(sheetdiff.Sheets(
			sheetdiff.Input{},
			sheetdiff.Input{Name: u.parsed.Name, MaxHP: u.parsed.MaxHP, Sheet: u.parsed.Sheet},
		)),
		Companions: sheetdiff.Companions(nil, u.parsed.Companions),
		Inventory:  sheetdiff.Inventory(nil, store.Currency{}, u.parsed.Inventory),
	})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilChanges(c []sheetdiff.Change) []sheetdiff.Change {
	if c == nil {
		return []sheetdiff.Change{}
	}
	return c
}

// ownedImport loads an import the requester uploaded. Someone else's import
// and a missing (or expired) one are both 404.
func ownedImport(w http.ResponseWriter, r *http.Request, userID string) (*store.CharacterSheetImport, bool) {
	imp, err := importDB.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return nil, false
	}
	if imp == nil || imp.OwnerUserID != userID {
		http.NotFound(w, r)
		return nil, false
	}
	return imp, true
}

// DiscardSheetImport deletes a draft; nothing else changes.
func DiscardSheetImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	imp, ok := ownedImport(w, r, userID)
	if !ok {
		return
	}
	if imp.Status != store.ImportDraft {
		http.Error(w, "this import was already applied", http.StatusConflict)
		return
	}
	if err := importDB.Delete(imp.ID); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ConfirmSheetImport applies a draft: it replaces the whole sheet, updates the
// PC, and replaces its companions. The sheet replace goes first and is the
// guard — if the sheet changed since the upload, nothing else happens.
func ConfirmSheetImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	imp, ok := ownedImport(w, r, userID)
	if !ok {
		return
	}
	if imp.Status != store.ImportDraft {
		http.Error(w, "this import was already applied", http.StatusConflict)
		return
	}
	parsed := imp.Parsed
	if imp.PCID == "" && (strings.TrimSpace(parsed.Name) == "" || parsed.MaxHP <= 0) {
		http.Error(w, "the sheet needs a character name and a positive max HP to create a character", http.StatusBadRequest)
		return
	}

	// Validate the target before claiming, so a bad request leaves the draft usable.
	var pc *store.PC
	if imp.PCID != "" {
		var err error
		pc, err = pcDB.GetPCByID(imp.PCID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if pc == nil || pc.OwnerUserID != userID {
			http.NotFound(w, r)
			return
		}
	}

	claimed, err := importDB.Claim(imp.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if claimed == nil {
		http.Error(w, "this import was already applied", http.StatusConflict)
		return
	}

	pcID, revision, status, msg := applyImport(claimed, pc, userID)
	if status != http.StatusOK {
		if err := importDB.Release(claimed.ID); err != nil {
			log.Printf("sheet import %s: releasing after failed apply: %v", claimed.ID, err)
		}
		http.Error(w, msg, status)
		return
	}
	if err := importDB.MarkApplied(claimed.ID, pcID); err != nil {
		log.Printf("sheet import %s: marking applied: %v", claimed.ID, err)
	} else if err := importDB.DeleteOthersForPC(pcID, claimed.ID); err != nil {
		log.Printf("sheet import %s: deleting older imports: %v", claimed.ID, err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pc_id": pcID, "revision": revision})
}

// applyImport runs the confirm steps. pc is nil for a new character. It
// returns the PC id and new sheet revision, or an HTTP status and message.
func applyImport(imp *store.CharacterSheetImport, pc *store.PC, userID string) (pcID string, revision, status int, msg string) {
	parsed := imp.Parsed
	const dbErr = "database error"
	created := false

	name, maxHP := parsed.Name, parsed.MaxHP
	base := imp.BaseRevision
	if pc == nil {
		newPC, err := pcDB.CreatePC(userID, strings.TrimSpace(name), maxHP)
		if err != nil {
			return "", 0, http.StatusInternalServerError, dbErr
		}
		pc, created, base = newPC, true, 0
	} else {
		if strings.TrimSpace(name) == "" {
			name = pc.Name
		}
		if maxHP <= 0 {
			maxHP = pc.MaxHP
		}
	}

	src := imp.Source
	now := time.Now().UTC()
	src.ImportedAt = &now
	src.ImportID = imp.ID
	sheet := parsed.Sheet
	sheet.PCID = pc.ID
	sheet.Source = &src
	sheet.Creature = nil

	revision, err := sheetDB.ReplaceSheet(sheet, base)
	if err != nil {
		if created {
			_ = pcDB.DeletePC(pc.ID)
		}
		if errors.Is(err, store.ErrSheetConflict) {
			return "", 0, http.StatusConflict, "the sheet changed after this file was uploaded; upload it again"
		}
		return "", 0, http.StatusInternalServerError, dbErr
	}

	fields := store.PCUpdateFields{Name: name, MaxHP: maxHP}
	if parsed.Inventory.Present {
		fields.Items = mergeCatalogLinks(pc.Items, parsed.Inventory.Items)
		cur := parsed.Inventory.Currency
		fields.Currency = &cur
	}
	if err := pcDB.UpdatePC(pc.ID, fields); err != nil {
		return "", 0, http.StatusInternalServerError, dbErr
	}

	old, err := pcDB.GetCompanionsByParentID(pc.ID)
	if err != nil {
		return "", 0, http.StatusInternalServerError, dbErr
	}
	for _, c := range parsed.Companions {
		hp := c.MaxHP
		if hp <= 0 {
			hp = 1
		}
		comp, err := pcDB.CreateCompanion(pc.ID, userID, c.Name, hp, false)
		if err != nil {
			return "", 0, http.StatusInternalServerError, dbErr
		}
		creature := c.Creature
		if _, err := sheetDB.ReplaceSheet(store.CharacterSheet{PCID: comp.ID, Creature: &creature}, 0); err != nil {
			return "", 0, http.StatusInternalServerError, dbErr
		}
	}
	// The old companions go last, so a failure above leaves them in place.
	oldIDs := make([]string, 0, len(old))
	for _, c := range old {
		if err := pcDB.DeletePC(c.ID); err != nil {
			return "", 0, http.StatusInternalServerError, dbErr
		}
		oldIDs = append(oldIDs, c.ID)
	}
	if err := sheetDB.DeleteByPCIDs(oldIDs); err != nil {
		log.Printf("sheet import %s: deleting old companion sheets: %v", imp.ID, err)
	}
	return pc.ID, revision, http.StatusOK, ""
}

// mergeCatalogLinks keeps a stored item's catalog reference when an imported
// item has exactly the same name.
func mergeCatalogLinks(existing, imported []store.Item) []store.Item {
	byName := make(map[string]store.Item, len(existing))
	for _, it := range existing {
		if it.CatalogItemID != "" {
			byName[it.Name] = it
		}
	}
	out := make([]store.Item, len(imported))
	for i, it := range imported {
		if old, ok := byName[it.Name]; ok {
			it.CatalogItemID = old.CatalogItemID
			it.ReferenceURL = old.ReferenceURL
		}
		out[i] = it
	}
	return out
}
