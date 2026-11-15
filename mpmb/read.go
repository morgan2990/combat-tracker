// Package mpmb reads an MPMB character record sheet PDF and maps its form
// fields onto the app's character sheet. It does no I/O of its own beyond the
// reader it is handed, and touches neither the database nor HTTP.
package mpmb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"combatapp/store"
)

// Limits on what an uploaded file may make the reader do. Variables, not
// constants, only so tests can lower them.
var (
	maxFields     = 20000
	maxDepth      = 32
	maxValueBytes = int64(1 << 20)
)

// Errors Read returns; each message is safe to show to the uploader.
var (
	ErrNoForm     = errors.New("this PDF has no form fields; upload the MPMB sheet without flattening it")
	ErrNotMPMB    = errors.New("this PDF is not an MPMB character sheet")
	ErrUnreadable = errors.New("this file could not be read as a PDF")
	ErrTooComplex = errors.New("this PDF has too many form fields, or they are nested too deeply")
)

// scriptsField holds the sheet's loaded scripts, including published book
// content. It is megabytes of code and is never decoded or kept.
const scriptsField = "User_Imported_Files.Stringified"

// Info is the sheet's version metadata from the PDF's document info.
type Info struct {
	Version     string // e.g. "14.0.8"
	Build       string // e.g. "260530"
	VersionType string // e.g. "beta"
	SheetType   string // e.g. "Colorful - A4"
	Title       string
}

// Document is a read MPMB sheet: its version info and every non-empty form
// field, in document order, minus the scripts bundle.
type Document struct {
	Info    Info
	Fields  []store.RawField
	Skipped int // values left out because they were too large or undecodable
	byName  map[string]int
}

// Get returns a field's value, or "" if it is absent or empty.
func (d *Document) Get(name string) string {
	if i, ok := d.byName[name]; ok {
		return d.Fields[i].Value
	}
	return ""
}

// Display returns a choice field's selected label, falling back to its value.
func (d *Document) Display(name string) string {
	i, ok := d.byName[name]
	if !ok {
		return ""
	}
	if f := d.Fields[i]; f.Display != "" {
		return f.Display
	}
	return d.Fields[i].Value
}

// TM returns a field's mapping name, or "".
func (d *Document) TM(name string) string {
	if i, ok := d.byName[name]; ok {
		return d.Fields[i].TM
	}
	return ""
}

// Has reports whether a non-empty field with this name exists.
func (d *Document) Has(name string) bool {
	_, ok := d.byName[name]
	return ok
}

// Read parses an MPMB sheet. It does not validate the PDF — MPMB's hidden
// template pages fail pdfcpu's validation — so it enforces its own limits,
// and never executes or evaluates anything in the file.
func Read(rs io.ReadSeeker) (doc *Document, err error) {
	defer func() {
		if r := recover(); r != nil {
			doc, err = nil, fmt.Errorf("%w: %v", ErrUnreadable, r)
		}
	}()

	// A stateless configuration: the default one creates a config directory in
	// the user's home on first use, which a server has no business doing.
	ctx, err := api.ReadContext(context.Background(), rs, model.NewStatelessConfiguration())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	x := ctx.XRefTable

	w := &walker{x: x, seen: map[int]bool{}}
	if err := w.readForm(); err != nil {
		return nil, err
	}
	if len(w.fields) == 0 && !w.hasFields {
		return nil, ErrNoForm
	}

	info := readInfo(x)
	if info.Version == "" {
		return nil, ErrNotMPMB
	}

	doc = &Document{Info: info, Fields: w.fields, Skipped: w.skipped, byName: make(map[string]int, len(w.fields))}
	for i, f := range doc.Fields {
		doc.byName[f.Name] = i
	}
	return doc, nil
}

func readInfo(x *model.XRefTable) Info {
	var info Info
	if x.Info == nil {
		return info
	}
	d, err := x.DereferenceDict(*x.Info)
	if err != nil || d == nil {
		return info
	}
	get := func(key string) string {
		o, ok := d.Find(key)
		if !ok {
			return ""
		}
		if n, ok := o.(types.Name); ok {
			return n.Value()
		}
		s, err := x.DereferenceText(o)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(s)
	}
	info.Version = get("SheetVersion")
	info.Build = get("SheetVersionBuild")
	info.VersionType = get("SheetVersionType")
	info.SheetType = get("SheetType")
	info.Title = get("Title")
	return info
}

type walker struct {
	x         *model.XRefTable
	fields    []store.RawField
	seen      map[int]bool
	skipped   int
	hasFields bool
	count     int
	err       error
}

func (w *walker) readForm() error {
	cat, err := w.x.Catalog()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	af, ok := cat.Find("AcroForm")
	if !ok {
		return nil
	}
	afd, err := w.x.DereferenceDict(af)
	if err != nil || afd == nil {
		return nil
	}
	fieldsObj, ok := afd.Find("Fields")
	if !ok {
		return nil
	}
	roots, err := w.x.DereferenceArray(fieldsObj)
	if err != nil {
		return nil
	}
	for _, o := range roots {
		w.walk(o, "", "", 0)
		if w.err != nil {
			return w.err
		}
	}
	return nil
}

