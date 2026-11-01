package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"

	"combatapp/store"
)

// maxSheetBody caps one section save. The largest section in real sheets is a
// few KiB, so this is generous headroom, not a tight limit.
const maxSheetBody = 1 << 20

var (
	spellPreparations = []string{"always", "prepared", "unprepared", "known", "at_will", "limited", "other"}
	spellListKinds    = []string{"class", "feat", "item", "other"}
	featureKinds      = []string{"class", "species", "background", "feat", "other"}
)

// sheetSection describes one saveable section: whether a companion's sheet
// can hold it, and how to decode and validate a request's data into the
// typed value the store persists.
type sheetSection struct {
	forCompanion bool
	parse        func(raw json.RawMessage) (any, error)
}

// newSection builds a sheetSection that decodes into T and then validates it.
// Unknown keys in the data are ignored; a JSON null is rejected.
func newSection[T any](forCompanion bool, validate func(*T) error) sheetSection {
	return sheetSection{forCompanion: forCompanion, parse: func(raw json.RawMessage) (any, error) {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || len(bytes.TrimSpace(raw)) == 0 {
			return nil, errors.New("data required")
		}
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, errors.New("invalid data")
		}
		if validate != nil {
			if err := validate(&v); err != nil {
				return nil, err
			}
		}
		return &v, nil
	}}
}

// sheetSections is the full set of sections a PC's sheet supports, plus the
// single creature section a companion's sheet supports.
var sheetSections = map[string]sheetSection{
	"identity": newSection(false, func(v *store.SheetIdentity) error {
		v.Name = strings.TrimSpace(v.Name)
		if v.Name == "" {
			return errors.New("name required")
		}
		return nil
	}),
	"abilities": newSection[store.SheetAbilities](false, nil),
	"skills":    newSection[store.SheetSkills](false, nil),
	"combat": newSection(false, func(v *store.SheetCombat) error {
		if v.MaxHP <= 0 {
			return errors.New("max_hp must be greater than 0")
		}
		return nil
	}),
	"defenses":         newSection[store.SheetDefenses](false, nil),
	"attacks":          newSection[[]store.Attack](false, nil),
	"actions":          newSection[store.SheetActions](false, nil),
	"limited_uses":     newSection[[]store.LimitedUse](false, nil),
	"features":         newSection(false, validateFeatures),
	"weapon_masteries": newSection[[]store.WeaponMastery](false, nil),
	"proficiencies":    newSection[store.SheetProficiencies](false, nil),
	"magic_items":      newSection[[]store.MagicItem](false, nil),
	"spellcasting":     newSection(false, validateSpellcasting),
	"wild_shapes":      newSection[store.SheetWildShapes](false, nil),
	"notes":            newSection[store.SheetNotes](false, nil),
	"creature":         newSection[store.Creature](true, nil),
}

func validateFeatures(v *[]store.Feature) error {
	for _, f := range *v {
		if !slices.Contains(featureKinds, f.Kind) {
			return fmt.Errorf("invalid feature kind %q", f.Kind)
		}
	}
	return nil
}

func validateSpellcasting(v *store.SheetSpellcasting) error {
	for _, list := range v.SpellLists {
		if !slices.Contains(spellListKinds, list.Kind) {
			return fmt.Errorf("invalid spell list kind %q", list.Kind)
		}
		for _, sp := range list.Spells {
			if !slices.Contains(spellPreparations, sp.Preparation) {
				return fmt.Errorf("invalid spell preparation %q", sp.Preparation)
			}
			if sp.Level < 0 || sp.Level > 9 {
				return fmt.Errorf("spell level %d out of range", sp.Level)
			}
		}
	}
	if len(v.Slots) > 9 {
		return errors.New("at most 9 slot levels")
	}
	for _, n := range v.Slots {
		if n < 0 {
			return errors.New("slot totals cannot be negative")
		}
	}
	if pm := v.PactMagic; pm != nil && (pm.Total < 0 || pm.SlotLevel < 0 || pm.SlotLevel > 9) {
		return errors.New("invalid pact magic")
	}
	return nil
}

// sheetBundle is one PC (or companion) with its sheet.
type sheetBundle struct {
	PC    store.PC             `json:"pc"`
	Sheet store.CharacterSheet `json:"sheet"`
}

