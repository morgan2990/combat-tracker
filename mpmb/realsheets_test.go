package mpmb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openSamples reads every PDF in MPMB_SAMPLES_DIR, keyed by file name. The
// real sheets hold personal data and are never committed, so these tests only
// run on a machine that has them.
func openSamples(t *testing.T) map[string]*Document {
	t.Helper()
	dir := os.Getenv("MPMB_SAMPLES_DIR")
	if dir == "" {
		t.Skip("MPMB_SAMPLES_DIR not set")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.pdf"))
	if len(files) == 0 {
		t.Skip("no PDFs in MPMB_SAMPLES_DIR")
	}
	docs := map[string]*Document{}
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := Read(fh)
		fh.Close()
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		docs[filepath.Base(f)] = doc
	}
	return docs
}

// TestMap_RealSheets checks properties every real v14 sheet must satisfy. It
// prints the mapped result as JSON with -v when MPMB_DUMP is set, for review.
func TestMap_RealSheets(t *testing.T) {
	for name, doc := range openSamples(t) {
		t.Run(name, func(t *testing.T) {
			res := Map(doc)
			s := res.Sheet

			if res.Name == "" || res.MaxHP <= 0 {
				t.Errorf("name=%q maxHP=%d", res.Name, res.MaxHP)
			}
			if s.Identity == nil || len(s.Identity.Classes) == 0 || s.Identity.Species == "" || s.Identity.Size == "" {
				t.Errorf("identity = %+v", s.Identity)
			}
			if s.Abilities == nil || s.Abilities.Wis.Score == nil {
				t.Errorf("abilities = %+v", s.Abilities)
			}
			if s.Skills == nil || len(s.Skills.Entries) < 18 {
				t.Errorf("skills = %+v", s.Skills)
			}
			if s.Combat == nil || s.Combat.AC == nil || len(s.Combat.HitDice) == 0 {
				t.Errorf("combat = %+v", s.Combat)
			}
			if len(s.Features) == 0 || len(s.LimitedUses) == 0 || len(s.MagicItems) == 0 || s.Actions == nil {
				t.Errorf("features=%d uses=%d items=%d actions=%v", len(s.Features), len(s.LimitedUses), len(s.MagicItems), s.Actions)
			}
			if s.Spellcasting == nil || len(s.Spellcasting.SpellLists) == 0 {
				t.Errorf("spellcasting = %+v", s.Spellcasting)
			}
			for _, l := range s.Spellcasting.SpellLists {
				for _, sp := range l.Spells {
					if sp.Preparation == "other" {
						t.Errorf("unrecognized marker %q on %q", sp.Mark, sp.Name)
					}
					if sp.Level < 0 || sp.Level > 9 {
						t.Errorf("%q has level %d", sp.Name, sp.Level)
					}
				}
			}
			for _, f := range s.Features {
				if strings.ContainsAny(f.Name+f.Text, "#◆") || strings.Contains(f.Text, "**") {
					t.Errorf("marker characters left in feature %q", f.Name)
				}
				if f.Kind == "" || f.Name == "" {
					t.Errorf("feature = %+v", f)
				}
			}

			if os.Getenv("MPMB_DUMP") != "" {
				b, _ := json.MarshalIndent(res, "", " ")
				_ = os.WriteFile(filepath.Join(os.Getenv("MPMB_DUMP"), name+".mapped.json"), b, 0o644)
			}
		})
	}
}
