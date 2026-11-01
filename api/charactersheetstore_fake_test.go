package api

import (
	"testing"

	"combatapp/store"
)

// fakeSheetStore is an in-memory sheetStore for tests. SaveSection mirrors
// the real compare-and-set on revision, and — like the real bson encoding —
// drops the identity name and combat max_hp, which live on the PC document.
type fakeSheetStore struct {
	byPC map[string]*store.CharacterSheet
	// forceErr, if set, is returned by every method instead of doing anything.
	forceErr error
}

func newFakeSheetStore(sheets ...store.CharacterSheet) *fakeSheetStore {
	f := &fakeSheetStore{byPC: make(map[string]*store.CharacterSheet)}
	for i := range sheets {
		sh := sheets[i]
		f.byPC[sh.PCID] = &sh
	}
	return f
}

func withFakeSheetStore(t *testing.T, fake *fakeSheetStore) {
	t.Helper()
	original := sheetDB
	sheetDB = fake
	t.Cleanup(func() { sheetDB = original })
}

func (f *fakeSheetStore) GetByPCID(pcID string) (*store.CharacterSheet, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	sh, ok := f.byPC[pcID]
	if !ok {
		return nil, nil
	}
	cp := *sh
	return &cp, nil
}

func (f *fakeSheetStore) GetByPCIDs(pcIDs []string) (map[string]store.CharacterSheet, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := make(map[string]store.CharacterSheet)
	for _, id := range pcIDs {
		if sh, ok := f.byPC[id]; ok {
			out[id] = *sh
		}
	}
	return out, nil
}

func (f *fakeSheetStore) SaveSection(pcID, section string, revision int, value any) (int, error) {
	if f.forceErr != nil {
		return 0, f.forceErr
	}
	sh, exists := f.byPC[pcID]
	switch {
	case revision == 0 && exists:
		return 0, store.ErrSheetConflict
	case revision != 0 && (!exists || sh.Revision != revision):
		return 0, store.ErrSheetConflict
	case revision == 0:
		fresh := store.EmptyCharacterSheet(pcID)
		sh = &fresh
		f.byPC[pcID] = sh
	}
	assignSection(sh, value)
	sh.Revision++
	return sh.Revision, nil
}

func (f *fakeSheetStore) DeleteByPCIDs(pcIDs []string) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	for _, id := range pcIDs {
		delete(f.byPC, id)
	}
	return nil
}

func assignSection(sh *store.CharacterSheet, value any) {
	switch v := value.(type) {
	case *store.SheetIdentity:
		cp := *v
		cp.Name = ""
		sh.Identity = &cp
	case *store.SheetAbilities:
		sh.Abilities = v
	case *store.SheetSkills:
		sh.Skills = v
	case *store.SheetCombat:
		cp := *v
		cp.MaxHP = 0
		sh.Combat = &cp
	case *store.SheetDefenses:
		sh.Defenses = v
	case *[]store.Attack:
		sh.Attacks = *v
	case *store.SheetActions:
		sh.Actions = v
	case *[]store.LimitedUse:
		sh.LimitedUses = *v
	case *[]store.Feature:
		sh.Features = *v
	case *[]store.WeaponMastery:
		sh.WeaponMasteries = *v
	case *store.SheetProficiencies:
		sh.Proficiencies = v
	case *[]store.MagicItem:
		sh.MagicItems = *v
	case *store.SheetSpellcasting:
		sh.Spellcasting = v
	case *store.SheetWildShapes:
		sh.WildShapes = v
	case *store.SheetNotes:
		sh.Notes = v
	case *store.Creature:
		sh.Creature = v
	}
}
