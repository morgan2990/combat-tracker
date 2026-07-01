package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EncounterMonster is one monster group within a saved encounter blueprint.
// For official monsters (IsCustom == false), Name identifies the monster and
// is resolved via GetMonsterByName at injection time; MonsterID is empty.
// For custom monsters (IsCustom == true), MonsterID identifies the custom
// monster document and is resolved via GetCustomMonsterByID; Name is a
// display label only, not used for resolution.
type EncounterMonster struct {
	Name        string `bson:"name"                    json:"name"`
	MonsterID   string `bson:"monster_id,omitempty"     json:"monster_id,omitempty"`
	IsCustom    bool   `bson:"is_custom"                json:"is_custom"`
	Quantity    int    `bson:"quantity"                 json:"quantity"`
	DisplayName string `bson:"display_name,omitempty"   json:"display_name,omitempty"`
}

// Encounter is a DM-authored, reusable combat blueprint: a named list of
// monster groups a DM can inject into any live room of the matching edition.
type Encounter struct {
	ID       string             `bson:"id"       json:"id"`
	Name     string             `bson:"name"     json:"name"`
	OwnerID  string             `bson:"owner_id" json:"owner_id"`
	Edition  string             `bson:"edition"  json:"edition"`
	Monsters []EncounterMonster `bson:"monsters" json:"monsters"`
}

// EncounterStore wraps the encounters MongoDB collection.
type EncounterStore struct {
	col *mongo.Collection
}

var GlobalEncounters EncounterStore

// CreateEncounter inserts a new encounter blueprint. If e.ID is already set
// it is used as-is; otherwise a new id is generated.
func (s *EncounterStore) CreateEncounter(e Encounter) (Encounter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e.ID == "" {
		e.ID = newID()
	}
	if _, err := s.col.InsertOne(ctx, e); err != nil {
		return Encounter{}, err
	}
	return e, nil
}

// GetEncounterByID returns the encounter with the given id, or nil if not found.
func (s *EncounterStore) GetEncounterByID(id string) (*Encounter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var e Encounter
	err := s.col.FindOne(ctx, bson.M{"id": id}).Decode(&e)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateEncounter replaces the fields of an existing encounter in place,
// keyed by id. Callers must verify ownership first, and must populate
// e.OwnerID (typically copied from the existing document) since this
// performs a full document replace.
func (s *EncounterStore) UpdateEncounter(id string, e Encounter) (Encounter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	e.ID = id
	opts := options.FindOneAndReplace().SetReturnDocument(options.After)
	var result Encounter
	if err := s.col.FindOneAndReplace(ctx, bson.M{"id": id}, e, opts).Decode(&result); err != nil {
		return Encounter{}, err
	}
	return result, nil
}

// DeleteEncounter removes the document by id.
func (s *EncounterStore) DeleteEncounter(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.DeleteOne(ctx, bson.M{"id": id})
	return err
}

// ListEncountersByOwner returns all encounters owned by ownerID, optionally
// filtered to a single edition when edition is non-empty.
func (s *EncounterStore) ListEncountersByOwner(ownerID string, edition string) ([]Encounter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"owner_id": ownerID}
	if edition != "" {
		filter["edition"] = edition
	}
	cursor, err := s.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var results []Encounter
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
