package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Monster struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"            json:"id,omitempty"`
	Name               string        `bson:"name"                     json:"name"`
	Edition            string        `bson:"edition"                  json:"edition"`
	MaxHP              int           `bson:"max_hp"                   json:"max_hp"`
	InitiativeModifier *int          `bson:"initiative_modifier,omitempty" json:"initiative_modifier,omitempty"`
	IsCustom           bool          `bson:"is_custom"                json:"is_custom"`
	SourceType         string        `bson:"source_type,omitempty"    json:"source_type,omitempty"`
	ReferenceURL       string        `bson:"reference_url,omitempty"  json:"reference_url,omitempty"`
	PDFObjectKey       string        `bson:"pdf_object_key,omitempty" json:"pdf_object_key,omitempty"`
	FiveEToolsID       string        `bson:"five_etools_id,omitempty" json:"five_etools_id,omitempty"`
	SourceBook         string        `bson:"source_book,omitempty"    json:"source_book,omitempty"`
}

type MonsterStore struct {
	col *mongo.Collection
}

var GlobalMonsters MonsterStore

// UpsertOutcome describes what UpsertMonster actually did.
type UpsertOutcome int

const (
	UpsertInserted UpsertOutcome = iota
	UpsertUpdated
)

// UpsertMonster writes m (an official, scrubber-sourced monster) to MongoDB
// keyed by {name, edition} and mirrors it into Typesense on a best-effort
// basis. It returns the resulting document (with its MongoDB id populated)
// whether m was inserted fresh or replaced an existing document.
//
// This collection only ever holds official monsters (is_custom: false);
// DM-authored monsters live in the separate custom_monsters collection
// (see CustomMonsterStore), so there is no cross-write collision to guard
// against here.
func (s *MonsterStore) UpsertMonster(m Monster) (Monster, UpsertOutcome, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"name": m.Name, "edition": m.Edition}

	outcome := UpsertUpdated
	count, err := s.col.CountDocuments(ctx, filter)
	if err != nil {
		return Monster{}, 0, err
	}
	if count == 0 {
		outcome = UpsertInserted
	}

	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)
	var result Monster
	if err := s.col.FindOneAndReplace(ctx, filter, m, opts).Decode(&result); err != nil {
		return Monster{}, 0, err
	}

	syncMonsterToTypesense(result)

	return result, outcome, nil
}

// SearchMonsters performs a typo-tolerant, prefix-matching, edition-filtered
// search against the Typesense monsters index, scoped to what requesterID is
// allowed to see (official monsters, public custom monsters, and their own
// private custom monsters). Returns an empty slice (not an error) if
// Typesense is unreachable.
func (s *MonsterStore) SearchMonsters(query, edition, requesterID string) ([]MonsterHit, error) {
	return searchTypesenseMonsters(query, edition, requesterID)
}

func (s *MonsterStore) GetMonsterByName(name string) (*Monster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var m Monster
	err := s.col.FindOne(ctx, bson.M{"name": name}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
