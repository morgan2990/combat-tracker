package mpmb

import "combatapp/store"

// docOf builds a Document from name/value pairs, without going through a PDF.
func docOf(pairs ...string) *Document {
	d := &Document{Info: Info{Version: "14.0.8"}, byName: map[string]int{}}
	for i := 0; i+1 < len(pairs); i += 2 {
		d.add(store.RawField{Name: pairs[i], Value: pairs[i+1]})
	}
	return d
}

func (d *Document) add(f store.RawField) *Document {
	d.byName[f.Name] = len(d.Fields)
	d.Fields = append(d.Fields, f)
	return d
}

// withTM adds a field that carries a /TM mapping name.
func (d *Document) withTM(name, value, tm string) *Document {
	return d.add(store.RawField{Name: name, Value: value, TM: tm})
}

// withDisplay adds a choice field with its selected label.
func (d *Document) withDisplay(name, value, display string) *Document {
	return d.add(store.RawField{Name: name, Value: value, Display: display, Type: "/Ch"})
}

// with adds name/value pairs to an existing document.
func (d *Document) with(pairs ...string) *Document {
	for i := 0; i+1 < len(pairs); i += 2 {
		d.add(store.RawField{Name: pairs[i], Value: pairs[i+1]})
	}
	return d
}
