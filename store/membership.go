package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// RoomMembership records that a user has joined a room as a player with a given PC.
// It grants no permission by itself; it is purely a recency record powering the
// frontend's "recent rooms" list.
type RoomMembership struct {
	UserID       string    `bson:"user_id"        json:"user_id"`
	RoomID       string    `bson:"room_id"        json:"room_id"`
	LastPCID     string    `bson:"last_pc_id"     json:"last_pc_id"`
	LastJoinedAt time.Time `bson:"last_joined_at" json:"last_joined_at"`
}

type MembershipStore struct {
	col *mongo.Collection
}

var GlobalMemberships MembershipStore

func ensureMembershipIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "room_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// Upsert records that userID joined roomID with pcID, updating last_joined_at.
func (s *MembershipStore) Upsert(userID, roomID, pcID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"user_id": userID, "room_id": roomID}
	update := bson.M{"$set": bson.M{
		"last_pc_id":     pcID,
		"last_joined_at": time.Now(),
	}}
	_, err := s.col.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	return err
}

// ListByUser returns all memberships for userID, ordered by last_joined_at descending.
func (s *MembershipStore) ListByUser(userID string) ([]RoomMembership, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "last_joined_at", Value: -1}})
	cursor, err := s.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	var results []RoomMembership
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
