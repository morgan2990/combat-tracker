package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const sessionRollingWindow = 90 * 24 * time.Hour
const sessionTouchDebounce = 5 * time.Minute

// User is a self-serve account: a username/passphrase pair that owns Rooms and PCs.
type User struct {
	ID             string    `bson:"id"               json:"id"`
	Username       string    `bson:"username"         json:"username"`
	PassphraseHash string    `bson:"passphrase_hash"  json:"-"`
	DisplayName    string    `bson:"display_name"     json:"display_name"`
	CreatedAt      time.Time `bson:"created_at"       json:"created_at"`
}

// Session is a DB-backed login session, identified by an opaque token carried in a cookie.
type Session struct {
	Token      string    `bson:"token"        json:"token"`
	UserID     string    `bson:"user_id"      json:"user_id"`
	CreatedAt  time.Time `bson:"created_at"   json:"created_at"`
	LastSeenAt time.Time `bson:"last_seen_at" json:"last_seen_at"`
	ExpiresAt  time.Time `bson:"expires_at"   json:"expires_at"`
}

type UserStore struct {
	users    *mongo.Collection
	sessions *mongo.Collection
}

var GlobalUsers UserStore

func ensureUserIndexes(ctx context.Context, users, sessions *mongo.Collection) error {
	if _, err := users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	_, err := sessions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// CreateUser inserts a new user with a generated ID. Returns an error if the username is taken.
func (s *UserStore) CreateUser(username, passphraseHash string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	u := User{
		ID:             NewID(8),
		Username:       username,
		PassphraseHash: passphraseHash,
		DisplayName:    username,
		CreatedAt:      time.Now(),
	}
	if _, err := s.users.InsertOne(ctx, u); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errors.New("username taken")
		}
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername returns the user with the given username, or nil if not found.
func (s *UserStore) GetUserByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u User
	err := s.users.FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID returns the user with the given id, or nil if not found.
func (s *UserStore) GetUserByID(id string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u User
	err := s.users.FindOne(ctx, bson.M{"id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateSession creates a new rolling session for userID and returns it.
func (s *UserStore) CreateSession(userID string) (*Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now()
	sess := Session{
		Token:      NewID(8),
		UserID:     userID,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(sessionRollingWindow),
	}
	if _, err := s.sessions.InsertOne(ctx, sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// ResolveSession looks up a session by token. It returns nil if the session does not
// exist or has expired. If the session is valid and was last touched more than
// sessionTouchDebounce ago, its rolling expiry is refreshed.
func (s *UserStore) ResolveSession(token string) (*Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sess Session
	err := s.sessions.FindOne(ctx, bson.M{"token": token}).Decode(&sess)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if now.After(sess.ExpiresAt) {
		return nil, nil
	}
	if now.Sub(sess.LastSeenAt) > sessionTouchDebounce {
		sess.LastSeenAt = now
		sess.ExpiresAt = now.Add(sessionRollingWindow)
		_, _ = s.sessions.UpdateOne(ctx, bson.M{"token": token}, bson.M{"$set": bson.M{
			"last_seen_at": sess.LastSeenAt,
			"expires_at":   sess.ExpiresAt,
		}})
	}
	return &sess, nil
}

// DeleteSession removes a session (logout).
func (s *UserStore) DeleteSession(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.sessions.DeleteOne(ctx, bson.M{"token": token})
	return err
}

// NewID generates a random hex-encoded id of the given byte length. Exported
// so callers outside store (e.g. room, api) can pre-generate ids using the
// same scheme, instead of each maintaining their own random-id generator.
func NewID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
