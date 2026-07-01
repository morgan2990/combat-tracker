package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// RoomEntitySnapshot is the persisted representation of a single entity within
// a room snapshot. It mirrors room.Entity field-for-field (store cannot import
// room — room already imports store) plus the persistence-only Connected field.
type RoomEntitySnapshot struct {
	ID                 string   `bson:"id"                             json:"id"`
	Name               string   `bson:"name"                           json:"name"`
	Type               string   `bson:"type"                           json:"type"`
	OwnerID            string   `bson:"owner_id,omitempty"             json:"owner_id,omitempty"`
	PCID               string   `bson:"pc_id,omitempty"                json:"pc_id,omitempty"`
	MaxHP              int      `bson:"max_hp"                         json:"max_hp"`
	CurrentHP          int      `bson:"current_hp"                     json:"current_hp"`
	TempHP             int      `bson:"temp_hp"                        json:"temp_hp"`
	Initiative         *int     `bson:"initiative"                     json:"initiative"`
	SharesInitiative   bool     `bson:"shares_initiative"              json:"shares_initiative"`
	Conditions         []string `bson:"conditions"                     json:"conditions"`
	Dead               bool     `bson:"dead"                           json:"dead"`
	SourceType         string   `bson:"source_type,omitempty"          json:"source_type,omitempty"`
	ReferenceURL       string   `bson:"reference_url,omitempty"        json:"reference_url,omitempty"`
	PDFObjectKey       string   `bson:"pdf_object_key,omitempty"       json:"pdf_object_key,omitempty"`
	InitiativeModifier *int     `bson:"initiative_modifier,omitempty"  json:"initiative_modifier,omitempty"`
	InitiativeRoll     *int     `bson:"initiative_roll,omitempty"      json:"initiative_roll,omitempty"`
	DisplayName        string   `bson:"display_name,omitempty"         json:"display_name,omitempty"`
	IsHidden           bool     `bson:"is_hidden"                      json:"is_hidden"`
	Connected          bool     `bson:"connected"                      json:"connected"`
}

// RoomSnapshot is the persisted representation of a room's combat state.
type RoomSnapshot struct {
	RoomID             string               `bson:"room_id"               json:"room_id"`
	OwnerUserID        string               `bson:"owner_user_id"         json:"owner_user_id"`
	IsCombatActive     bool                 `bson:"is_combat_active"      json:"is_combat_active"`
	CurrentRound       int                  `bson:"current_round"         json:"current_round"`
	ActiveTurnEntityID *string              `bson:"active_turn_entity_id" json:"active_turn_entity_id"`
	Edition            string               `bson:"edition"               json:"edition"`
	Entities           []RoomEntitySnapshot `bson:"entities"              json:"entities"`
}

// RoomStore exposes MongoDB operations for persisted room snapshots.
type RoomStore struct {
	col *mongo.Collection
}

var GlobalRooms RoomStore

func ensureRoomIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "room_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// SaveRoomSnapshot inserts or replaces the room document keyed by RoomID.
func (s *RoomStore) SaveRoomSnapshot(snap RoomSnapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"room_id": snap.RoomID}
	opts := options.Replace().SetUpsert(true)
	_, err := s.col.ReplaceOne(ctx, filter, snap, opts)
	return err
}

// GetRoomSnapshot returns the persisted snapshot for roomID, or nil if not found.
func (s *RoomStore) GetRoomSnapshot(roomID string) (*RoomSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var snap RoomSnapshot
	err := s.col.FindOne(ctx, bson.M{"room_id": roomID}).Decode(&snap)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// RoomSummary is a lightweight projection of a persisted room for dashboard listings.
type RoomSummary struct {
	RoomID         string `bson:"room_id"          json:"room_id"`
	Edition        string `bson:"edition"          json:"edition"`
	IsCombatActive bool   `bson:"is_combat_active" json:"is_combat_active"`
}

// ListByOwner returns a summary of every room owned by ownerUserID.
func (s *RoomStore) ListByOwner(ownerUserID string) ([]RoomSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	projection := bson.M{"room_id": 1, "edition": 1, "is_combat_active": 1}
	cursor, err := s.col.Find(ctx, bson.M{"owner_user_id": ownerUserID}, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	var results []RoomSummary
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
