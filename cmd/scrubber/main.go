package main

import (
	"encoding/json"
	"flag"
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

func main() {
	source := flag.String("source", "", "path to local 5etools repository root")
	edition := flag.String("edition", "", "target edition: \"5e\" or \"5.5e\"")
	flag.Parse()

	if *source == "" {
		log.Fatal("--source is required")
	}
	if *edition != "5e" && *edition != "5.5e" {
		log.Fatal("--edition must be \"5e\" or \"5.5e\"")
	}

	var baseURL string
	switch *edition {
	case "5e":
		baseURL = "https://2014.5e.tools/bestiary/"
	case "5.5e":
		baseURL = "https://5e.tools/bestiary/"
	}

	if err := store.Init(); err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	store.InitTypesense()

	bestiaryDir := filepath.Join(*source, "data", "bestiary")
	files, err := filepath.Glob(filepath.Join(bestiaryDir, "bestiary-*.json"))
	if err != nil || len(files) == 0 {
		log.Fatalf("no bestiary-*.json files found under %s", bestiaryDir)
	}

	// Collect all entries from all files.
	var direct []fiveEToolsEntry
	var pending []fiveEToolsEntry

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
		for _, entry := range bf.Monster {
			if entry.Name == "" || entry.Source == "" {
				continue
			}
			// Entries with their own hp go to pass 1; pure _copy entries are deferred.
			if entry.Copy == nil || (entry.HP != nil && entry.HP.Average > 0) {
				direct = append(direct, entry)
			} else {
				pending = append(pending, entry)
			}
		}
	}

	// resolvedStats holds hp+dex keyed by the entry's own (name, source) so
	// _copy children can inherit them.
	type stats struct{ hp, initMod int }
	resolvedStats := make(map[entryKey]stats, len(direct))

	var processed, inserted, updated int

	upsert := func(m store.Monster) {
		result, outcome, err := store.GlobalMonsters.UpsertMonster(m)
		if err != nil {
			log.Printf("warning: upsert failed for %s (%s): %v", m.Name, m.Edition, err)
			return
		}
		switch outcome {
		case store.UpsertSkippedCustomProtected:
			// Logged by UpsertMonster itself; not counted as processed.
			return
		case store.UpsertInserted:
			inserted++
		case store.UpsertUpdated:
			updated++
		}
		processed++
		resolvedStats[entryKey{result.Name, result.SourceBook}] = stats{result.MaxHP, derefOrZero(result.InitiativeModifier)}
	}

	normalize := func(name, source string, hp int, dex *int) store.Monster {
		initMod := 0
		if dex != nil {
			initMod = int(math.Floor(float64(*dex-10) / 2))
		} else {
			log.Printf("warning: %s has no dex — initiative_modifier defaulting to 0", name)
		}
		nameKebab := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		sourceLower := strings.ToLower(source)
		id := nameKebab + "-" + sourceLower
		mod := initMod
		return store.Monster{
			Name:               name,
			Edition:            *edition,
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
		m := normalize(e.Name, e.Source, e.HP.Average, e.Dex)
		upsert(m)
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

			// Use entry's own hp/dex if present, else inherit from base.
			hp := base.hp
			if e.HP != nil && e.HP.Average > 0 {
				hp = e.HP.Average
			}
			var dex *int
			if e.Dex != nil {
				dex = e.Dex
			}
			initMod := base.initMod
			if dex != nil {
				initMod = int(math.Floor(float64(*dex-10) / 2))
			}

			m := normalize(e.Name, e.Source, hp, dex)
			imod := initMod
			m.InitiativeModifier = &imod
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

func derefOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
