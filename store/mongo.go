package store

import (
	"context"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Profile is the persistent representation of a player or companion character.
type Profile struct {
	Name             string `bson:"name"              json:"name"`
	Type             string `bson:"type"              json:"type"` // "player" | "companion"
	MaxHP            int    `bson:"max_hp"            json:"max_hp"`
	ParentPCName     string `bson:"parent_pc_name"    json:"parent_pc_name,omitempty"`
	SharesInitiative bool   `bson:"shares_initiative" json:"shares_initiative"`
}

// Store exposes MongoDB operations used by handlers.
type Store struct {
	col *mongo.Collection
}

var Global Store

// Init connects to MongoDB and stores the collection reference in Global.
func Init() error {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:password@192.168.0.94:27017/combatapp?authSource=admin"
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}
	db := client.Database("combatapp")
	Global = Store{col: db.Collection("entities")}
	GlobalMonsters = MonsterStore{col: db.Collection("monsters")}
	return nil
}

// UpsertEntity inserts or replaces the profile document keyed by Name.
func (s *Store) UpsertEntity(p Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"name": p.Name}
	opts := options.Replace().SetUpsert(true)
	_, err := s.col.ReplaceOne(ctx, filter, p, opts)
	return err
}

// GetEntityByName returns the profile with the given name, or nil if not found.
func (s *Store) GetEntityByName(name string) (*Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p Profile
	err := s.col.FindOne(ctx, bson.M{"name": name}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetCompanionsByParent returns all companion profiles whose parent_pc_name matches.
func (s *Store) GetCompanionsByParent(parentName string) ([]Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := s.col.Find(ctx, bson.M{"type": "companion", "parent_pc_name": parentName})
	if err != nil {
		return nil, err
	}
	var results []Profile
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
