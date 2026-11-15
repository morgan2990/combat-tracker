package mpmb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// mapper holds the state of one Map call: the document and the warnings
// collected so far. A value that does not fit never fails the import — it
// stays in the raw fields and produces a warning.
type mapper struct {
	d     *Document
	warns []string
	seen  map[string]bool
}

func newMapper(d *Document) *mapper { return &mapper{d: d, seen: map[string]bool{}} }

// warn records a warning once; repeated identical warnings are dropped.
func (m *mapper) warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if m.seen[msg] {
		return
	}
	m.seen[msg] = true
	m.warns = append(m.warns, msg)
}

// get returns a field's value with surrounding whitespace trimmed.
func (m *mapper) get(name string) string { return strings.TrimSpace(m.d.Get(name)) }

// text returns a field's value as display text: markers removed, line breaks
// normalized, outer whitespace trimmed.
func (m *mapper) text(name string) string { return cleanText(m.d.Get(name)) }

// num reads an integer field, accepting a leading + (as in "+6").
func (m *mapper) num(name string) (int, bool) { return parseInt(m.get(name)) }

// intPtr is num for optional sheet fields: nil when the field is blank or not a number.
func (m *mapper) intPtr(name string) *int {
	if n, ok := m.num(name); ok {
		return &n
	}
	return nil
}

// checked reports whether a checkbox field is on.
func (m *mapper) checked(name string) bool {
	switch strings.ToLower(m.get(name)) {
	case "", "/off", "off", "false", "0", "no":
		return false
	}
	return true
}

// prefixes returns the page prefixes a sheet lists for a template page
// (e.g. "AScomp" -> ["P4.AScomp."]), in document order. A sheet with several
// copies of a page lists them all, comma-separated.
func (m *mapper) prefixes(page string) []string {
	var out []string
	for p := range strings.SplitSeq(m.d.Get("Template.extras."+page), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// indices returns the sorted numbers N for which a field prefix+N exists, e.g.
// indices("Attack.", ".Weapon") finds 1, 2, 3 from "Attack.1.Weapon" and so on.
func (m *mapper) indices(prefix, suffix string) []int {
	seen := map[int]bool{}
	for _, f := range m.d.Fields {
		rest, ok := strings.CutPrefix(f.Name, prefix)
		if !ok {
			continue
		}
		if suffix != "" {
			if rest, ok = strings.CutSuffix(rest, suffix); !ok {
				continue
			}
		}
		if n, err := strconv.Atoi(rest); err == nil {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sortInts(out)
	return out
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func parseInt(s string) (int, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "+"))
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func itoa(i int) string { return strconv.Itoa(i) }

var (
	// pendingRE matches the "(1/2)" a sheet appends to an unresolved choice.
	pendingRE = regexp.MustCompile(`\(\d+/\d+\)$`)
	markerRE  = regexp.MustCompile(`#{1,2}◆? ?|\*\*`)
	nbspRE    = regexp.MustCompile(`[ \x{00A0}]+`)
	multiNLRE = regexp.MustCompile(`\n{3,}`)
)

// cleanText turns MPMB's rich text into plain text: \r becomes a newline and
// the bold/bullet markers (#, ##, **, ◆) are removed.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = markerRE.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(nbspRE.ReplaceAllString(l, " "))
	}
	s = multiNLRE.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(s)
}

// abilityName turns a field like "Cha Mod" into "Cha"; "nothing" and "empty" mean none.
func abilityName(s string) string {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), " Mod"))
	switch strings.ToLower(s) {
	case "nothing", "empty", "":
		return ""
	}
	return s
}
