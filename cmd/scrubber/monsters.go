package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"combatapp/store"
)

type fiveEToolsHP struct {
	Average int `json:"average"`
}

// fiveEToolsCopy is the "_copy" reference shared by monster and item entries
// alike, pointing at the (name, source) of the entry to inherit fields from.
type fiveEToolsCopy struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type fiveEToolsEntry struct {
	Name   string          `json:"name"`
	Source string          `json:"source"`
	HP     *fiveEToolsHP   `json:"hp"`
	Dex    *int            `json:"dex"`
	Copy   *fiveEToolsCopy `json:"_copy"`
}

type fiveEToolsFile struct {
	Monster []fiveEToolsEntry `json:"monster"`
}

type entryKey struct{ Name, Source string }

// monsterStats holds hp+dex-derived initiative modifier keyed by a resolved
// entry's own (name, source), so _copy children can inherit them.
type monsterStats struct{ hp, initMod int }

// monsterBaseURL returns the bestiary base URL 5etools hash-routes monster
// pages under for the given edition.
func monsterBaseURL(edition string) (string, error) {
	switch edition {
	case "5e":
		return "https://2014.5e.tools/bestiary/", nil
	case "5.5e":
		return "https://5e.tools/bestiary/", nil
	default:
		return "", fmt.Errorf(`--edition must be "5e" or "5.5e", got %q`, edition)
	}
}

// initiativeModifier converts a dex score into a D&D ability modifier
// (floor((dex-10)/2)), defaulting to 0 when dex is unknown.
func initiativeModifier(dex *int) int {
	if dex == nil {
		return 0
	}
	return int(math.Floor(float64(*dex-10) / 2))
}

func monsterFiveEToolsID(name, source string) string {
	nameKebab := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return nameKebab + "-" + strings.ToLower(source)
}

// classifyMonsterEntries splits raw bestiary entries into those that carry
// their own hp (resolvable in pass 1) and pure _copy entries that must wait
// for their base to resolve first (pass 2).
func classifyMonsterEntries(entries []fiveEToolsEntry) (direct, pending []fiveEToolsEntry) {
	for _, entry := range entries {
		if entry.Name == "" || entry.Source == "" {
			continue
		}
		if entry.Copy == nil || (entry.HP != nil && entry.HP.Average > 0) {
			direct = append(direct, entry)
		} else {
			pending = append(pending, entry)
		}
	}
	return direct, pending
}

// resolveCopyStats computes the hp/dex/initiative-modifier a _copy entry
// should use, inheriting from its already-resolved base for any field the
// entry does not specify itself.
func resolveCopyStats(e fiveEToolsEntry, base monsterStats) (hp int, dex *int, initMod int) {
	hp = base.hp
	if e.HP != nil && e.HP.Average > 0 {
		hp = e.HP.Average
	}
	dex = e.Dex
	initMod = base.initMod
	if dex != nil {
		initMod = initiativeModifier(dex)
	}
	return hp, dex, initMod
}

// loadBestiaryEntries reads and concatenates the monster[] arrays out of
// every bestiary-*.json file under bestiaryDir.
func loadBestiaryEntries(bestiaryDir string) ([]fiveEToolsEntry, error) {
	files, err := filepath.Glob(filepath.Join(bestiaryDir, "bestiary-*.json"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no bestiary-*.json files found under %s", bestiaryDir)
	}

	var all []fiveEToolsEntry
	for _, fpath := range files {
		data, err := os.ReadFile(fpath)
		if err != nil {
			log.Printf("warning: could not read %s: %v — skipping", fpath, err)
			continue
		}
		var bf fiveEToolsFile
		if err := json.Unmarshal(data, &bf); err != nil {
			log.Printf("warning: could not parse %s: %v — skipping", fpath, err)
			continue
		}
		all = append(all, bf.Monster...)
	}
	return all, nil
}

func derefOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// runMonsterScrub parses every bestiary-*.json file under source (expected
// to be the local 5etools checkout for edition) and upserts each monster
// into the global monster catalog via the shared UpsertMonster path.
func runMonsterScrub(source, edition string) {
	baseURL, err := monsterBaseURL(edition)
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	store.InitTypesense()

	bestiaryDir := filepath.Join(source, "data", "bestiary")
	entries, err := loadBestiaryEntries(bestiaryDir)
	if err != nil {
		log.Fatal(err)
	}
	direct, pending := classifyMonsterEntries(entries)

	// resolvedStats holds hp+dex keyed by the entry's own (name, source) so
	// _copy children can inherit them.
	resolvedStats := make(map[entryKey]monsterStats, len(direct))

	var processed, inserted, updated int

	upsert := func(m store.Monster) {
		result, outcome, err := store.GlobalMonsters.UpsertMonster(m)
		if err != nil {
			log.Printf("warning: upsert failed for %s (%s): %v", m.Name, m.Edition, err)
			return
		}
		switch outcome {
		case store.UpsertInserted:
			inserted++
		case store.UpsertUpdated:
			updated++
		}
		processed++
		resolvedStats[entryKey{result.Name, result.SourceBook}] = monsterStats{result.MaxHP, derefOrZero(result.InitiativeModifier)}
	}

	normalize := func(name, source string, hp int, dex *int) store.Monster {
		if dex == nil {
			log.Printf("warning: %s has no dex — initiative_modifier defaulting to 0", name)
		}
		mod := initiativeModifier(dex)
		id := monsterFiveEToolsID(name, source)
		return store.Monster{
			Name:               name,
			Edition:            edition,
			MaxHP:              hp,
			InitiativeModifier: &mod,
			IsCustom:           false,
			SourceType:         "url",
			ReferenceURL:       baseURL + id + ".html",
			FiveEToolsID:       id,
			SourceBook:         source,
		}
	}

	// Pass 1: entries with their own hp.
	for _, e := range direct {
		if e.HP == nil || e.HP.Average <= 0 {
			log.Printf("warning: %s has no hp.average — skipping", e.Name)
			continue
		}
		upsert(normalize(e.Name, e.Source, e.HP.Average, e.Dex))
	}

	// Pass 2: iteratively resolve _copy entries until no more progress.
	for len(pending) > 0 {
		var stillPending []fiveEToolsEntry
		resolvedThisRound := 0

		for _, e := range pending {
			baseKey := entryKey{e.Copy.Name, e.Copy.Source}
			base, ok := resolvedStats[baseKey]
			if !ok {
				stillPending = append(stillPending, e)
				continue
			}

			hp, dex, initMod := resolveCopyStats(e, base)
			m := normalize(e.Name, e.Source, hp, dex)
			mod := initMod
			m.InitiativeModifier = &mod
			upsert(m)
			resolvedThisRound++
		}

		if resolvedThisRound == 0 {
			for _, e := range stillPending {
				log.Printf("warning: cannot resolve _copy base %q (%s) for %q — skipping",
					e.Copy.Name, e.Copy.Source, e.Name)
			}
			break
		}
		pending = stillPending
	}

	fmt.Printf("Done: %d processed, %d inserted, %d updated\n", processed, inserted, updated)
}
