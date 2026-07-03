package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CatalogItem is a 5etools-sourced equipment/magic-item catalog entry,
// distinct from Item (the free-text entry embedded on a PC's inventory).
// Unlike the monster catalog, there is no is_custom/private/owner_id: every
// document here always comes from the scrubber, so there is no custom-item
// concept to protect or filter by.
type CatalogItem struct {
	ID           bson.ObjectID `bson:"_id,omitempty"            json:"id,omitempty"`
	Name         string        `bson:"name"                     json:"name"`
	Edition      string        `bson:"edition"                  json:"edition"`
	Category     string        `bson:"category"                 json:"category"`
	Rarity       string        `bson:"rarity,omitempty"         json:"rarity,omitempty"`
	ReferenceURL string        `bson:"reference_url,omitempty"  json:"reference_url,omitempty"`
	FiveEToolsID string        `bson:"five_etools_id,omitempty" json:"five_etools_id,omitempty"`
	SourceBook   string        `bson:"source_book,omitempty"    json:"source_book,omitempty"`
}

// CatalogItemStore wraps the item catalog MongoDB collection.
type CatalogItemStore struct {
	col *mongo.Collection
}

var GlobalCatalogItems CatalogItemStore

// UpsertItem writes ci to MongoDB keyed by {name, edition} and mirrors it into
// Typesense on a best-effort basis, mirroring UpsertMonster exactly. It
// returns the resulting document (with its MongoDB id populated) whether ci
// was inserted fresh or replaced an existing document.
func (s *CatalogItemStore) UpsertItem(ci CatalogItem) (CatalogItem, UpsertOutcome, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"name": ci.Name, "edition": ci.Edition}

	outcome := UpsertUpdated
	count, err := s.col.CountDocuments(ctx, filter)
	if err != nil {
		return CatalogItem{}, 0, err
	}
	if count == 0 {
		outcome = UpsertInserted
	}

	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)
	var result CatalogItem
	if err := s.col.FindOneAndReplace(ctx, filter, ci, opts).Decode(&result); err != nil {
		return CatalogItem{}, 0, err
	}

	syncItemToTypesense(result)

	return result, outcome, nil
}

// SearchItems performs a typo-tolerant, prefix-matching, edition-filtered
// search against the Typesense items index. Returns an empty slice (not an
// error) if Typesense is unreachable.
func (s *CatalogItemStore) SearchItems(query, edition string) ([]CatalogItemHit, error) {
	return searchTypesenseItems(query, edition)
}

// GetItemByID returns the catalog item with the given hex id, or nil if not found.
func (s *CatalogItemStore) GetItemByID(id string) (*CatalogItem, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ci CatalogItem
	err = s.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&ci)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ci, nil
}
