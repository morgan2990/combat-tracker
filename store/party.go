package store

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
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
	JoinCode    string   `bson:"join_code"     json:"join_code"`
	MemberPCIDs []string `bson:"member_pc_ids" json:"member_pc_ids"`
	Currency    Currency `bson:"currency"      json:"currency"`
}

// joinCodeChars mirrors room.idChars: uppercase letters and digits with
// visually ambiguous characters (I, O, 0, 1) removed.
const joinCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const joinCodeLength = 5
const maxJoinCodeAttempts = 5

func newPartyJoinCode() string {
	b := make([]byte, joinCodeLength)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(joinCodeChars))))
		b[i] = joinCodeChars[n.Int64()]
	}
	return string(b)
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
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "member_pc_ids", Value: 1}},
	}); err != nil {
		return err
	}
	// Sparse: parties created before join codes existed are lazily backfilled
	// on next read/write, so multiple docs may transiently lack the field.
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "join_code", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	return err
}

// CreateParty inserts a new party with a generated ID, a unique join code, no
// members, and zeroed pooled currency.
func (s *PartyStore) CreateParty(name string) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	joinCode, err := s.generateUniqueJoinCode(ctx)
	if err != nil {
		return nil, err
	}
	p := Party{
		ID:          NewID(8),
		Name:        name,
		JoinCode:    joinCode,
		MemberPCIDs: []string{},
	}
	if _, err := s.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// generateUniqueJoinCode generates a join code not already stored on another
// party, retrying on collision up to maxJoinCodeAttempts.
func (s *PartyStore) generateUniqueJoinCode(ctx context.Context) (string, error) {
	for range maxJoinCodeAttempts {
		code := newPartyJoinCode()
		count, err := s.col.CountDocuments(ctx, bson.M{"join_code": code}, options.Count().SetLimit(1))
		if err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not generate a unique party join code")
}

// ensureJoinCode backfills a join_code for a party predating this field,
// persisting the generated code and updating p in place. A no-op if p
// already has one.
func (s *PartyStore) ensureJoinCode(ctx context.Context, p *Party) error {
	if p.JoinCode != "" {
		return nil
	}
	code, err := s.generateUniqueJoinCode(ctx)
	if err != nil {
		return err
	}
	if _, err := s.col.UpdateOne(ctx, bson.M{"id": p.ID}, bson.M{"$set": bson.M{"join_code": code}}); err != nil {
		return err
	}
	p.JoinCode = code
	return nil
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
	if err := s.ensureJoinCode(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPartyByJoinCode returns the party with the given join code, or nil if
// no party has that code (including parties whose code hasn't been
// backfilled yet — see ensureJoinCode).
func (s *PartyStore) GetPartyByJoinCode(joinCode string) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p Party
	err := s.col.FindOne(ctx, bson.M{"join_code": joinCode}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// AddMember adds pcID to the party's member_pc_ids if not already present,
// returning the resulting document. Idempotent by construction ($addToSet).
func (s *PartyStore) AddMember(id, pcID string) (*Party, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var result Party
	err := s.col.FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$addToSet": bson.M{"member_pc_ids": pcID}}, opts).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
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
	if err := s.ensureJoinCode(ctx, &result); err != nil {
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
	for i := range results {
		if err := s.ensureJoinCode(ctx, &results[i]); err != nil {
			return nil, err
		}
	}
	return results, nil
}
