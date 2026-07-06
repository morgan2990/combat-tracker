package main

import (
	"errors"
	"testing"

	"combatapp/store"
)

func withFakeMonsterStore(t *testing.T, fake *fakeMonsterUpserter) {
	t.Helper()
	original := monsterDB
	monsterDB = fake
	t.Cleanup(func() { monsterDB = original })
}

func TestMonsterBaseURL(t *testing.T) {
	cases := []struct {
		edition string
		want    string
		wantErr bool
	}{
		{"5e", "https://2014.5e.tools/bestiary/", false},
		{"5.5e", "https://5e.tools/bestiary/", false},
		{"", "", true},
		{"3.5e", "", true},
	}
	for _, c := range cases {
		got, err := monsterBaseURL(c.edition)
		if (err != nil) != c.wantErr {
			t.Errorf("monsterBaseURL(%q) error = %v, wantErr %v", c.edition, err, c.wantErr)
		}
		if got != c.want {
			t.Errorf("monsterBaseURL(%q) = %q, want %q", c.edition, got, c.want)
		}
	}
}

func TestInitiativeModifier(t *testing.T) {
	cases := []struct {
		dex  *int
		want int
	}{
		{nil, 0},
		{intPtr(10), 0},
		{intPtr(11), 0},
		{intPtr(20), 5},
		{intPtr(1), -5},
		{intPtr(8), -1},
	}
	for _, c := range cases {
		if got := initiativeModifier(c.dex); got != c.want {
			t.Errorf("initiativeModifier(%v) = %d, want %d", derefOrZero(c.dex), got, c.want)
		}
	}
}

func TestMonsterFiveEToolsID(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{"Goblin", "MM", "goblin-mm"},
		{"Ancient Red Dragon", "MM", "ancient-red-dragon-mm"},
		{"Tarrasque", "XMM", "tarrasque-xmm"},
	}
	for _, c := range cases {
		if got := monsterFiveEToolsID(c.name, c.source); got != c.want {
			t.Errorf("monsterFiveEToolsID(%q, %q) = %q, want %q", c.name, c.source, got, c.want)
		}
	}
}

func TestClassifyMonsterEntries(t *testing.T) {
	entries := []fiveEToolsEntry{
		{Name: "", Source: "MM", HP: &fiveEToolsHP{Average: 10}},         // dropped: no name
		{Name: "Goblin", Source: "", HP: &fiveEToolsHP{Average: 10}},     // dropped: no source
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}},    // direct: own hp
		{Name: "Hobgoblin", Source: "MM", HP: nil, Copy: nil},            // direct: no _copy at all
		{Name: "Goblin Boss", Source: "MM", HP: &fiveEToolsHP{Average: 21}, Copy: &fiveEToolsCopy{Name: "Goblin", Source: "MM"}}, // direct: _copy but has own hp
		{Name: "Goblin Variant", Source: "MM", Copy: &fiveEToolsCopy{Name: "Goblin", Source: "MM"}},                              // pending: pure _copy
	}

	direct, pending := classifyMonsterEntries(entries)

	if len(direct) != 3 {
		t.Fatalf("len(direct) = %d, want 3", len(direct))
	}
	if len(pending) != 1 {
		t.Fatalf("len(pending) = %d, want 1", len(pending))
	}
	if pending[0].Name != "Goblin Variant" {
		t.Errorf("pending[0].Name = %q, want %q", pending[0].Name, "Goblin Variant")
	}
}

func TestResolveCopyStats(t *testing.T) {
	base := monsterStats{hp: 7, initMod: 0}

	// Entry has no hp/dex of its own: inherits both from base.
	hp, dex, initMod := resolveCopyStats(fiveEToolsEntry{}, base)
	if hp != 7 || dex != nil || initMod != 0 {
		t.Errorf("resolveCopyStats(empty) = (%d, %v, %d), want (7, nil, 0)", hp, dex, initMod)
	}

	// Entry overrides hp but not dex.
	hp, dex, initMod = resolveCopyStats(fiveEToolsEntry{HP: &fiveEToolsHP{Average: 15}}, base)
	if hp != 15 || dex != nil || initMod != 0 {
		t.Errorf("resolveCopyStats(hp override) = (%d, %v, %d), want (15, nil, 0)", hp, dex, initMod)
	}

	// Entry overrides dex: initMod is recomputed from it, not inherited.
	d := 16
	hp, dex, initMod = resolveCopyStats(fiveEToolsEntry{Dex: &d}, base)
	if hp != 7 || dex == nil || *dex != 16 || initMod != 3 {
		t.Errorf("resolveCopyStats(dex override) = (%d, %v, %d), want (7, 16, 3)", hp, dex, initMod)
	}
}

func intPtr(i int) *int { return &i }

