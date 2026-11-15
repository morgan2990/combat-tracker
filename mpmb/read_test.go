package mpmb

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"combatapp/mpmb/mpmbtest"
)

var sizeOpts = [][2]string{{"8", "Gargantuan"}, {"4", "Huge"}, {"2", "Large"}, {"1", "Medium"}, {"1", "Small"}, {"0.5", "Tiny"}}

func read(t *testing.T, d mpmbtest.Doc) (*Document, error) {
	t.Helper()
	return Read(bytes.NewReader(mpmbtest.Build(d)))
}

func mustRead(t *testing.T, d mpmbtest.Doc) *Document {
	t.Helper()
	if d.Info == nil {
		d.Info = mpmbtest.MPMBInfo()
	}
	doc, err := read(t, d)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	return doc
}

func TestRead_NamesFieldsByQualifiedPath(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "PC Name", Value: "Puck"},
		{Name: "Attack.1.To Hit", Value: "+6"},
		{Name: "Attack.1.Weapon", Value: "club"},
		{Name: "P7.SSfront.spells.name.3", Value: "Hex*"},
	}})

	for name, want := range map[string]string{
		"PC Name": "Puck", "Attack.1.To Hit": "+6", "Attack.1.Weapon": "club", "P7.SSfront.spells.name.3": "Hex*",
	} {
		if got := doc.Get(name); got != want {
			t.Errorf("Get(%q) = %q, want %q", name, got, want)
		}
	}
	if len(doc.Fields) != 4 {
		t.Errorf("got %d fields, want 4", len(doc.Fields))
	}
}

func TestRead_KeepsMappingNameAsDataWithoutRenamingTheField(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "Action 9", Value: "Tomá wacho!", TM: "Tomá wacho!#!#Kid Tricks (feat)"},
		{Name: "HP Max", Value: "21", TM: "20,21,27,nothing"},
	}})

	if got := doc.Get("Action 9"); got != "Tomá wacho!" {
		t.Errorf("value = %q (the field must keep its own name, not its /TM)", got)
	}
	if got := doc.TM("Action 9"); got != "Tomá wacho!#!#Kid Tricks (feat)" {
		t.Errorf("TM = %q", got)
	}
	if doc.Get("20,21,27,nothing") != "" {
		t.Error("a field was keyed by its /TM")
	}
}

func TestRead_DecodesNonASCIIText(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "Class Features", Value: "**Beast Master, level 3:**\r#◆ Favored Enemy# (Ranger 1, P24 119)\r   I always…"},
	}})

	got := doc.Get("Class Features")
	if !strings.Contains(got, "◆ Favored Enemy") || !strings.Contains(got, "\r") || !strings.Contains(got, "…") {
		t.Errorf("value = %q", got)
	}
}

func TestRead_DropsEmptyAndUncheckedFields(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "Alignment", Value: ""},
		{Name: "Resistance Damage Type 2", Value: " "},
		{Name: "Dec Prof", Kind: "name", Value: "Off"},
		{Name: "Ste Prof", Kind: "name", Value: "True"},
	}})

	if len(doc.Fields) != 1 || doc.Fields[0].Name != "Ste Prof" || doc.Fields[0].Value != "/True" {
		t.Errorf("fields = %+v, want only the checked checkbox as /True", doc.Fields)
	}
}

func TestRead_ResolvesChoiceLabelFromSelectedIndex(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "Size Category", Kind: "choice", Value: "1", Opt: sizeOpts, Selected: 4},
		{Name: "P4.AScomp.Comp.Desc.Size", Kind: "choice", Value: "1", Opt: sizeOpts, Selected: 3},
		{Name: "Wildshape.Size", Kind: "choice", Value: "0.5", Opt: sizeOpts, Selected: -1},
	}})

	if got := doc.Display("Size Category"); got != "Small" {
		t.Errorf("selected index 4 shows %q, want Small (Medium and Small share the value 1)", got)
	}
	if got := doc.Display("P4.AScomp.Comp.Desc.Size"); got != "Medium" {
		t.Errorf("selected index 3 shows %q, want Medium", got)
	}
	if got := doc.Display("Wildshape.Size"); got != "Tiny" {
		t.Errorf("no index, matched by value: %q, want Tiny", got)
	}
	if got := doc.Get("Size Category"); got != "1" {
		t.Errorf("Get() = %q, want the raw value 1", got)
	}
}

func TestRead_DecodesStreamValuesAndSkipsTheScriptsBundle(t *testing.T) {
	big := strings.Repeat("x", 3<<20)
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "User Script", Kind: "stream", Value: "var iFileName = \"x.js\";"},
		{Name: "User_Imported_Files.Stringified", Kind: "stream", Value: big},
		{Name: "PC Name", Value: "Puck"},
	}})

	if got := doc.Get("User Script"); got != `var iFileName = "x.js";` {
		t.Errorf("stream value = %q", got)
	}
	if doc.Has("User_Imported_Files.Stringified") {
		t.Error("the scripts bundle was kept")
	}
	for _, f := range doc.Fields {
		if len(f.Value) > 1<<20 {
			t.Errorf("field %q holds %d bytes", f.Name, len(f.Value))
		}
	}
}

func TestRead_LeavesOutAnOversizedValueWithoutFailing(t *testing.T) {
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{
		{Name: "Huge", Kind: "stream", Value: strings.Repeat("y", 2<<20)},
		{Name: "PC Name", Value: "Puck"},
	}})

	if doc.Has("Huge") || doc.Get("PC Name") != "Puck" || doc.Skipped != 1 {
		t.Errorf("Has(Huge)=%v PC Name=%q Skipped=%d", doc.Has("Huge"), doc.Get("PC Name"), doc.Skipped)
	}
}

