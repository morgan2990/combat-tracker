package mpmb

import (
	"regexp"
	"strings"

	"combatapp/store"
)

var (
	// **Druid (Circle of Dreams), level 2:**
	classHeaderRE = regexp.MustCompile(`^\*\*(.+?), level (\d+):\*\*\s*$`)
	// #◆ Name# (Source 1, Book 12) #[note]# text   (## for companion rows)
	rowRE = regexp.MustCompile(`^#{1,2}◆ (.+?)#{1,2}\s*(?:\(([^()]*)\))?\s*(?:#\[(.*?)\]#)?\s*(.*)$`)
	// the "Ranger 1" in "(Ranger 1, P24 119)"
	firstSourceRE = regexp.MustCompile(`^(.+?) \d+(?:,|$)`)
	classWithSub  = regexp.MustCompile(`^(.+?) \((.+)\)$`)
	subclassSplit = regexp.MustCompile(`\s+`)
)

// splitLines splits MPMB's rich text on \r and \n.
func splitLines(s string) []string {
	return strings.FieldsFunc(strings.ReplaceAll(s, "\r\n", "\n"), func(r rune) bool { return r == '\r' || r == '\n' })
}

// featureRow is one "#◆ Name# (Source N, Book P) #[note]# text" entry.
type featureRow struct {
	Name, Source, Book, Page, UsesNote string
	Level                              *int
	Text                               string
}

// parseSourceParen reads the parenthesis after a row's name: "Ranger 1, P24 119"
// is source Ranger, level 1, book P24, page 119; "Kid Tricks, HB-KT 1" has no level.
func parseSourceParen(s string) (source string, level *int, book, page string) {
	parts := strings.Split(s, ", ")
	source = strings.TrimSpace(parts[0])
	if sp := strings.LastIndex(source, " "); sp > 0 {
		if n, ok := parseInt(source[sp+1:]); ok {
			level = &n
			source = strings.TrimSpace(source[:sp])
		}
	}
	if len(parts) > 1 {
		bp := strings.TrimSpace(parts[len(parts)-1])
		if sp := strings.LastIndex(bp, " "); sp > 0 {
			if _, ok := parseInt(bp[sp+1:]); ok {
				return source, level, strings.TrimSpace(bp[:sp]), bp[sp+1:]
			}
		}
		book = bp
	}
	return source, level, book, ""
}

// parseRows splits a rich-text field into feature rows. lead is whatever
// precedes the first row. Class headers end the current row and are skipped.
func parseRows(text string) (rows []featureRow, lead string) {
	var cur *featureRow
	var curText, leadLines []string
	flush := func() {
		if cur != nil {
			cur.Text = cleanText(strings.Join(curText, "\n"))
			rows = append(rows, *cur)
			cur, curText = nil, nil
		}
	}
	for _, line := range splitLines(text) {
		if classHeaderRE.MatchString(strings.TrimSpace(line)) {
			flush()
			continue
		}
		if g := rowRE.FindStringSubmatch(strings.TrimSpace(line)); g != nil {
			flush()
			cur = &featureRow{Name: strings.TrimSpace(g[1]), UsesNote: strings.TrimSpace(g[3])}
			if g[2] != "" {
				cur.Source, cur.Level, cur.Book, cur.Page = parseSourceParen(g[2])
			}
			curText = []string{g[4]}
			continue
		}
		if cur != nil {
			curText = append(curText, line)
		} else {
			leadLines = append(leadLines, line)
		}
	}
	flush()
	return rows, cleanText(strings.Join(leadLines, "\n"))
}

// classesFromFeatures reads the class entries from the "**Class (Sub), level N:**"
// headers of Class Features. A header with no base class ("Beast Master") takes
// its base class from the source of its first feature ("Ranger 1").
func classesFromFeatures(text string) []store.SheetClass {
	type section struct {
		header string
		level  int
		first  string
	}
	var secs []*section
	var cur *section
	for _, line := range splitLines(text) {
		line = strings.TrimSpace(line)
		if g := classHeaderRE.FindStringSubmatch(line); g != nil {
			lv, _ := parseInt(g[2])
			cur = &section{header: strings.TrimSpace(g[1]), level: lv}
			secs = append(secs, cur)
			continue
		}
		if cur != nil && cur.first == "" {
			if g := rowRE.FindStringSubmatch(line); g != nil && g[2] != "" {
				if f := firstSourceRE.FindStringSubmatch(g[2]); f != nil {
					cur.first = strings.TrimSpace(f[1])
				}
			}
		}
	}
	var out []store.SheetClass
	for _, s := range secs {
		lv := s.level
		c := store.SheetClass{Level: &lv}
		switch g := classWithSub.FindStringSubmatch(s.header); {
		case g != nil:
			c.Class, c.Subclass = g[1], g[2]
		case s.first != "" && !strings.EqualFold(s.first, s.header):
			c.Class, c.Subclass = s.first, s.header
		default:
			c.Class = s.header
		}
		out = append(out, c)
	}
	_ = subclassSplit
	return out
}

