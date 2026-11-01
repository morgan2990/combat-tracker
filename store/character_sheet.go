package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SheetSchemaVersion is written on every newly created sheet so a future
// change to the section shapes can tell old documents from new ones.
const SheetSchemaVersion = 1

// ErrSheetConflict is returned by SaveSection when the caller's revision no
// longer matches the stored one (including a first save racing another).
var ErrSheetConflict = errors.New("sheet revision conflict")

// SheetSource records where a sheet's data came from. A manual save never
// writes or clears it; it is reserved for the PDF import.
type SheetSource struct {
	Kind          string     `bson:"kind,omitempty"           json:"kind,omitempty"`
	SheetVersion  string     `bson:"sheet_version,omitempty"  json:"sheet_version,omitempty"`
	SheetBuild    string     `bson:"sheet_build,omitempty"    json:"sheet_build,omitempty"`
	ImportedAt    *time.Time `bson:"imported_at,omitempty"    json:"imported_at,omitempty"`
	FileName      string     `bson:"file_name,omitempty"      json:"file_name,omitempty"`
	FileSHA256    string     `bson:"file_sha256,omitempty"    json:"file_sha256,omitempty"`
	MapperVersion string     `bson:"mapper_version,omitempty" json:"mapper_version,omitempty"`
	ImportID      string     `bson:"import_id,omitempty"      json:"import_id,omitempty"`
}

// SheetClass is one entry of a multiclass character. Every field is optional:
// the sheet may only carry a free-text class line.
type SheetClass struct {
	Class    string `bson:"class,omitempty"    json:"class,omitempty"`
	Subclass string `bson:"subclass,omitempty" json:"subclass,omitempty"`
	Level    *int   `bson:"level,omitempty"    json:"level,omitempty"`
}

// SheetIdentity is the identity section. Name is accepted on save but lives
// only on the PC document (hence bson:"-"); readers take it from the PC.
type SheetIdentity struct {
	Name           string       `bson:"-"                         json:"name,omitempty"`
	Player         string       `bson:"player,omitempty"          json:"player,omitempty"`
	ClassesText    string       `bson:"classes_text,omitempty"    json:"classes_text,omitempty"`
	CharacterLevel *int         `bson:"character_level,omitempty" json:"character_level,omitempty"`
	Classes        []SheetClass `bson:"classes,omitempty"         json:"classes,omitempty"`
	Species        string       `bson:"species,omitempty"         json:"species,omitempty"`
	Background     string       `bson:"background,omitempty"      json:"background,omitempty"`
	Size           string       `bson:"size,omitempty"            json:"size,omitempty"`
	Alignment      string       `bson:"alignment,omitempty"       json:"alignment,omitempty"`
	XP             *int         `bson:"xp,omitempty"              json:"xp,omitempty"`
}

// AbilityScore holds one ability as displayed. Mod and SaveBonus are stored
// as given — never derived from Score.
type AbilityScore struct {
	Score          *int `bson:"score,omitempty"      json:"score,omitempty"`
	Mod            *int `bson:"mod,omitempty"        json:"mod,omitempty"`
	SaveBonus      *int `bson:"save_bonus,omitempty" json:"save_bonus,omitempty"`
	SaveProficient bool `bson:"save_proficient"      json:"save_proficient"`
}

// SheetAbilities is the abilities section: the six scores with their saves.
type SheetAbilities struct {
	Str AbilityScore `bson:"str" json:"str"`
	Dex AbilityScore `bson:"dex" json:"dex"`
	Con AbilityScore `bson:"con" json:"con"`
	Int AbilityScore `bson:"int" json:"int"`
	Wis AbilityScore `bson:"wis" json:"wis"`
	Cha AbilityScore `bson:"cha" json:"cha"`
}

// SkillEntry is one skill row.
type SkillEntry struct {
	Key        string `bson:"key"         json:"key"`
	Ability    string `bson:"ability"     json:"ability,omitempty"`
	Bonus      *int   `bson:"bonus,omitempty" json:"bonus,omitempty"`
	Proficient bool   `bson:"proficient"  json:"proficient"`
	Expertise  bool   `bson:"expertise"   json:"expertise"`
}

