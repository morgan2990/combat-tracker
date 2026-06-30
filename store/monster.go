package store

import (
	"context"
	"errors"
	"log"
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

// shouldPreserveCustom reports whether a non-custom write should be skipped
// to avoid overwriting an existing DM-customized document.
func shouldPreserveCustom(existingIsCustom, incomingIsCustom bool) bool {
	return existingIsCustom && !incomingIsCustom
}

// UpsertOutcome describes what UpsertMonster actually did.
type UpsertOutcome int

const (
	UpsertInserted UpsertOutcome = iota
	UpsertUpdated
	UpsertSkippedCustomProtected
)

// UpsertMonster writes m to MongoDB keyed by {name, edition} and mirrors it
// into Typesense on a best-effort basis. It returns the resulting document
// (with its MongoDB id populated) whether m was inserted fresh or replaced
// an existing document.
//
// A non-custom write (m.IsCustom == false, i.e. scrubber-sourced) targeting
// a {name, edition} pair whose existing document is already custom
// (IsCustom == true) is skipped to preserve the DM's customization; the
// existing document is returned unchanged with UpsertSkippedCustomProtected.
func (s *MonsterStore) UpsertMonster(m Monster) (Monster, UpsertOutcome, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"name": m.Name, "edition": m.Edition}

	outcome := UpsertUpdated
	if !m.IsCustom {
		var existing Monster
		err := s.col.FindOne(ctx, filter).Decode(&existing)
		switch {
		case err == nil && shouldPreserveCustom(existing.IsCustom, m.IsCustom):
			log.Printf("monster upsert skipped: %q (%s) is custom, preserving over non-custom write", m.Name, m.Edition)
			return existing, UpsertSkippedCustomProtected, nil
		case errors.Is(err, mongo.ErrNoDocuments):
			outcome = UpsertInserted
		case err != nil:
			return Monster{}, 0, err
		}
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
// search against the Typesense monsters index. Returns an empty slice (not
// an error) if Typesense is unreachable.
func (s *MonsterStore) SearchMonsters(query, edition string) ([]MonsterHit, error) {
	return searchTypesenseMonsters(query, edition)
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