func (w *walker) text(o types.Object) string {
	s, err := w.x.DereferenceText(o)
	if err != nil {
		return ""
	}
	return s
}

func (w *walker) walk(o types.Object, prefix, ft string, depth int) {
	if w.err != nil {
		return
	}
	if depth > maxDepth {
		w.err = ErrTooComplex
		return
	}
	if ref, ok := o.(types.IndirectRef); ok {
		n := ref.ObjectNumber.Value()
		if w.seen[n] {
			return
		}
		w.seen[n] = true
	}
	d, err := w.x.DereferenceDict(o)
	if err != nil || d == nil {
		return
	}

	name := prefix
	if t, ok := d.Find("T"); ok {
		if s := w.text(t); s != "" {
			if prefix != "" {
				name = prefix + "." + s
			} else {
				name = s
			}
		}
	}
	if f, ok := d.Find("FT"); ok {
		if fo, err := w.x.Dereference(f); err == nil {
			if n, ok := fo.(types.Name); ok {
				ft = "/" + n.Value()
			}
		}
	}

	// A field with named kids is a parent; kids without /T are widgets.
	var kids types.Array
	if k, ok := d.Find("Kids"); ok {
		kids, _ = w.x.DereferenceArray(k)
	}
	for _, k := range kids {
		if kd, err := w.x.DereferenceDict(k); err == nil && kd != nil {
			if _, named := kd.Find("T"); named {
				for _, k2 := range kids {
					w.walk(k2, name, ft, depth+1)
				}
				return
			}
		}
	}

	w.hasFields = true
	w.count++
	if w.count > maxFields {
		w.err = ErrTooComplex
		return
	}
	f := store.RawField{Name: name, Type: ft}
	if t, ok := d.Find("TM"); ok {
		f.TM = w.text(t)
	}
	if v, ok := d.Find("V"); ok {
		f.Value = w.value(name, v)
	}
	if strings.TrimSpace(f.Value) == "" || f.Value == "/Off" || name == scriptsField {
		return
	}
	if ft == "/Ch" {
		f.Display = w.choiceDisplay(d, f.Value)
	}
	w.fields = append(w.fields, f)
}

// value renders a field's /V as text: a name as "/Name", a string decoded,
// a stream decoded with a size limit, an array joined by commas.
func (w *walker) value(name string, o types.Object) string {
	d, err := w.x.Dereference(o)
	if err != nil || d == nil {
		return ""
	}
	switch v := d.(type) {
	case types.Name:
		return "/" + v.Value()
	case types.StringLiteral, types.HexLiteral:
		return w.text(d)
	case types.StreamDict:
		if name == scriptsField {
			return ""
		}
		sd := v
		if err := sd.DecodeWithLimit(maxValueBytes); err != nil {
			w.skipped++
			return ""
		}
		if int64(len(sd.Content)) > maxValueBytes {
			w.skipped++
			return ""
		}
		return string(sd.Content)
	case types.Array:
		parts := make([]string, 0, len(v))
		for _, e := range v {
			if s := w.value(name, e); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ",")
	case types.Integer:
		return strconv.Itoa(v.Value())
	case types.Float:
		return strconv.FormatFloat(v.Value(), 'f', -1, 64)
	case types.Boolean:
		return strconv.FormatBool(v.Value())
	default:
		return ""
	}
}

// choiceDisplay finds the label of a choice field's selected option. The
// stored value alone can be ambiguous (MPMB gives Medium and Small the same
// export value), so the selected index (/I) wins over matching on the value.
func (w *walker) choiceDisplay(d types.Dict, value string) string {
	optObj, ok := d.Find("Opt")
	if !ok {
		return ""
	}
	opts, err := w.x.DereferenceArray(optObj)
	if err != nil {
		return ""
	}
	label := func(i int) (export, text string) {
		e, err := w.x.Dereference(opts[i])
		if err != nil {
			return "", ""
		}
		if pair, ok := e.(types.Array); ok && len(pair) >= 2 {
			return w.text(pair[0]), w.text(pair[1])
		}
		s := w.text(e)
		return s, s
	}
	if io, ok := d.Find("I"); ok {
		if arr, err := w.x.DereferenceArray(io); err == nil && len(arr) > 0 {
			if idx, err := w.x.DereferenceInteger(arr[0]); err == nil && idx != nil {
				if i := idx.Value(); i >= 0 && i < len(opts) {
					if _, text := label(i); text != "" {
						return text
					}
				}
			}
		}
	}
	for i := range opts {
		if export, text := label(i); export == value {
			return text
		}
	}
	return ""
}