// SheetSkills is the skills section, including passive senses.
type SheetSkills struct {
	Entries              []SkillEntry `bson:"entries,omitempty"               json:"entries,omitempty"`
	PassivePerception    *int         `bson:"passive_perception,omitempty"    json:"passive_perception,omitempty"`
	PassiveInsight       *int         `bson:"passive_insight,omitempty"       json:"passive_insight,omitempty"`
	PassiveInvestigation *int         `bson:"passive_investigation,omitempty" json:"passive_investigation,omitempty"`
}

// HitDie is a pool of hit dice of one size. Spent dice are not stored.
type HitDie struct {
	Die   int `bson:"die"   json:"die"`
	Total int `bson:"total" json:"total"`
}

// Speeds holds per-mode speeds; SheetCombat.SpeedText keeps the sheet's own wording.
type Speeds struct {
	Walk   *int `bson:"walk,omitempty"   json:"walk,omitempty"`
	Climb  *int `bson:"climb,omitempty"  json:"climb,omitempty"`
	Swim   *int `bson:"swim,omitempty"   json:"swim,omitempty"`
	Fly    *int `bson:"fly,omitempty"    json:"fly,omitempty"`
	Burrow *int `bson:"burrow,omitempty" json:"burrow,omitempty"`
}

// SheetCombat is the combat section. MaxHP is accepted on save but lives only
// on the PC document (bson:"-"); current/temp HP and death saves are not stored.
type SheetCombat struct {
	MaxHP            int      `bson:"-"                          json:"max_hp,omitempty"`
	AC               *int     `bson:"ac,omitempty"               json:"ac,omitempty"`
	Initiative       *int     `bson:"initiative,omitempty"       json:"initiative,omitempty"`
	ProficiencyBonus *int     `bson:"proficiency_bonus,omitempty" json:"proficiency_bonus,omitempty"`
	SpeedText        string   `bson:"speed_text,omitempty"       json:"speed_text,omitempty"`
	Speeds           Speeds   `bson:"speeds"                     json:"speeds"`
	HitDice          []HitDie `bson:"hit_dice,omitempty"         json:"hit_dice,omitempty"`
}

// SheetDefenses is the defenses section.
type SheetDefenses struct {
	Vision      string   `bson:"vision,omitempty"       json:"vision,omitempty"`
	Resistances []string `bson:"resistances,omitempty"  json:"resistances,omitempty"`
	SaveNotes   string   `bson:"save_notes,omitempty"   json:"save_notes,omitempty"`
}

// Attack is one attack row. ToHit and Damage are free text ("+7", "DC 12", "1d8+3").
type Attack struct {
	Name       string `bson:"name"                  json:"name"`
	Ability    string `bson:"ability,omitempty"     json:"ability,omitempty"`
	Range      string `bson:"range,omitempty"       json:"range,omitempty"`
	ToHit      string `bson:"to_hit,omitempty"      json:"to_hit,omitempty"`
	Damage     string `bson:"damage,omitempty"      json:"damage,omitempty"`
	DamageType string `bson:"damage_type,omitempty" json:"damage_type,omitempty"`
	Notes      string `bson:"notes,omitempty"       json:"notes,omitempty"`
}

// ActionEntry is one line of the actions lists; Source names the feat/feature it comes from.
type ActionEntry struct {
	Name      string `bson:"name"             json:"name"`
	Source    string `bson:"source,omitempty" json:"source,omitempty"`
	IsDefault bool   `bson:"is_default"       json:"is_default"`
}

// SheetActions is the actions section.
type SheetActions struct {
	Action      []ActionEntry `bson:"action,omitempty"       json:"action,omitempty"`
	BonusAction []ActionEntry `bson:"bonus_action,omitempty" json:"bonus_action,omitempty"`
	Reaction    []ActionEntry `bson:"reaction,omitempty"     json:"reaction,omitempty"`
}

// LimitedUse is one limited-use feature. Max and Recovery are free text
// ("2d6", "LR/SS", "dawn"); Source optionally names a magic item or feature.
type LimitedUse struct {
	Name     string `bson:"name"               json:"name"`
	Max      string `bson:"max,omitempty"      json:"max,omitempty"`
	Recovery string `bson:"recovery,omitempty" json:"recovery,omitempty"`
	Source   string `bson:"source,omitempty"   json:"source,omitempty"`
}

