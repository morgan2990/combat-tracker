// Package sheetdiff compares two character sheets as the clean model — never
// the raw PDF fields, some of which (like CurrentEvals) change on every save.
package sheetdiff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"combatapp/store"
)

// Kinds of change.
const (
	Added   = "added"
	Removed = "removed"
	Changed = "changed"
)

// Change is one difference. Path is empty for a whole-section change; list
// rows are addressed by their title, e.g. "attacks[Sting].to_hit".
type Change struct {
	Section string `json:"section"`
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Old     any    `json:"old,omitempty"`
	New     any    `json:"new,omitempty"`
}

// Input is one side of a comparison. Name and MaxHP live on the PC, not the
// sheet, but are part of what an import changes, so they are compared too.
type Input struct {
	Name  string
	MaxHP int
	Sheet store.CharacterSheet
}

// sectionOrder is the order sections are reported in.
var sectionOrder = []string{
	"identity", "abilities", "skills", "combat", "defenses", "attacks", "actions", "limited_uses",
	"features", "weapon_masteries", "proficiencies", "magic_items", "spellcasting", "wild_shapes", "notes",
}

// titleKeys are the fields that name a list row, tried in order.
var titleKeys = []string{"name", "weapon", "key", "class"}

// Sheets lists the differences from old to new.
func Sheets(old, new Input) []Change {
	o, n := toSections(old), toSections(new)
	var out []Change
	for _, sec := range sectionOrder {
		diff(&out, sec, "", o[sec], n[sec])
	}
	return out
}

// toSections renders a sheet as {section: value}, with empty values dropped
// so that "absent" and "empty" compare equal, and the PC's name and max HP
// folded into identity and combat where the sheet UI shows them.
func toSections(in Input) map[string]any {
	raw, _ := json.Marshal(in.Sheet)
	var all map[string]any
	_ = json.Unmarshal(raw, &all)

	out := map[string]any{}
	for _, sec := range sectionOrder {
		if v, ok := all[sec]; ok {
			out[sec] = v
		}
	}
	if in.Name != "" {
		withField(out, "identity", "name", in.Name)
	}
	if in.MaxHP > 0 {
		withField(out, "combat", "max_hp", float64(in.MaxHP))
	}
	for k, v := range out {
		out[k] = prune(v)
	}
	return out
}

