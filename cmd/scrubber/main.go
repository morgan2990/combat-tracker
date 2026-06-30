package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

type Monster struct {
	Name               string `bson:"name"                     json:"name"`
	Edition            string `bson:"edition"                  json:"edition"`
	MaxHP              int    `bson:"max_hp"                   json:"max_hp"`
	InitiativeModifier *int   `bson:"initiative_modifier,omitempty" json:"initiative_modifier,omitempty"`
	IsCustom           bool   `bson:"is_custom"                json:"is_custom"`
	SourceType         string `bson:"source_type,omitempty"    json:"source_type,omitempty"`
	ReferenceURL       string `bson:"reference_url,omitempty"  json:"reference_url,omitempty"`
	FiveEToolsID       string `bson:"five_etools_id,omitempty" json:"five_etools_id,omitempty"`
	SourceBook         string `bson:"source_book,omitempty"    json:"source_book,omitempty"`
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

	col, err := connectMongo()
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

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

	upsert := func(m Monster) {
		filter := bson.M{"name": m.Name, "edition": m.Edition}
		opts := options.Replace().SetUpsert(true)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := col.ReplaceOne(ctx, filter, m, opts)
		cancel()
		if err != nil {
			log.Printf("warning: upsert failed for %s (%s): %v", m.Name, m.Edition, err)
			return
		}
		processed++
		if result.UpsertedCount > 0 {
			inserted++
		} else {
			updated++
		}
	}

	normalize := func(name, source string, hp int, dex *int) Monster {
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
		return Monster{
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
		resolvedStats[entryKey{e.Name, e.Source}] = stats{m.MaxHP, *m.InitiativeModifier}
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
			resolvedStats[entryKey{e.Name, e.Source}] = stats{m.MaxHP, *m.InitiativeModifier}
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

func connectMongo() (*mongo.Collection, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:password@192.168.0.94:27017/combatapp?authSource=admin"
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client.Database("combatapp").Collection("monsters"), nil
}