func TestScrubMonsterEntries_DirectEntry(t *testing.T) {
	fake := newFakeMonsterUpserter()
	withFakeMonsterStore(t, fake)

	dex := 14
	entries := []fiveEToolsEntry{
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}, Dex: &dex},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 1 || inserted != 1 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (1, 1, 0)", processed, inserted, updated)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(fake.calls))
	}
	got := fake.calls[0]
	if got.Name != "Goblin" || got.Edition != "5e" || got.MaxHP != 7 {
		t.Errorf("got monster = %+v, want Name=Goblin Edition=5e MaxHP=7", got)
	}
	if got.InitiativeModifier == nil || *got.InitiativeModifier != 2 {
		t.Errorf("InitiativeModifier = %v, want 2", got.InitiativeModifier)
	}
	if got.ReferenceURL != "https://2014.5e.tools/bestiary/goblin-mm.html" {
		t.Errorf("ReferenceURL = %q", got.ReferenceURL)
	}
	if got.FiveEToolsID != "goblin-mm" || got.SourceBook != "MM" || got.IsCustom || got.SourceType != "url" {
		t.Errorf("got monster = %+v", got)
	}
}

func TestScrubMonsterEntries_SkipsMissingHP(t *testing.T) {
	fake := newFakeMonsterUpserter()
	withFakeMonsterStore(t, fake)

	entries := []fiveEToolsEntry{
		{Name: "No HP", Source: "MM", HP: nil, Copy: nil},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 0 || inserted != 0 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want all 0", processed, inserted, updated)
	}
}

func TestScrubMonsterEntries_CopyInheritsHPAndDex(t *testing.T) {
	fake := newFakeMonsterUpserter()
	withFakeMonsterStore(t, fake)

	dex := 14
	entries := []fiveEToolsEntry{
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}, Dex: &dex},
		{Name: "Goblin Variant", Source: "MM", Copy: &fiveEToolsCopy{Name: "Goblin", Source: "MM"}},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 2 || inserted != 2 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (2, 2, 0)", processed, inserted, updated)
	}
	variant := fake.calls[1]
	if variant.MaxHP != 7 {
		t.Errorf("variant.MaxHP = %d, want 7 (inherited)", variant.MaxHP)
	}
	if variant.InitiativeModifier == nil || *variant.InitiativeModifier != 2 {
		t.Errorf("variant.InitiativeModifier = %v, want 2 (inherited)", variant.InitiativeModifier)
	}
}

func TestScrubMonsterEntries_CopyOverridesDex(t *testing.T) {
	fake := newFakeMonsterUpserter()
	withFakeMonsterStore(t, fake)

	baseDex, ownDex := 14, 20
	entries := []fiveEToolsEntry{
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}, Dex: &baseDex},
		{Name: "Fast Goblin", Source: "MM", Dex: &ownDex, Copy: &fiveEToolsCopy{Name: "Goblin", Source: "MM"}},
	}

	_, _, _ = scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	variant := fake.calls[1]
	if variant.MaxHP != 7 {
		t.Errorf("variant.MaxHP = %d, want 7 (inherited)", variant.MaxHP)
	}
	if variant.InitiativeModifier == nil || *variant.InitiativeModifier != 5 {
		t.Errorf("variant.InitiativeModifier = %v, want 5 (recomputed from own dex)", variant.InitiativeModifier)
	}
}

func TestScrubMonsterEntries_UnresolvableCopySkipped(t *testing.T) {
	fake := newFakeMonsterUpserter()
	withFakeMonsterStore(t, fake)

	entries := []fiveEToolsEntry{
		{Name: "Orphan", Source: "MM", Copy: &fiveEToolsCopy{Name: "Does Not Exist", Source: "MM"}},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 0 || inserted != 0 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want all 0", processed, inserted, updated)
	}
}

func TestScrubMonsterEntries_UpsertErrorSkipsButContinues(t *testing.T) {
	fake := newFakeMonsterUpserter()
	fake.forceErr = errors.New("boom")
	withFakeMonsterStore(t, fake)

	entries := []fiveEToolsEntry{
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 0 || inserted != 0 || updated != 0 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want all 0 on upsert error", processed, inserted, updated)
	}
}

func TestScrubMonsterEntries_ReportsUpdated(t *testing.T) {
	fake := newFakeMonsterUpserter()
	fake.byKey["Goblin|5e"] = store.Monster{Name: "Goblin", Edition: "5e", MaxHP: 1}
	withFakeMonsterStore(t, fake)

	entries := []fiveEToolsEntry{
		{Name: "Goblin", Source: "MM", HP: &fiveEToolsHP{Average: 7}},
	}

	processed, inserted, updated := scrubMonsterEntries(entries, "5e", "https://2014.5e.tools/bestiary/")

	if processed != 1 || inserted != 0 || updated != 1 {
		t.Fatalf("got (processed=%d, inserted=%d, updated=%d), want (1, 0, 1)", processed, inserted, updated)
	}
}
