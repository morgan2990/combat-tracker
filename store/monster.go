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
	Name         string `bson:"name"                     json:"name"`
	MaxHP        int    `bson:"max_hp"                   json:"max_hp"`
	SourceType   string `bson:"source_type,omitempty"    json:"source_type,omitempty"`
	ReferenceURL string `bson:"reference_url,omitempty"  json:"reference_url,omitempty"`
	PDFObjectKey string `bson:"pdf_object_key,omitempty" json:"pdf_object_key,omitempty"`
	FivEToolsID  string `bson:"five_etools_id,omitempty" json:"five_etools_id,omitempty"`
	SourceBook   string `bson:"source_book,omitempty"    json:"source_book,omitempty"`
}

type MonsterStore struct {
	col *mongo.Collection
}

var GlobalMonsters MonsterStore

func (s *MonsterStore) UpsertMonster(m Monster) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"name": m.Name}
	opts := options.Replace().SetUpsert(true)
	_, err := s.col.ReplaceOne(ctx, filter, m, opts)
	return err
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