// featureResult is everything mapped from the sheet's feature text.
type featureResult struct {
	features  []store.Feature
	masteries []store.WeaponMastery
	free      []string
}

// featKey normalizes a feat name so "Alert [Origin]" and "Alert" match.
func featKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if i := strings.Index(name, " ["); i > 0 {
		name = name[:i]
	}
	return name
}

func (m *mapper) featureSection() featureResult {
	var res featureResult
	var classRows []store.Feature

	addRows := func(text, kind string) {
		rows, lead := parseRows(text)
		if lead != "" && kind == "other" {
			res.free = append(res.free, lead)
		}
		for _, r := range rows {
			if strings.EqualFold(r.Source, "Weapon Mastery") {
				res.masteries = append(res.masteries, store.WeaponMastery{Weapon: r.Name, Mastery: masteryName(r.Text), Notes: r.Text})
				continue
			}
			classRows = append(classRows, store.Feature{
				Kind: kind, Name: r.Name, Text: r.Text, Source: r.Source, Level: r.Level,
				Book: r.Book, Page: r.Page, UsesNote: r.UsesNote,
			})
		}
	}
	addRows(m.d.Get("Class Features"), "class")
	addRows(m.d.Get("Extra.Notes"), "class")
	for _, p := range m.prefixes("ASnotes") {
		addRows(m.d.Get(p+"Notes.Left"), "other")
		addRows(m.d.Get(p+"Notes.Right"), "other")
	}

	// Feats come from their own list; a notes row of the same name describes
	// the same feat, so the two merge into one, keeping the longer text.
	var feats []store.Feature
	byKey := map[string]int{}
	for _, n := range m.indices("Feat Name ", "") {
		i := itoa(n)
		name := m.get("Feat Name " + i)
		if name == "" {
			continue
		}
		f := store.Feature{Kind: "feat", Name: name, Text: m.text("Feat Description " + i)}
		f.Book, f.Page = parseNote(m.get("Feat Note " + i))
		byKey[featKey(name)] = len(feats)
		feats = append(feats, f)
	}
	var kept []store.Feature
	for _, f := range classRows {
		if i, ok := byKey[featKey(f.Name)]; ok {
			g := &feats[i]
			if len(f.Text) > len(g.Text) {
				g.Text = f.Text
			}
			if g.UsesNote == "" {
				g.UsesNote = f.UsesNote
			}
			if g.Level == nil {
				g.Level = f.Level
			}
			if g.Book == "" {
				g.Book, g.Page = f.Book, f.Page
			}
			continue
		}
		kept = append(kept, f)
	}

	res.features = append(res.features, kept...)
	res.features = append(res.features, m.speciesFeatures()...)
	res.features = append(res.features, feats...)
	return res
}

// parseNote reads a feat's "[P24 201]" source note into book and page.
func parseNote(s string) (book, page string) {
	// Notes can nest: "Human bonus feat [P24 200]" — the source is the last bracket.
	if i := strings.LastIndex(s, "["); i >= 0 {
		s = s[i+1:]
	}
	s = strings.Trim(s, "[] ")
	if sp := strings.LastIndex(s, " "); sp > 0 {
		if _, ok := parseInt(s[sp+1:]); ok {
			return strings.TrimSpace(s[:sp]), s[sp+1:]
		}
	}
	return s, ""
}

var masteryRE = regexp.MustCompile(`'([^']+)' Mastery`)

// masteryName reads the property from a mastery row's text: "I gain access to
// the Sling's 'Slow' Mastery feature" is Slow.
func masteryName(text string) string {
	if g := masteryRE.FindStringSubmatch(text); g != nil {
		return g[1]
	}
	return ""
}

// speciesFeatures splits Racial Traits on its bullets: "• Trance: text".
func (m *mapper) speciesFeatures() []store.Feature {
	var out []store.Feature
	for _, line := range splitLines(m.d.Get("Racial Traits")) {
		t := strings.TrimSpace(strings.ReplaceAll(line, " ", " "))
		if rest, ok := strings.CutPrefix(t, "•"); ok {
			name, text := strings.TrimSpace(rest), ""
			if i := strings.Index(rest, ":"); i > 0 && i <= 60 {
				name, text = strings.TrimSpace(rest[:i]), strings.TrimSpace(rest[i+1:])
			}
			out = append(out, store.Feature{Kind: "species", Name: name, Text: text})
			continue
		}
		if len(out) > 0 && t != "" {
			last := &out[len(out)-1]
			last.Text = strings.TrimSpace(last.Text + "\n" + t)
		}
	}
	return out
}