// Feature is a class, species, background or feat feature.
type Feature struct {
	Kind     string `bson:"kind"               json:"kind"`
	Name     string `bson:"name"               json:"name"`
	Text     string `bson:"text,omitempty"     json:"text,omitempty"`
	Source   string `bson:"source,omitempty"   json:"source,omitempty"`
	Level    *int   `bson:"level,omitempty"    json:"level,omitempty"`
	Book     string `bson:"book,omitempty"     json:"book,omitempty"`
	Page     string `bson:"page,omitempty"     json:"page,omitempty"`
	UsesNote string `bson:"uses_note,omitempty" json:"uses_note,omitempty"`
}

// WeaponMastery is one weapon mastery entry.
type WeaponMastery struct {
	Weapon  string `bson:"weapon"           json:"weapon"`
	Mastery string `bson:"mastery,omitempty" json:"mastery,omitempty"`
	Notes   string `bson:"notes,omitempty"   json:"notes,omitempty"`
}

// Proficiency is one armor/weapon/tool/language entry. Pending marks a choice
// the sheet has not resolved yet (e.g. "from Human").
type Proficiency struct {
	Name    string `bson:"name"    json:"name"`
	Pending bool   `bson:"pending" json:"pending"`
}

// SheetProficiencies is the proficiencies section.
type SheetProficiencies struct {
	Armor     []Proficiency `bson:"armor,omitempty"     json:"armor,omitempty"`
	Weapons   []Proficiency `bson:"weapons,omitempty"   json:"weapons,omitempty"`
	Tools     []Proficiency `bson:"tools,omitempty"     json:"tools,omitempty"`
	Languages []Proficiency `bson:"languages,omitempty" json:"languages,omitempty"`
}

// MagicItem is a magic item. Its uses are limited uses naming it as Source.
type MagicItem struct {
	Name               string `bson:"name"                         json:"name"`
	Stage              string `bson:"stage,omitempty"              json:"stage,omitempty"`
	AttunementRequired bool   `bson:"attunement_required"          json:"attunement_required"`
	Attuned            bool   `bson:"attuned"                      json:"attuned"`
	Text               string `bson:"text,omitempty"               json:"text,omitempty"`
	Source             string `bson:"source,omitempty"             json:"source,omitempty"`
	CatalogItemID      string `bson:"catalog_item_id,omitempty"    json:"catalog_item_id,omitempty"`
}

