package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
)

const itemCollectionName = "items"

// typesenseItemDoc is the document shape stored in the Typesense `items`
// collection — deliberately leaner than store.CatalogItem (no reference_url/
// five_etools_id/source_book); see design.md decision 3. Unlike the monsters
// collection, there is no is_custom/private/owner_id — the item catalog has
// no custom-item or ownership concept.
type typesenseItemDoc struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Edition  string `json:"edition"`
	Category string `json:"category"`
	Rarity   string `json:"rarity,omitempty"`
}

// CatalogItemHit is the lightweight result shape returned by a Typesense search.
type CatalogItemHit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Rarity   string `json:"rarity,omitempty"`
}

// ensureItemCollection creates the Typesense `items` collection if it doesn't
// already exist. Called alongside ensureMonsterCollection during
// InitTypesense; items do not introduce a second Typesense connection.
func ensureItemCollection(ctx context.Context, client *typesense.Client) error {
	_, err := client.Collection(itemCollectionName).Retrieve(ctx)
	if err == nil {
		return nil
	}
	var httpErr *typesense.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 404 {
		return err
	}

	facet := true
	optional := true
	schema := &api.CollectionSchema{
		Name: itemCollectionName,
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string", Facet: &facet},
			{Name: "edition", Type: "string", Facet: &facet},
			{Name: "category", Type: "string", Facet: &facet},
			{Name: "rarity", Type: "string", Facet: &facet, Optional: &optional},
		},
	}
	_, err = client.Collections().Create(ctx, schema)
	return err
}

// syncItemToTypesense best-effort mirrors a saved CatalogItem into the
// Typesense index. Failures are logged, never returned — MongoDB is the
// source of truth and a missed sync self-heals on the document's next write.
func syncItemToTypesense(ci CatalogItem) {
	if globalTypesense.client == nil {
		return
	}
	doc := typesenseItemDoc{
		ID:       ci.ID.Hex(),
		Name:     ci.Name,
		Edition:  ci.Edition,
		Category: ci.Category,
		Rarity:   ci.Rarity,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := globalTypesense.client.Collection(itemCollectionName).Documents().Upsert(ctx, doc); err != nil {
		log.Printf("typesense: item upsert failed for %q (%s): %v", doc.Name, doc.Edition, err)
	}
}

// searchTypesenseItems queries the Typesense items collection with typo
// tolerance and prefix matching on name, filtered to the given edition.
// Returns an empty slice (not an error) if Typesense is unreachable.
func searchTypesenseItems(query, edition string) ([]CatalogItemHit, error) {
	if globalTypesense.client == nil {
		return []CatalogItemHit{}, nil
	}

	queryBy := "name"
	filterBy := fmt.Sprintf("edition:=%s", edition)
	numTypos := "2"
	prefix := "true"
	perPage := 10
	params := &api.SearchCollectionParams{
		Q:        &query,
		QueryBy:  &queryBy,
		FilterBy: &filterBy,
		NumTypos: &numTypos,
		Prefix:   &prefix,
		PerPage:  &perPage,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := globalTypesense.client.Collection(itemCollectionName).Documents().Search(ctx, params)
	if err != nil {
		log.Printf("typesense: item search failed for %q (%s): %v", query, edition, err)
		return []CatalogItemHit{}, nil
	}
	if result.Hits == nil {
		return []CatalogItemHit{}, nil
	}

	hits := make([]CatalogItemHit, 0, len(*result.Hits))
	for _, h := range *result.Hits {
		if h.Document == nil {
			continue
		}
		raw, err := json.Marshal(*h.Document)
		if err != nil {
			continue
		}
		var doc typesenseItemDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		hits = append(hits, CatalogItemHit{
			ID:       doc.ID,
			Name:     doc.Name,
			Category: doc.Category,
			Rarity:   doc.Rarity,
		})
	}
	return hits, nil
}