func withField(sections map[string]any, section, key string, v any) {
	m, _ := sections[section].(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m[key] = v
	sections[section] = m
}

// prune drops nil, "", false, and empty maps and slices, recursively.
func prune(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, e := range x {
			if p := prune(e); p != nil {
				out[k] = p
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, e := range x {
			if p := prune(e); p != nil {
				out = append(out, p)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case string:
		if x == "" {
			return nil
		}
	case bool:
		if !x {
			return nil
		}
	}
	return v
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func diff(out *[]Change, sec, path string, old, new any) {
	if reflect.DeepEqual(old, new) {
		return
	}
	_, oldIsMap := old.(map[string]any)
	_, newIsMap := new.(map[string]any)
	switch {
	case old == nil && newIsMap:
		// A section or object that appears whole: report it field by field.
		diff(out, sec, path, map[string]any{}, new)
	case new == nil && oldIsMap:
		diff(out, sec, path, old, map[string]any{})
	case old == nil:
		added(out, sec, path, new)
	case new == nil:
		removed(out, sec, path, old)
	default:
		om, oIsMap := old.(map[string]any)
		nm, nIsMap := new.(map[string]any)
		ol, oIsList := old.([]any)
		nl, nIsList := new.([]any)
		switch {
		case oIsMap && nIsMap:
			keys := map[string]bool{}
			for k := range om {
				keys[k] = true
			}
			for k := range nm {
				keys[k] = true
			}
			sorted := make([]string, 0, len(keys))
			for k := range keys {
				sorted = append(sorted, k)
			}
			sort.Strings(sorted)
			for _, k := range sorted {
				diff(out, sec, join(path, k), om[k], nm[k])
			}
		case oIsList && nIsList:
			diffLists(out, sec, path, ol, nl)
		default:
			*out = append(*out, Change{Section: sec, Path: path, Kind: Changed, Old: old, New: new})
		}
	}
}

// added reports a value that only the new side has. A list of rows is
// reported row by row, so each row's title is visible.
func added(out *[]Change, sec, path string, v any) {
	if rows, ok := v.([]any); ok && allObjects(rows) {
		for i, r := range rows {
			*out = append(*out, Change{Section: sec, Path: rowPath(path, rowTitles(rows)[i], i), Kind: Added, New: r})
		}
		return
	}
	*out = append(*out, Change{Section: sec, Path: path, Kind: Added, New: v})
}

func removed(out *[]Change, sec, path string, v any) {
	if rows, ok := v.([]any); ok && allObjects(rows) {
		for i, r := range rows {
			*out = append(*out, Change{Section: sec, Path: rowPath(path, rowTitles(rows)[i], i), Kind: Removed, Old: r})
		}
		return
	}
	*out = append(*out, Change{Section: sec, Path: path, Kind: Removed, Old: v})
}

func allObjects(rows []any) bool {
	for _, r := range rows {
		if _, ok := r.(map[string]any); !ok {
			return false
		}
	}
	return len(rows) > 0
}

// rowTitles names each row by its title field; duplicates get a "#2" suffix so
// they stay distinct. A row with no title field is named by position.
func rowTitles(rows []any) []string {
	titles := make([]string, len(rows))
	count := map[string]int{}
	for i, r := range rows {
		m := r.(map[string]any)
		t := ""
		for _, k := range titleKeys {
			if s, ok := m[k].(string); ok && s != "" {
				t = s
				break
			}
		}
		if t == "" {
			titles[i] = ""
			continue
		}
		count[t]++
		if count[t] > 1 {
			t = fmt.Sprintf("%s#%d", t, count[t])
		}
		titles[i] = t
	}
	return titles
}

func rowPath(path, title string, index int) string {
	if title == "" {
		title = fmt.Sprintf("#%d", index+1)
	}
	return fmt.Sprintf("%s[%s]", path, title)
}

// diffLists compares two lists. Rows are paired by title when every row has
// one, so a reordered list is not a rewrite; otherwise by position.
func diffLists(out *[]Change, sec, path string, old, new []any) {
	if !allObjects(old) || !allObjects(new) {
		// Plain values (resistances, slot totals): compare as a whole.
		*out = append(*out, Change{Section: sec, Path: path, Kind: Changed, Old: old, New: new})
		return
	}
	ot, nt := rowTitles(old), rowTitles(new)
	titled := true
	for _, t := range append(append([]string{}, ot...), nt...) {
		if t == "" {
			titled = false
			break
		}
	}
	if !titled {
		for i := 0; i < len(old) || i < len(new); i++ {
			p := rowPath(path, "", i)
			switch {
			case i >= len(old):
				*out = append(*out, Change{Section: sec, Path: p, Kind: Added, New: new[i]})
			case i >= len(new):
				*out = append(*out, Change{Section: sec, Path: p, Kind: Removed, Old: old[i]})
			default:
				diff(out, sec, p, old[i], new[i])
			}
		}
		return
	}
	newByTitle := map[string]any{}
	for i, t := range nt {
		newByTitle[t] = new[i]
	}
	seen := map[string]bool{}
	for i, t := range ot {
		p := rowPath(path, t, i)
		if n, ok := newByTitle[t]; ok {
			seen[t] = true
			diff(out, sec, p, old[i], n)
		} else {
			*out = append(*out, Change{Section: sec, Path: p, Kind: Removed, Old: old[i]})
		}
	}
	for i, t := range nt {
		if !seen[t] {
			*out = append(*out, Change{Section: sec, Path: rowPath(path, t, i), Kind: Added, New: new[i]})
		}
	}
}

// --- companions and inventory ---

// CompanionChange lists the companions an import removes and creates. They are
// never matched: the old ones are deleted and the PDF's are created.
type CompanionChange struct {
	Removed []string `json:"removed"`
	Added   []string `json:"added"`
}

// Companions compares a character's current companions with the imported ones.
func Companions(current []store.PC, imported []store.ImportedCompanion) CompanionChange {
	c := CompanionChange{Removed: []string{}, Added: []string{}}
	for _, p := range current {
		c.Removed = append(c.Removed, p.Name)
	}
	for _, i := range imported {
		c.Added = append(c.Added, i.Name)
	}
	return c
}

// InventoryChange says what an import does to a PC's items and currency.
type InventoryChange struct {
	// Action is "replace" when the PDF carries an inventory, else "keep".
	Action string `json:"action"`
	// Changed is true when replacing would actually alter something.
	Changed  bool `json:"changed"`
	OldItems int  `json:"old_items"`
	NewItems int  `json:"new_items"`
}

// Inventory compares a PC's items and currency with the imported inventory.
func Inventory(items []store.Item, currency store.Currency, imported store.ImportedInventory) InventoryChange {
	c := InventoryChange{Action: "keep", OldItems: len(items), NewItems: len(items)}
	if !imported.Present {
		return c
	}
	c.Action = "replace"
	c.NewItems = len(imported.Items)
	c.Changed = currency != imported.Currency || !sameItems(items, imported.Items)
	return c
}

func sameItems(a, b []store.Item) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(items []store.Item) []string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = fmt.Sprintf("%s|%d", strings.ToLower(it.Name), it.Quantity)
		}
		sort.Strings(out)
		return out
	}
	return reflect.DeepEqual(key(a), key(b))
}