// Spell is one spell row. Preparation is one of the SpellPreparations; Mark
// keeps the sheet's original marker when Preparation is "other".
type Spell struct {
	Key         string `bson:"key,omitempty"         json:"key,omitempty"`
	Name        string `bson:"name"                  json:"name"`
	Level       int    `bson:"level"                 json:"level"`
	Preparation string `bson:"preparation"           json:"preparation"`
	Mark        string `bson:"mark,omitempty"        json:"mark,omitempty"`
	Save        string `bson:"save,omitempty"        json:"save,omitempty"`
	School      string `bson:"school,omitempty"      json:"school,omitempty"`
	CastingTime string `bson:"casting_time,omitempty" json:"casting_time,omitempty"`
	Range       string `bson:"range,omitempty"       json:"range,omitempty"`
	Components  string `bson:"components,omitempty"  json:"components,omitempty"`
	Duration    string `bson:"duration,omitempty"    json:"duration,omitempty"`
	Book        string `bson:"book,omitempty"        json:"book,omitempty"`
	Page        string `bson:"page,omitempty"        json:"page,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// SpellList is one source of spells: a class, a feat, an item, or other.
type SpellList struct {
	Kind         string  `bson:"kind"                    json:"kind"`
	Name         string  `bson:"name"                    json:"name"`
	Ability      string  `bson:"ability,omitempty"       json:"ability,omitempty"`
	SaveDC       string  `bson:"save_dc,omitempty"       json:"save_dc,omitempty"`
	Attack       string  `bson:"attack,omitempty"        json:"attack,omitempty"`
	PrepareCount *int    `bson:"prepare_count,omitempty" json:"prepare_count,omitempty"`
	Spells       []Spell `bson:"spells,omitempty"        json:"spells,omitempty"`
}

// PactMagic holds Pact Magic slots, kept apart from the regular slots.
type PactMagic struct {
	SlotLevel int `bson:"slot_level" json:"slot_level"`
	Total     int `bson:"total"      json:"total"`
}

// SheetSpellcasting is the spellcasting section. Slots[i] is the total number
// of slots of level i+1; spent slots are not stored.
type SheetSpellcasting struct {
	SpellLists []SpellList `bson:"spell_lists,omitempty" json:"spell_lists,omitempty"`
	Slots      []int       `bson:"slots,omitempty"       json:"slots,omitempty"`
	PactMagic  *PactMagic  `bson:"pact_magic,omitempty"  json:"pact_magic,omitempty"`
}

// CreatureGear is one equipment row on a creature.
type CreatureGear struct {
	Name   string `bson:"name"             json:"name"`
	Amount string `bson:"amount,omitempty" json:"amount,omitempty"`
}

// Creature is the reduced sheet shared by companions and wild shape forms.
// AC and HP are free text ("13+Prof", "10 (3d6)").
type Creature struct {
	Name             string         `bson:"name,omitempty"              json:"name,omitempty"`
	Species          string         `bson:"species,omitempty"           json:"species,omitempty"`
	Type             string         `bson:"type,omitempty"              json:"type,omitempty"`
	Size             string         `bson:"size,omitempty"              json:"size,omitempty"`
	Alignment        string         `bson:"alignment,omitempty"         json:"alignment,omitempty"`
	AC               string         `bson:"ac,omitempty"                json:"ac,omitempty"`
	HP               string         `bson:"hp,omitempty"                json:"hp,omitempty"`
	SpeedText        string         `bson:"speed_text,omitempty"        json:"speed_text,omitempty"`
	Senses           string         `bson:"senses,omitempty"            json:"senses,omitempty"`
	ProficiencyBonus *int           `bson:"proficiency_bonus,omitempty" json:"proficiency_bonus,omitempty"`
	Abilities        SheetAbilities `bson:"abilities"                   json:"abilities"`
	Saves            []string       `bson:"saves,omitempty"             json:"saves,omitempty"`
	Skills           []SkillEntry   `bson:"skills,omitempty"            json:"skills,omitempty"`
	Attacks          []Attack       `bson:"attacks,omitempty"           json:"attacks,omitempty"`
	Traits           string         `bson:"traits,omitempty"            json:"traits,omitempty"`
	Gear             []CreatureGear `bson:"gear,omitempty"              json:"gear,omitempty"`
	Notes            string         `bson:"notes,omitempty"             json:"notes,omitempty"`
}

// SheetWildShapes is the wild shapes section.
type SheetWildShapes struct {
	Uses        string     `bson:"uses,omitempty"        json:"uses,omitempty"`
	Duration    string     `bson:"duration,omitempty"    json:"duration,omitempty"`
	Limitations string     `bson:"limitations,omitempty" json:"limitations,omitempty"`
	Forms       []Creature `bson:"forms,omitempty"       json:"forms,omitempty"`
}

// SheetNotes is the notes and description section.
type SheetNotes struct {
	Appearance  string `bson:"appearance,omitempty"  json:"appearance,omitempty"`
	Personality string `bson:"personality,omitempty" json:"personality,omitempty"`
	Ideals      string `bson:"ideals,omitempty"      json:"ideals,omitempty"`
	Bonds       string `bson:"bonds,omitempty"       json:"bonds,omitempty"`
	Flaws       string `bson:"flaws,omitempty"       json:"flaws,omitempty"`
	Backstory   string `bson:"backstory,omitempty"   json:"backstory,omitempty"`
	FreeText    string `bson:"free_text,omitempty"   json:"free_text,omitempty"`
}

// CharacterSheet is one PC's (or companion's) sheet. A PC uses every section
// except Creature; a companion uses only Creature. Unset sections are nil.
type CharacterSheet struct {
	PCID          string       `bson:"pc_id"          json:"pc_id"`
	SchemaVersion int          `bson:"schema_version" json:"schema_version"`
	Revision      int          `bson:"revision"       json:"revision"`
	UpdatedAt     time.Time    `bson:"updated_at"     json:"-"`
	Source        *SheetSource `bson:"source,omitempty" json:"source,omitempty"`

	Identity        *SheetIdentity      `bson:"identity,omitempty"         json:"identity,omitempty"`
	Abilities       *SheetAbilities     `bson:"abilities,omitempty"        json:"abilities,omitempty"`
	Skills          *SheetSkills        `bson:"skills,omitempty"           json:"skills,omitempty"`
	Combat          *SheetCombat        `bson:"combat,omitempty"           json:"combat,omitempty"`
	Defenses        *SheetDefenses      `bson:"defenses,omitempty"         json:"defenses,omitempty"`
	Attacks         []Attack            `bson:"attacks,omitempty"          json:"attacks,omitempty"`
	Actions         *SheetActions       `bson:"actions,omitempty"          json:"actions,omitempty"`
	LimitedUses     []LimitedUse        `bson:"limited_uses,omitempty"     json:"limited_uses,omitempty"`
	Features        []Feature           `bson:"features,omitempty"         json:"features,omitempty"`
	WeaponMasteries []WeaponMastery     `bson:"weapon_masteries,omitempty" json:"weapon_masteries,omitempty"`
	Proficiencies   *SheetProficiencies `bson:"proficiencies,omitempty"    json:"proficiencies,omitempty"`
	MagicItems      []MagicItem         `bson:"magic_items,omitempty"      json:"magic_items,omitempty"`
	Spellcasting    *SheetSpellcasting  `bson:"spellcasting,omitempty"     json:"spellcasting,omitempty"`
	WildShapes      *SheetWildShapes    `bson:"wild_shapes,omitempty"      json:"wild_shapes,omitempty"`
	Notes           *SheetNotes         `bson:"notes,omitempty"            json:"notes,omitempty"`
	Creature        *Creature           `bson:"creature,omitempty"         json:"creature,omitempty"`
}

// EmptyCharacterSheet is what a PC with no stored sheet reads as.
func EmptyCharacterSheet(pcID string) CharacterSheet {
	return CharacterSheet{PCID: pcID, SchemaVersion: SheetSchemaVersion}
}

// CharacterSheetStore wraps the character_sheets MongoDB collection.
type CharacterSheetStore struct {
	col *mongo.Collection
}

var GlobalCharacterSheets CharacterSheetStore

// ensureCharacterSheetIndex ensures one sheet per PC.
func ensureCharacterSheetIndex(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "pc_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// GetByPCID returns the stored sheet for pcID, or nil if none exists.
func (s *CharacterSheetStore) GetByPCID(pcID string) (*CharacterSheet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sh CharacterSheet
	err := s.col.FindOne(ctx, bson.M{"pc_id": pcID}).Decode(&sh)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

// GetByPCIDs returns the stored sheets among pcIDs, keyed by pc_id. PCs
// without a sheet are simply absent from the map.
func (s *CharacterSheetStore) GetByPCIDs(pcIDs []string) (map[string]CharacterSheet, error) {
	out := make(map[string]CharacterSheet, len(pcIDs))
	if len(pcIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := s.col.Find(ctx, bson.M{"pc_id": bson.M{"$in": pcIDs}})
	if err != nil {
		return nil, err
	}
	var sheets []CharacterSheet
	if err := cursor.All(ctx, &sheets); err != nil {
		return nil, err
	}
	for _, sh := range sheets {
		out[sh.PCID] = sh
	}
	return out, nil
}

// SaveSection sets one section of pcID's sheet and returns the new revision.
// It is a compare-and-set on revision: revision 0 creates the sheet (and
// fails with ErrSheetConflict if one already exists); any other revision must
// match the stored one. Other sections and Source are never touched.
// value is encoded with its bson tags, so callers pass the typed section.
func (s *CharacterSheetStore) SaveSection(pcID, section string, revision int, value any) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now().UTC()
	if revision == 0 {
		_, err := s.col.InsertOne(ctx, bson.M{
			"pc_id":          pcID,
			"schema_version": SheetSchemaVersion,
			"revision":       1,
			"updated_at":     now,
			section:          value,
		})
		if mongo.IsDuplicateKeyError(err) {
			return 0, ErrSheetConflict
		}
		if err != nil {
			return 0, err
		}
		return 1, nil
	}
	res, err := s.col.UpdateOne(ctx,
		bson.M{"pc_id": pcID, "revision": revision},
		bson.M{
			"$set": bson.M{section: value, "updated_at": now},
			"$inc": bson.M{"revision": 1},
		})
	if err != nil {
		return 0, err
	}
	if res.MatchedCount == 0 {
		return 0, ErrSheetConflict
	}
	return revision + 1, nil
}

// DeleteByPCIDs deletes the sheets of every PC in pcIDs.
func (s *CharacterSheetStore) DeleteByPCIDs(pcIDs []string) error {
	if len(pcIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.col.DeleteMany(ctx, bson.M{"pc_id": bson.M{"$in": pcIDs}})
	return err
}
