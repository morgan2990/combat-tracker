package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ImportDraftTTL is how long an unconfirmed upload is kept.
const ImportDraftTTL = 24 * time.Hour

// RawField is one non-empty form field from an uploaded MPMB PDF. Names are
// the qualified /T names (they contain dots, so fields are kept as an array,
// never a map). TM is the field's mapping name, which MPMB uses to carry data
// (e.g. an action's source); Display is a choice field's selected label.
type RawField struct {
	Name    string `bson:"name"              json:"name"`
	TM      string `bson:"tm,omitempty"      json:"tm,omitempty"`
	Type    string `bson:"type,omitempty"    json:"type,omitempty"`
	Value   string `bson:"value"             json:"value"`
	Display string `bson:"display,omitempty" json:"display,omitempty"`
}

// ImportedCompanion is a companion read from the PDF. MaxHP is 0 when the
// sheet gave no usable number; applying it then falls back to 1.
type ImportedCompanion struct {
	Name     string   `bson:"name"     json:"name"`
	MaxHP    int      `bson:"max_hp"   json:"max_hp"`
	Creature Creature `bson:"creature" json:"creature"`
}

// ImportedInventory is the gear and coins read from the PDF. Present is true
// when it holds at least one item or a non-zero coin; only then does applying
// the import replace the PC's own items and currency.
type ImportedInventory struct {
	Present  bool     `bson:"present"  json:"present"`
	Items    []Item   `bson:"items"    json:"items"`
	Currency Currency `bson:"currency" json:"currency"`
}

// ImportParsed is everything mapped from a PDF. Name and MaxHP are the PC's;
// MaxHP is 0 when the sheet's HP Max was not a positive number. Only the
// sections of Sheet are meaningful; its identity fields are filled in when
// the import is applied.
type ImportParsed struct {
	Name       string              `bson:"name"       json:"name"`
	MaxHP      int                 `bson:"max_hp"     json:"max_hp"`
	Sheet      CharacterSheet      `bson:"sheet"      json:"sheet"`
	Companions []ImportedCompanion `bson:"companions" json:"companions"`
	Inventory  ImportedInventory   `bson:"inventory"  json:"inventory"`
	Warnings   []string            `bson:"warnings"   json:"warnings"`
}

// Import statuses. An import is claimed (draft -> applying) before it is
// applied, so two confirms of one draft cannot both run.
const (
	ImportDraft    = "draft"
	ImportApplying = "applying"
	ImportApplied  = "applied"
)

// CharacterSheetImport is one upload: a draft awaiting confirmation, or the
// latest applied import of a PC. PCID is empty for a draft that will create a
// new character. ExpiresAt is set only on drafts (a TTL index removes them);
// applying an import clears it.
type CharacterSheetImport struct {
	ID           string       `bson:"id"                      json:"id"`
	OwnerUserID  string       `bson:"owner_user_id"           json:"owner_user_id"`
	PCID         string       `bson:"pc_id,omitempty"         json:"pc_id,omitempty"`
	Status       string       `bson:"status"                  json:"status"`
	CreatedAt    time.Time    `bson:"created_at"              json:"created_at"`
	ExpiresAt    *time.Time   `bson:"expires_at,omitempty"    json:"expires_at,omitempty"`
	AppliedAt    *time.Time   `bson:"applied_at,omitempty"    json:"applied_at,omitempty"`
	BaseRevision int          `bson:"base_revision"           json:"base_revision"`
	Source       SheetSource  `bson:"source"                  json:"source"`
	Parsed       ImportParsed `bson:"parsed"                  json:"parsed"`
	RawFields    []RawField   `bson:"raw_fields"              json:"raw_fields"`
}

// CharacterSheetImportStore wraps the character_sheet_imports collection.
type CharacterSheetImportStore struct {
	col *mongo.Collection
}

var GlobalSheetImports CharacterSheetImportStore

func ensureSheetImportIndexes(ctx context.Context, col *mongo.Collection) error {
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "pc_id", Value: 1}}}); err != nil {
		return err
	}
	// Drafts expire on their own; expires_at is unset once an import is applied,
	// and a TTL index ignores documents without a date there.
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

// Create inserts a draft. ID, Status, CreatedAt and ExpiresAt are set here.
func (s *CharacterSheetImportStore) Create(imp CharacterSheetImport) (*CharacterSheetImport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := time.Now().UTC()
	exp := now.Add(ImportDraftTTL)
	imp.ID = NewID(8)
	imp.Status = ImportDraft
	imp.CreatedAt = now
	imp.ExpiresAt = &exp
	if imp.RawFields == nil {
		imp.RawFields = []RawField{}
	}
	if _, err := s.col.InsertOne(ctx, imp); err != nil {
		return nil, err
	}
	return &imp, nil
}

// Get returns the import with this id, or nil.
func (s *CharacterSheetImportStore) Get(id string) (*CharacterSheetImport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var imp CharacterSheetImport
	err := s.col.FindOne(ctx, bson.M{"id": id}).Decode(&imp)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &imp, nil
}

// Claim atomically moves a draft to applying and returns it, or nil if there
// is no such draft (already claimed, applied, or gone).
func (s *CharacterSheetImportStore) Claim(id string) (*CharacterSheetImport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var imp CharacterSheetImport
	err := s.col.FindOneAndUpdate(ctx,
		bson.M{"id": id, "status": ImportDraft},
		bson.M{"$set": bson.M{"status": ImportApplying}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&imp)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &imp, nil
}

// Release puts a claimed import back to draft, after an apply that failed.
func (s *CharacterSheetImportStore) Release(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.UpdateOne(ctx, bson.M{"id": id, "status": ImportApplying}, bson.M{"$set": bson.M{"status": ImportDraft}})
	return err
}

// MarkApplied finishes a claimed import: it records the PC it was applied to
// and stops it from expiring.
func (s *CharacterSheetImportStore) MarkApplied(id, pcID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.UpdateOne(ctx, bson.M{"id": id}, bson.M{
		"$set":   bson.M{"status": ImportApplied, "pc_id": pcID, "applied_at": time.Now().UTC()},
		"$unset": bson.M{"expires_at": ""},
	})
	return err
}

// Delete removes one import.
func (s *CharacterSheetImportStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.DeleteOne(ctx, bson.M{"id": id})
	return err
}

// DeleteOthersForPC removes every import of pcID except keepID: only the
// latest applied import is kept.
func (s *CharacterSheetImportStore) DeleteOthersForPC(pcID, keepID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.DeleteMany(ctx, bson.M{"pc_id": pcID, "id": bson.M{"$ne": keepID}})
	return err
}

// DeleteByPCIDs removes the imports of PCs that no longer exist.
func (s *CharacterSheetImportStore) DeleteByPCIDs(pcIDs []string) error {
	if len(pcIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.DeleteMany(ctx, bson.M{"pc_id": bson.M{"$in": pcIDs}})
	return err
}
