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

// Item is one entry in a PC's personal item list.
type Item struct {
	Name     string `bson:"name"     json:"name"`
	Quantity int    `bson:"quantity" json:"quantity"`
}

// Currency is a 5e-style coin purse. Denominations are stored and edited
// independently — the system never auto-converts between them.
type Currency struct {
	PP int `bson:"pp" json:"pp"`
	GP int `bson:"gp" json:"gp"`
	EP int `bson:"ep" json:"ep"`
	SP int `bson:"sp" json:"sp"`
	CP int `bson:"cp" json:"cp"`
}

// IsNegative reports whether any denomination is below zero.
func (c Currency) IsNegative() bool {
	return c.PP < 0 || c.GP < 0 || c.EP < 0 || c.SP < 0 || c.CP < 0
}

// PC is the persistent representation of a player character or companion, owned by a User.
// name is a display label only — it is not required to be unique, globally or per-owner.
type PC struct {
	ID               string   `bson:"id"                json:"id"`
	OwnerUserID      string   `bson:"owner_user_id"     json:"owner_user_id"`
	Name             string   `bson:"name"              json:"name"`
	Type             string   `bson:"type"              json:"type"` // "pc" | "companion"
	MaxHP            int      `bson:"max_hp"            json:"max_hp"`
	ParentPCID       string   `bson:"parent_pc_id"      json:"parent_pc_id,omitempty"`
	SharesInitiative bool     `bson:"shares_initiative" json:"shares_initiative"`
	Items            []Item   `bson:"items"             json:"items"`
	Currency         Currency `bson:"currency"          json:"currency"`
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
		return errors.New("MONGODB_URI not set")
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
	Global = Store{col: db.Collection("pcs")}
	if err := ensurePCIndex(ctx, Global.col); err != nil {
		return err
	}
	monstersCol := db.Collection("monsters")
	GlobalMonsters = MonsterStore{col: monstersCol}
	if err := ensureMonsterIndex(ctx, monstersCol); err != nil {
		return err
	}
	customMonstersCol := db.Collection("custom_monsters")
	GlobalCustomMonsters = CustomMonsterStore{col: customMonstersCol}
	if err := ensureCustomMonsterIndex(ctx, customMonstersCol); err != nil {
		return err
	}
	roomsCol := db.Collection("rooms")
	GlobalRooms = RoomStore{col: roomsCol}
	if err := ensureRoomIndex(ctx, roomsCol); err != nil {
		return err
	}
	usersCol := db.Collection("users")
	sessionsCol := db.Collection("sessions")
	GlobalUsers = UserStore{users: usersCol, sessions: sessionsCol}
	if err := ensureUserIndexes(ctx, usersCol, sessionsCol); err != nil {
		return err
	}
	membershipsCol := db.Collection("room_memberships")
	GlobalMemberships = MembershipStore{col: membershipsCol}
	if err := ensureMembershipIndex(ctx, membershipsCol); err != nil {
		return err
	}
	encountersCol := db.Collection("encounters")
	GlobalEncounters = EncounterStore{col: encountersCol}
	if err := ensureEncounterIndex(ctx, encountersCol); err != nil {
		return err
	}
	partiesCol := db.Collection("parties")
	GlobalParties = PartyStore{col: partiesCol}
	if err := ensurePartyIndex(ctx, partiesCol); err != nil {
		return err
	}
	return nil
}

func ensurePCIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// ensureCustomMonsterIndex ensures a unique index on id (no index on
// {name, edition} — unlike official monsters, custom monster names are not
// unique across owners, so no such constraint should exist here).
func ensureCustomMonsterIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// ensureEncounterIndex ensures a unique index on id.
func ensureEncounterIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func ensureMonsterIndex(ctx context.Context, col *mongo.Collection) error {
	iv := col.Indexes()
	cur, err := iv.List(ctx)
	if err != nil {
		return err
	}
	var indexes []bson.M
	if err := cur.All(ctx, &indexes); err != nil {
		return err
	}
	for _, idx := range indexes {
		key, _ := idx["key"].(bson.M)
		_, hasName := key["name"]
		_, hasEdition := key["edition"]
		unique, _ := idx["unique"].(bool)
		if hasName && !hasEdition && unique {
			name, _ := idx["name"].(string)
			if err := iv.DropOne(ctx, name); err != nil {
				return err
			}
			break
		}
	}
	_, err = iv.CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}, {Key: "edition", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// CreatePC inserts a new owned PC with a generated ID.
func (s *Store) CreatePC(ownerUserID, name string, maxHP int) (*PC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := PC{
		ID:          NewID(8),
		OwnerUserID: ownerUserID,
		Name:        name,
		Type:        "pc",
		MaxHP:       maxHP,
		Items:       []Item{},
	}
	if _, err := s.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePC overwrites the editable fields of an existing PC. Callers must verify ownership first.
func (s *Store) UpdatePC(id, name string, maxHP int, items []Item, currency Currency) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if items == nil {
		items = []Item{}
	}
	_, err := s.col.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{
		"name":     name,
		"max_hp":   maxHP,
		"items":    items,
		"currency": currency,
	}})
	return err
}

// GetPCByID returns the PC (or companion) with the given id, or nil if not found.
func (s *Store) GetPCByID(id string) (*PC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p PC
	err := s.col.FindOne(ctx, bson.M{"id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateCompanion inserts a new companion linked to parentPCID. Callers must verify
// that the parent PC belongs to the requesting user before calling this.
func (s *Store) CreateCompanion(parentPCID, ownerUserID, name string, maxHP int, sharesInitiative bool) (*PC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := PC{
		ID:               NewID(8),
		OwnerUserID:      ownerUserID,
		Name:             name,
		Type:             "companion",
		MaxHP:            maxHP,
		ParentPCID:       parentPCID,
		SharesInitiative: sharesInitiative,
	}
	if _, err := s.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetCompanionsByParentID returns all companion documents whose parent_pc_id matches.
func (s *Store) GetCompanionsByParentID(parentID string) ([]PC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := s.col.Find(ctx, bson.M{"type": "companion", "parent_pc_id": parentID})
	if err != nil {
		return nil, err
	}
	var results []PC
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ListPCsByOwner returns all PCs (type "pc", not companions) owned by ownerUserID.
func (s *Store) ListPCsByOwner(ownerUserID string) ([]PC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := s.col.Find(ctx, bson.M{"type": "pc", "owner_user_id": ownerUserID})
	if err != nil {
		return nil, err
	}
	var results []PC
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// OwnsAnyPC reports whether ownerUserID owns at least one PC among pcIDs, via
// a targeted existence check rather than fetching full PC documents.
func (s *Store) OwnsAnyPC(ownerUserID string, pcIDs []string) (bool, error) {
	if len(pcIDs) == 0 {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := s.col.CountDocuments(ctx, bson.M{
		"owner_user_id": ownerUserID,
		"id":            bson.M{"$in": pcIDs},
	}, options.Count().SetLimit(1))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
