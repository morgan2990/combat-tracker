package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
)

const monsterCollectionName = "monsters"

type typesenseStore struct {
	client *typesense.Client
}

var globalTypesense typesenseStore

// typesenseMonsterDoc is the document shape stored in the Typesense `monsters`
// collection — deliberately leaner than store.Monster (no statblock-reference
// fields); see design.md decision 6. Both official (store.Monster) and
// custom (store.CustomMonster) documents mirror into this same collection,
// distinguished by IsCustom.
type typesenseMonsterDoc struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MaxHP              int    `json:"max_hp"`
	InitiativeModifier *int   `json:"initiative_modifier,omitempty"`
	Edition            string `json:"edition"`
	IsCustom           bool   `json:"is_custom"`
	Private            bool   `json:"private,omitempty"`
	OwnerID            string `json:"owner_id,omitempty"`
	OwnerDisplayName   string `json:"owner_display_name,omitempty"`
}

// MonsterHit is the lightweight result shape returned by a Typesense search.
type MonsterHit struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MaxHP              int    `json:"max_hp"`
	InitiativeModifier *int   `json:"initiative_modifier,omitempty"`
	IsCustom           bool   `json:"is_custom"`
	OwnerDisplayName   string `json:"owner_display_name,omitempty"`
}

// InitTypesense connects to Typesense and ensures the monsters collection
// schema exists. Unlike store.Init() (MongoDB), failure here is logged, not
// fatal — Typesense is a best-effort search layer, not the source of truth.
func InitTypesense() {
	url := os.Getenv("TYPESENSE_URL")
	apiKey := os.Getenv("TYPESENSE_API_KEY")
	if url == "" || apiKey == "" {
		log.Printf("typesense: TYPESENSE_URL/TYPESENSE_API_KEY not fully set, skipping init")
		return
	}

	client := typesense.NewClient(
		typesense.WithServer(url),
		typesense.WithAPIKey(apiKey),
		typesense.WithConnectionTimeout(5*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Health(ctx, 5*time.Second); err != nil {
		log.Printf("typesense: init failed: %v", err)
		return
	}

	if err := ensureMonsterCollection(ctx, client); err != nil {
		log.Printf("typesense: schema init failed: %v", err)
		return
	}

	globalTypesense = typesenseStore{client: client}
}

func ensureMonsterCollection(ctx context.Context, client *typesense.Client) error {
	_, err := client.Collection(monsterCollectionName).Retrieve(ctx)
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
		Name: monsterCollectionName,
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string", Facet: &facet},
			{Name: "max_hp", Type: "int32"},
			{Name: "initiative_modifier", Type: "int32", Optional: &optional},
			{Name: "edition", Type: "string", Facet: &facet},
			{Name: "is_custom", Type: "bool", Facet: &facet},
			{Name: "private", Type: "bool", Optional: &optional},
			{Name: "owner_id", Type: "string", Optional: &optional},
			{Name: "owner_display_name", Type: "string", Optional: &optional},
		},
	}
	_, err = client.Collections().Create(ctx, schema)
	return err
}

// syncMonsterToTypesense best-effort mirrors a saved official Monster into
// the Typesense index. Failures are logged, never returned — MongoDB is the
// source of truth and a missed sync self-heals on the document's next write.
func syncMonsterToTypesense(m Monster) {
	upsertTypesenseDoc(typesenseMonsterDoc{
		ID:                 m.ID.Hex(),
		Name:               m.Name,
		MaxHP:              m.MaxHP,
		InitiativeModifier: m.InitiativeModifier,
		Edition:            m.Edition,
		IsCustom:           false,
	})
}

// syncCustomMonsterToTypesense best-effort mirrors a saved CustomMonster into
// the Typesense index, carrying its ownership/privacy fields.
func syncCustomMonsterToTypesense(m CustomMonster) {
	upsertTypesenseDoc(typesenseMonsterDoc{
		ID:                 m.ID,
		Name:               m.Name,
		MaxHP:              m.MaxHP,
		InitiativeModifier: m.InitiativeModifier,
		Edition:            m.Edition,
		IsCustom:           true,
		Private:            m.Private,
		OwnerID:            m.OwnerID,
		OwnerDisplayName:   m.OwnerDisplayName,
	})
}

func upsertTypesenseDoc(doc typesenseMonsterDoc) {
	if globalTypesense.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := globalTypesense.client.Collection(monsterCollectionName).Documents().Upsert(ctx, doc); err != nil {
		log.Printf("typesense: upsert failed for %q (%s): %v", doc.Name, doc.Edition, err)
	}
}

// deleteTypesenseDoc best-effort removes a document from the Typesense
// monsters collection by id. Failures are logged, never returned.
func deleteTypesenseDoc(id string) {
	if globalTypesense.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := globalTypesense.client.Collection(monsterCollectionName).Document(id).Delete(ctx); err != nil {
		log.Printf("typesense: delete failed for id %q: %v", id, err)
	}
}

// searchTypesenseMonsters queries the Typesense monsters collection with
// typo tolerance and prefix matching on name, filtered to the given edition
// and scoped to what requesterID is allowed to see: official monsters,
// public custom monsters, or the requester's own private custom monsters.
// Returns an empty slice (not an error) if Typesense is unreachable.
func searchTypesenseMonsters(query, edition, requesterID string) ([]MonsterHit, error) {
	if globalTypesense.client == nil {
		return []MonsterHit{}, nil
	}

	queryBy := "name"
	filterBy := fmt.Sprintf(
		"edition:=%s && (is_custom:=false || (is_custom:=true && private:=false) || (is_custom:=true && private:=true && owner_id:=%s))",
		edition, requesterID,
	)
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
	result, err := globalTypesense.client.Collection(monsterCollectionName).Documents().Search(ctx, params)
	if err != nil {
		log.Printf("typesense: search failed for %q (%s): %v", query, edition, err)
		return []MonsterHit{}, nil
	}
	if result.Hits == nil {
		return []MonsterHit{}, nil
	}

	hits := make([]MonsterHit, 0, len(*result.Hits))
	for _, h := range *result.Hits {
		if h.Document == nil {
			continue
		}
		raw, err := json.Marshal(*h.Document)
		if err != nil {
			continue
		}
		var doc typesenseMonsterDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		hits = append(hits, MonsterHit{
			ID:                 doc.ID,
			Name:               doc.Name,
			MaxHP:              doc.MaxHP,
			InitiativeModifier: doc.InitiativeModifier,
			IsCustom:           doc.IsCustom,
			OwnerDisplayName:   doc.OwnerDisplayName,
		})
	}
	return hits, nil
}
