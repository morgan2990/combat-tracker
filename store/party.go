package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Party is a standalone, user-agnostic named container grouping PCs
// (potentially owned by different users) around a single pooled currency.
// Personal items always stay attached to the owning PC — a Party never
// holds its own item list.
type Party struct {
	ID          string   `bson:"id"            json:"id"`
	Name        string   `bson:"name"          json:"name"`
	MemberPCIDs []string `bson:"member_pc_ids" json:"member_pc_ids"`
	Currency    Currency `bson:"currency"      json:"currency"`
}

// PartyStore wraps the parties MongoDB collection.
type PartyStore struct {
	col *mongo.Collection
}

var GlobalParties PartyStore

func ensurePartyIndex(ctx context.Context, col *mongo.Collection) error {
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// Backs ListPartiesByMemberPCIDs' $in query on member_pc_ids.
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "member_pc_ids", Value: 1}},
	})
	return err
}

// CreateParty inserts a new party with a generated ID, no members, and zeroed pooled currency.
func (s *PartyStore) CreateParty(name string) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := Party{
		ID:          NewID(8),
		Name:        name,
		MemberPCIDs: []string{},
	}
	if _, err := s.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPartyByID returns the party with the given id, or nil if not found.
func (s *PartyStore) GetPartyByID(id string) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p Party
	err := s.col.FindOne(ctx, bson.M{"id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateParty overwrites a party's membership and pooled currency, returning the
// updated document. Callers must verify the requester owns a member PC (or that
// the party currently has no members) before calling this.
func (s *PartyStore) UpdateParty(id string, memberPCIDs []string, currency Currency) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if memberPCIDs == nil {
		memberPCIDs = []string{}
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var result Party
	err := s.col.FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{
		"member_pc_ids": memberPCIDs,
		"currency":      currency,
	}}, opts).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPartiesByMemberPCIDs returns all parties whose member_pc_ids includes at
// least one of pcIDs — used to find a user's party memberships via their PCs.
func (s *PartyStore) ListPartiesByMemberPCIDs(pcIDs []string) ([]Party, error) {
	if len(pcIDs) == 0 {
		return []Party{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := s.col.Find(ctx, bson.M{"member_pc_ids": bson.M{"$in": pcIDs}})
	if err != nil {
		return nil, err
	}
	var results []Party
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
