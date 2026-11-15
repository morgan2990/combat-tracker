// Package mpmbtest builds small synthetic PDFs shaped like an MPMB sheet, for
// tests. The repository is public, so tests never commit a real sheet.
package mpmbtest

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Field is one form field. Name is the qualified name; each dot-separated
// segment becomes a level of the field hierarchy, as in a real MPMB sheet.
type Field struct {
	Name  string
	Value string
	// Kind is "" for a text field, "name" for a checkbox-style /Name value
	// (Value is then e.g. "True"), "stream" for a value stored as a stream, or
	// "choice" for a choice field with Opt and Selected.
	Kind     string
	TM       string
	Opt      [][2]string // export value, label
	Selected int         // /I for a choice field; -1 for none
	// Widget adds a kid with /Type /Template and no /T, like MPMB's hidden
	// template pages, which pdfcpu's own validation rejects.
	Widget bool
}

// Doc is a whole PDF.
type Doc struct {
	Fields []Field
	Info   map[string]string
	NoForm bool // leave out the AcroForm, like a flattened sheet
}

// MPMBInfo is the document info of a v14.0.8-beta sheet.
func MPMBInfo() map[string]string {
	return map[string]string{
		"SheetVersion":      "14.0.8",
		"SheetVersionBuild": "260530",
		"SheetVersionType":  "beta",
		"SheetType":         "Colorful - A4",
		"Title":             "MorePurpleMoreBetter's D&D 5th edition Character Record Sheet v14.0.8-beta+260530 (Colorful - A4)",
	}
}

type node struct {
	name     string
	kids     []*node
	field    *Field
	id       int
	parentID int
}

func (n *node) child(name string) *node {
	for _, k := range n.kids {
		if k.name == name && k.field == nil {
			return k
		}
	}
	k := &node{name: name}
	n.kids = append(n.kids, k)
	return k
}

// Build renders d as a classic-xref PDF.
func Build(d Doc) []byte {
	root := &node{}
	for i := range d.Fields {
		f := &d.Fields[i]
		cur := root
		parts := strings.Split(f.Name, ".")
		for _, p := range parts[:len(parts)-1] {
			cur = cur.child(p)
		}
		cur.kids = append(cur.kids, &node{name: parts[len(parts)-1], field: f})
	}

	// Fixed objects: 1 catalog, 2 pages, 3 page, 4 acroform, 5 info.
	next := 6
	var objs [][2]string // id, body — id as string for ordering
	type obj struct {
		id   int
		body string
	}
	var all []obj
	add := func(id int, body string) { all = append(all, obj{id, body}) }

	var assign func(n *node, parent int)
	assign = func(n *node, parent int) {
		for _, k := range n.kids {
			k.id = next
			next++
			k.parentID = parent
			assign(k, k.id)
		}
	}
	assign(root, 0)

	var emit func(n *node)
	emit = func(n *node) {
		for _, k := range n.kids {
			var b strings.Builder
			fmt.Fprintf(&b, "<< /T %s", pdfString(k.name))
			if k.parentID != 0 {
				fmt.Fprintf(&b, " /Parent %d 0 R", k.parentID)
			}
			if k.field == nil {
				b.WriteString(" /Kids [")
				for _, c := range k.kids {
					fmt.Fprintf(&b, " %d 0 R", c.id)
				}
				b.WriteString(" ] >>")
				add(k.id, b.String())
				emit(k)
				continue
			}
			f := k.field
			switch f.Kind {
			case "name":
				fmt.Fprintf(&b, " /FT /Btn /V /%s", f.Value)
			case "stream":
				sid := next
				next++
				add(sid, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(f.Value), f.Value))
				fmt.Fprintf(&b, " /FT /Tx /V %d 0 R", sid)
			case "choice":
				b.WriteString(" /FT /Ch /Opt [")
				for _, o := range f.Opt {
					fmt.Fprintf(&b, " [%s %s]", pdfString(o[0]), pdfString(o[1]))
				}
				b.WriteString(" ]")
				fmt.Fprintf(&b, " /V %s", pdfString(f.Value))
				if f.Selected >= 0 {
					fmt.Fprintf(&b, " /I [%d]", f.Selected)
				}
			default:
				fmt.Fprintf(&b, " /FT /Tx /V %s", pdfString(f.Value))
			}
			if f.TM != "" {
				fmt.Fprintf(&b, " /TM %s", pdfString(f.TM))
			}
			if f.Widget {
				wid := next
				next++
				add(wid, fmt.Sprintf("<< /Type /Template /Subtype /Widget /Parent %d 0 R >>", k.id))
				fmt.Fprintf(&b, " /Kids [%d 0 R]", wid)
			}
			b.WriteString(" >>")
			add(k.id, b.String())
		}
	}
	emit(root)

	var top strings.Builder
	for _, k := range root.kids {
		fmt.Fprintf(&top, " %d 0 R", k.id)
	}

	var info strings.Builder
	info.WriteString("<<")
	for k, v := range d.Info {
		fmt.Fprintf(&info, " /%s %s", k, pdfString(v))
	}
	info.WriteString(" >>")

	catalog := "<< /Type /Catalog /Pages 2 0 R"
	if !d.NoForm {
		catalog += " /AcroForm 4 0 R"
	}
	catalog += " >>"
	add(1, catalog)
	add(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	add(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] >>")
	add(4, fmt.Sprintf("<< /Fields [%s ] >>", top.String()))
	add(5, info.String())
	_ = objs

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	offsets := make(map[int]int, len(all))
	for _, o := range all {
		offsets[o.id] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", o.id, o.body)
	}
	size := next
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", size)
	for id := 1; id < size; id++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /Info 5 0 R >>\nstartxref\n%d\n%%%%EOF\n", size, xref)
	return buf.Bytes()
}

// pdfString encodes s as a literal string, or a UTF-16BE hex string when it
// holds anything outside ASCII (which is how MPMB stores most text).
func pdfString(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 || s[i] == '\r' || s[i] == '\n' {
			ascii = false
			break
		}
	}
	if ascii {
		r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
		return "(" + r.Replace(s) + ")"
	}
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, u := range utf16.Encode([]rune(s)) {
		fmt.Fprintf(&b, "%04X", u)
	}
	b.WriteString(">")
	return b.String()
}
