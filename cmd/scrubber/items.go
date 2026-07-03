package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"combatapp/store"
)

// fiveDotFiveCoreSourceBooks are the only source codes that represent 2024
// (5.5e) core-rulebook content. This is an explicit allowlist, not a prefix
// check, since other "X"-prefixed codes (e.g. XGE, Xanathar's Guide to
// Everything, 2017) are unrelated legacy sources — see design.md Decision 7.
var fiveDotFiveCoreSourceBooks = map[string]bool{
	"XPHB": true,
	"XDMG": true,
	"XMM":  true,
}

func itemEdition(source string) string {
	if fiveDotFiveCoreSourceBooks[source] {
		return "5.5e"
	}
	return "5e"
}

type fiveEToolsItemTypeEntry struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

// fiveEToolsItemEntry covers both items-base.json's baseitem[] and
// items.json's item[] — both shapes carry the same fields relevant to the
// catalog.
type fiveEToolsItemEntry struct {
	Name   string          `json:"name"`
	Source string          `json:"source"`
	Type   string          `json:"type"`
	Rarity string          `json:"rarity"`
	Copy   *fiveEToolsCopy `json:"_copy"`
}

type fiveEToolsItemsBaseFile struct {
	BaseItem []fiveEToolsItemEntry     `json:"baseitem"`
	ItemType []fiveEToolsItemTypeEntry `json:"itemType"`
}

type fiveEToolsItemsFile struct {
	Item []fiveEToolsItemEntry `json:"item"`
}

// stripTypeSourceSuffix removes a trailing "|SOURCE" from a type code (e.g.
// "AF|DMG" -> "AF") — some entries qualify their type with the sourcebook it
// was introduced in, but the itemType lookup table is keyed by the bare code.
func stripTypeSourceSuffix(typeCode string) string {
	before, _, _ := strings.Cut(typeCode, "|")
	return before
}

func buildCategoryLookup(entries []fiveEToolsItemTypeEntry) map[string]string {
	lookup := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.Abbreviation == "" {
			continue
		}
		lookup[e.Abbreviation] = e.Name
	}
	return lookup
}

func normalizeRarity(rarity string) string {
	if rarity == "none" {
		return ""
	}
	return rarity
}

// itemReferenceURL builds a hash-routed link into 5e.tools' items.html,
// matching the site's own routing (UrlUtil.encodeArrayForHash(name, source))
// rather than a static path — see design.md Decision 8.
func itemReferenceURL(name, source string) string {
	return "https://5e.tools/items.html#" + toURLHashPart(name) + "_" + toURLHashPart(source)
}

// toURLHashPart mirrors 5etools' String.prototype.toUrlified:
// encodeURIComponent(this.toLowerCase()).toLowerCase().
func toURLHashPart(s string) string {
	return strings.ToLower(encodeURIComponent(strings.ToLower(s)))
}

func encodeURIComponent(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreservedURIComponent(c) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// isUnreservedURIComponent matches the character set JS's encodeURIComponent
// leaves unescaped: A-Z a-z 0-9 - _ . ! ~ * ' ( )
func isUnreservedURIComponent(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '!' || c == '~' || c == '*' || c == '\'' || c == '(' || c == ')'
}

func itemFiveEToolsID(name, source string) string {
	nameKebab := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return nameKebab + "-" + strings.ToLower(source)
}

// runItemScrub parses items-base.json's baseitem[] and items.json's item[]
// from source (expected to be SCRUBBER_SOURCE_5_5E — see design.md Decision
// 7 for why only one checkout is scrubbed for items) and upserts each into
// the item catalog via the shared UpsertItem path.
func runItemScrub(source string) {
	if err := store.Init(); err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	store.InitTypesense()

	baseFile := filepath.Join(source, "data", "items-base.json")
	baseData, err := os.ReadFile(baseFile)
	if err != nil {
		log.Fatalf("could not read %s: %v", baseFile, err)
	}
	var basePayload fiveEToolsItemsBaseFile
	if err := json.Unmarshal(baseData, &basePayload); err != nil {
		log.Fatalf("could not parse %s: %v", baseFile, err)
	}

	itemsFile := filepath.Join(source, "data", "items.json")
	itemsData, err := os.ReadFile(itemsFile)
	if err != nil {
		log.Fatalf("could not read %s: %v", itemsFile, err)
	}
	var itemsPayload fiveEToolsItemsFile
	if err := json.Unmarshal(itemsData, &itemsPayload); err != nil {
		log.Fatalf("could not parse %s: %v", itemsFile, err)
	}

	categoryLookup := buildCategoryLookup(basePayload.ItemType)

	// resolvedFields holds each successfully-upserted entry's category/rarity,
	// keyed by (name, source), so _copy children can inherit whichever of the
	// two fields they don't specify themselves.
	type resolvedFields struct{ category, rarity string }
	resolved := make(map[entryKey]resolvedFields)

	var processed, inserted, updated int
	upsert := func(ci store.CatalogItem) {
		result, outcome, err := store.GlobalCatalogItems.UpsertItem(ci)
		if err != nil {
			log.Printf("warning: upsert failed for %s (%s): %v", ci.Name, ci.Edition, err)
			return
		}
		switch outcome {
		case store.UpsertInserted:
			inserted++
		case store.UpsertUpdated:
			updated++
		}
		processed++
		resolved[entryKey{result.Name, result.SourceBook}] = resolvedFields{result.Category, result.Rarity}
	}

	buildItem := func(name, entrySource, category, rarity string) store.CatalogItem {
		return store.CatalogItem{
			Name:         name,
			Edition:      itemEdition(entrySource),
			Category:     category,
			Rarity:       rarity,
			ReferenceURL: itemReferenceURL(name, entrySource),
			FiveEToolsID: itemFiveEToolsID(name, entrySource),
			SourceBook:   entrySource,
		}
	}

	var pending []fiveEToolsItemEntry

	for _, e := range basePayload.BaseItem {
		if e.Name == "" || e.Source == "" {
			continue
		}
		upsert(buildItem(e.Name, e.Source, categoryLookup[stripTypeSourceSuffix(e.Type)], normalizeRarity(e.Rarity)))
	}

	for _, e := range itemsPayload.Item {
		if e.Name == "" || e.Source == "" {
			continue
		}
		if e.Copy == nil {
			upsert(buildItem(e.Name, e.Source, categoryLookup[stripTypeSourceSuffix(e.Type)], normalizeRarity(e.Rarity)))
		} else {
			pending = append(pending, e)
		}
	}

	// Iteratively resolve _copy entries until no more progress, mirroring the
	// bestiary parser's two-pass _copy resolution.
	for len(pending) > 0 {
		var stillPending []fiveEToolsItemEntry
		resolvedThisRound := 0

		for _, e := range pending {
			baseKey := entryKey{e.Copy.Name, e.Copy.Source}
			base, ok := resolved[baseKey]
			if !ok {
				stillPending = append(stillPending, e)
				continue
			}

			category := categoryLookup[stripTypeSourceSuffix(e.Type)]
			if category == "" {
				category = base.category
			}
			rarity := normalizeRarity(e.Rarity)
			if rarity == "" {
				rarity = base.rarity
			}

			upsert(buildItem(e.Name, e.Source, category, rarity))
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

	fmt.Printf("Items done: %d processed, %d inserted, %d updated\n", processed, inserted, updated)
}