// sheetResponse is a PC's bundle plus its companions' bundles.
type sheetResponse struct {
	PC         store.PC             `json:"pc"`
	Sheet      store.CharacterSheet `json:"sheet"`
	Companions []sheetBundle        `json:"companions"`
}

// writeSheetResponse loads pc's companions and every involved sheet, and
// writes the combined response. A PC with no stored sheet reads as empty.
func writeSheetResponse(w http.ResponseWriter, pc *store.PC) {
	companions, err := pcDB.GetCompanionsByParentID(pc.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	ids := []string{pc.ID}
	for _, c := range companions {
		ids = append(ids, c.ID)
	}
	sheets, err := sheetDB.GetByPCIDs(ids)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	sheetOf := func(id string) store.CharacterSheet {
		if sh, ok := sheets[id]; ok {
			return sh
		}
		return store.EmptyCharacterSheet(id)
	}
	resp := sheetResponse{PC: *pc, Sheet: sheetOf(pc.ID), Companions: []sheetBundle{}}
	for _, c := range companions {
		resp.Companions = append(resp.Companions, sheetBundle{PC: c, Sheet: sheetOf(c.ID)})
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetSheet returns the owner's view of a sheet. Like GetPC, it answers 404
// (not 403) for a PC the requester doesn't own, so existence isn't revealed.
func GetSheet(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	pc, err := pcDB.GetPCByID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pc == nil || pc.OwnerUserID != userID {
		http.NotFound(w, r)
		return
	}
	writeSheetResponse(w, pc)
}

// GetRoomPCSheet gives a room's DM read-only access to the sheet of a PC in
// the room's linked party (and its companions').
func GetRoomPCSheet(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	rm, ok := requireOwnedRoom(w, r, r.PathValue("room_id"), userID)
	if !ok {
		return
	}
	pcID := r.PathValue("pc_id")
	party, err := resolveLinkedParty(rm)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if party == nil || !slices.Contains(party.MemberPCIDs, pcID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	pc, err := pcDB.GetPCByID(pcID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pc == nil {
		http.NotFound(w, r)
		return
	}
	writeSheetResponse(w, pc)
}

type sheetSavePayload struct {
	Revision int             `json:"revision"`
	Data     json.RawMessage `json:"data"`
}

// SaveSheetSection saves one section of the owner's sheet. identity and
// combat also carry the PC's name and max_hp, which live on the PC document.
func SaveSheetSection(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	pc, ok := requireOwnedPC(w, r, id, userID)
	if !ok {
		return
	}
	name := r.PathValue("section")
	section, known := sheetSections[name]
	if !known {
		http.Error(w, "unknown section", http.StatusBadRequest)
		return
	}
	isCompanion := pc.Type == "companion"
	if isCompanion != section.forCompanion {
		http.Error(w, "section does not apply to this character", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSheetBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			http.Error(w, "section too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var payload sheetSavePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	value, err := section.parse(payload.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newName, newMaxHP := pc.Name, pc.MaxHP
	switch v := value.(type) {
	case *store.SheetIdentity:
		newName = v.Name
	case *store.SheetCombat:
		newMaxHP = v.MaxHP
	}

	revision, err := sheetDB.SaveSection(id, name, payload.Revision, value)
	if errors.Is(err, store.ErrSheetConflict) {
		http.Error(w, "sheet was changed elsewhere; reload and try again", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if newName != pc.Name || newMaxHP != pc.MaxHP {
		// Optional fields are omitted, so items/currency/shares_initiative are kept.
		if err := pcDB.UpdatePC(id, store.PCUpdateFields{Name: newName, MaxHP: newMaxHP}); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"revision": revision})
}

// deleteSheetsFor removes the sheets of pc and, for a top-level PC, of its
// companions. A failure is logged, not surfaced: the PC is already gone, and
// an unreachable sheet harms no one.
func deleteSheetsFor(pc *store.PC, companions []store.PC) {
	ids := []string{pc.ID}
	if pc.Type != "companion" {
		for _, c := range companions {
			ids = append(ids, c.ID)
		}
	}
	if err := sheetDB.DeleteByPCIDs(ids); err != nil {
		log.Printf("character sheet: deleting sheets for pc %s: %v", pc.ID, err)
	}
}