func TestRead_ReadsInfoAndIgnoresTemplateWidgets(t *testing.T) {
	// A kid with /Type /Template and no /T is what makes pdfcpu's own validation reject MPMB sheets.
	doc := mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{{Name: "PC Name", Value: "Puck", Widget: true}}})

	want := Info{Version: "14.0.8", Build: "260530", VersionType: "beta", SheetType: "Colorful - A4"}
	got := doc.Info
	if got.Version != want.Version || got.Build != want.Build || got.VersionType != want.VersionType || got.SheetType != want.SheetType {
		t.Errorf("Info = %+v, want %+v", got, want)
	}
	if doc.Get("PC Name") != "Puck" {
		t.Error("a field with a template widget was lost")
	}
}

func TestRead_Rejections(t *testing.T) {
	t.Run("no form fields", func(t *testing.T) {
		_, err := read(t, mpmbtest.Doc{NoForm: true, Info: mpmbtest.MPMBInfo()})
		if !errors.Is(err, ErrNoForm) {
			t.Errorf("error = %v, want ErrNoForm", err)
		}
	})
	t.Run("form without MPMB version metadata", func(t *testing.T) {
		_, err := read(t, mpmbtest.Doc{Fields: []mpmbtest.Field{{Name: "Name", Value: "x"}}, Info: map[string]string{"Title": "A form"}})
		if !errors.Is(err, ErrNotMPMB) {
			t.Errorf("error = %v, want ErrNotMPMB", err)
		}
	})
	t.Run("not a PDF", func(t *testing.T) {
		_, err := Read(strings.NewReader("this is not a pdf at all"))
		if !errors.Is(err, ErrUnreadable) {
			t.Errorf("error = %v, want ErrUnreadable", err)
		}
	})
	t.Run("truncated PDF", func(t *testing.T) {
		pdf := mpmbtest.Build(mpmbtest.Doc{Fields: []mpmbtest.Field{{Name: "PC Name", Value: "x"}}, Info: mpmbtest.MPMBInfo()})
		// pdfcpu reconstructs what it can from a cut-off file, so which
		// rejection this lands on depends on what survived; it must be one
		// of ours, never a success or a panic.
		_, err := Read(bytes.NewReader(pdf[:len(pdf)/2]))
		if !errors.Is(err, ErrUnreadable) && !errors.Is(err, ErrNotMPMB) && !errors.Is(err, ErrNoForm) {
			t.Errorf("error = %v, want one of the reader's rejections", err)
		}
	})
	t.Run("empty input", func(t *testing.T) {
		if _, err := Read(bytes.NewReader(nil)); !errors.Is(err, ErrUnreadable) {
			t.Errorf("error = %v, want ErrUnreadable", err)
		}
	})
}

func TestRead_EnforcesFieldAndDepthLimits(t *testing.T) {
	oldFields, oldDepth := maxFields, maxDepth
	t.Cleanup(func() { maxFields, maxDepth = oldFields, oldDepth })

	t.Run("too many fields", func(t *testing.T) {
		maxFields = 5
		var fs []mpmbtest.Field
		for i := range 20 {
			fs = append(fs, mpmbtest.Field{Name: "F" + string(rune('A'+i)), Value: "v"})
		}
		if _, err := read(t, mpmbtest.Doc{Fields: fs, Info: mpmbtest.MPMBInfo()}); !errors.Is(err, ErrTooComplex) {
			t.Errorf("error = %v, want ErrTooComplex", err)
		}
		maxFields = oldFields
	})
	t.Run("nested too deeply", func(t *testing.T) {
		maxDepth = 4
		name := strings.Repeat("a.", 10) + "leaf"
		if _, err := read(t, mpmbtest.Doc{Fields: []mpmbtest.Field{{Name: name, Value: "v"}}, Info: mpmbtest.MPMBInfo()}); !errors.Is(err, ErrTooComplex) {
			t.Errorf("error = %v, want ErrTooComplex", err)
		}
		maxDepth = oldDepth
	})
}

func TestRead_DoesNotWriteAConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))

	mustRead(t, mpmbtest.Doc{Fields: []mpmbtest.Field{{Name: "PC Name", Value: "Puck"}}})

	entries, _ := os.ReadDir(home)
	if len(entries) != 0 {
		t.Errorf("reading a PDF created %d entries in the home directory: %v", len(entries), entries)
	}
}

func TestRead_RealSheets(t *testing.T) {
	dir := os.Getenv("MPMB_SAMPLES_DIR")
	if dir == "" {
		t.Skip("MPMB_SAMPLES_DIR not set")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.pdf"))
	if len(files) == 0 {
		t.Skip("no PDFs in MPMB_SAMPLES_DIR")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			fh, err := os.Open(f)
			if err != nil {
				t.Fatal(err)
			}
			defer fh.Close()
			start := time.Now()
			doc, err := Read(fh)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if doc.Info.Version == "" || len(doc.Fields) < 500 || doc.Has("User_Imported_Files.Stringified") {
				t.Errorf("version=%q fields=%d", doc.Info.Version, len(doc.Fields))
			}
			if d := time.Since(start); d > 10*time.Second {
				t.Errorf("took %s", d)
			}
		})
	}
}
