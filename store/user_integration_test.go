//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUserStore_CreateUser_And_GetByUsername(t *testing.T) {
	u, err := GlobalUsers.CreateUser("frodo", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u.ID == "" || u.DisplayName != "frodo" {
		t.Fatalf("CreateUser() = %+v, want a non-empty ID and DisplayName=frodo", u)
	}

	got, err := GlobalUsers.GetUserByUsername("frodo")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if got == nil || got.ID != u.ID || got.PassphraseHash != "hash1" {
		t.Errorf("GetUserByUsername() = %+v, want the user just created", got)
	}
}

func TestUserStore_CreateUser_DuplicateUsername(t *testing.T) {
	if _, err := GlobalUsers.CreateUser("sam", "hash1"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	_, err := GlobalUsers.CreateUser("sam", "hash2")
	if err == nil {
		t.Fatal("CreateUser() with a taken username: error = nil, want an error")
	}
	if err.Error() != "username taken" {
		t.Errorf("CreateUser() with a taken username: error = %v, want \"username taken\"", err)
	}
}

func TestUserStore_GetUserByUsername_NotFound(t *testing.T) {
	got, err := GlobalUsers.GetUserByUsername("does-not-exist")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetUserByUsername() = %+v, want nil", got)
	}
}

func TestUserStore_GetUserByID(t *testing.T) {
	u, err := GlobalUsers.CreateUser("pippin", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	got, err := GlobalUsers.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if got == nil || got.Username != "pippin" {
		t.Errorf("GetUserByID() = %+v, want Username=pippin", got)
	}
}

func TestUserStore_GetUserByID_NotFound(t *testing.T) {
	got, err := GlobalUsers.GetUserByID("does-not-exist")
	if err != nil {
		t.Fatalf("GetUserByID() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetUserByID() = %+v, want nil", got)
	}
}

func TestUserStore_CreateSession_And_ResolveSession(t *testing.T) {
	u, err := GlobalUsers.CreateUser("merry", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	sess, err := GlobalUsers.CreateSession(u.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if sess.Token == "" || sess.UserID != u.ID {
		t.Fatalf("CreateSession() = %+v, want a non-empty Token and UserID=%s", sess, u.ID)
	}

	got, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() error = %v", err)
	}
	if got == nil || got.UserID != u.ID {
		t.Errorf("ResolveSession() = %+v, want UserID=%s", got, u.ID)
	}
}

func TestUserStore_ResolveSession_NotFound(t *testing.T) {
	got, err := GlobalUsers.ResolveSession("does-not-exist")
	if err != nil {
		t.Fatalf("ResolveSession() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("ResolveSession() = %+v, want nil", got)
	}
}

func TestUserStore_ResolveSession_Expired(t *testing.T) {
	u, err := GlobalUsers.CreateUser("expired-user", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	sess, err := GlobalUsers.CreateSession(u.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Backdate the session's expiry directly, bypassing CreateSession (which
	// always sets a fresh 90-day window), to exercise the expiry branch.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	past := time.Now().Add(-1 * time.Hour)
	if _, err := GlobalUsers.sessions.UpdateOne(ctx, bson.M{"token": sess.Token}, bson.M{"$set": bson.M{"expires_at": past}}); err != nil {
		t.Fatalf("failed to backdate session: %v", err)
	}

	got, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("ResolveSession() on expired session = %+v, want nil", got)
	}
}

func TestUserStore_ResolveSession_RefreshesWhenStale(t *testing.T) {
	u, err := GlobalUsers.CreateUser("stale-user", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	sess, err := GlobalUsers.CreateSession(u.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Push last_seen_at back beyond the debounce window so the next
	// ResolveSession() call refreshes the rolling expiry.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stale := time.Now().Add(-1 * time.Hour)
	if _, err := GlobalUsers.sessions.UpdateOne(ctx, bson.M{"token": sess.Token}, bson.M{"$set": bson.M{
		"last_seen_at": stale,
		"expires_at":   stale.Add(sessionRollingWindow),
	}}); err != nil {
		t.Fatalf("failed to stale-date session: %v", err)
	}

	got, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() error = %v", err)
	}
	if got == nil {
		t.Fatal("ResolveSession() = nil, want the refreshed session")
	}
	if !got.LastSeenAt.After(stale) {
		t.Errorf("ResolveSession() LastSeenAt = %v, want it refreshed to after %v", got.LastSeenAt, stale)
	}
	if !got.ExpiresAt.After(stale.Add(sessionRollingWindow)) {
		t.Errorf("ResolveSession() ExpiresAt = %v, want it pushed out past the stale expiry", got.ExpiresAt)
	}
}

func TestUserStore_ResolveSession_NoRefreshWhenFresh(t *testing.T) {
	u, err := GlobalUsers.CreateUser("fresh-user", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	sess, err := GlobalUsers.CreateSession(u.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Two ResolveSession() calls in quick succession are both well within the
	// debounce window, so the second should see the exact same LastSeenAt the
	// first one persisted (compared DB-to-DB to sidestep BSON's millisecond
	// time truncation vs. Go's nanosecond precision).
	first, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() (first) error = %v", err)
	}
	if first == nil {
		t.Fatal("ResolveSession() (first) = nil, want the session")
	}

	second, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() (second) error = %v", err)
	}
	if second == nil {
		t.Fatal("ResolveSession() (second) = nil, want the session")
	}
	if !second.LastSeenAt.Equal(first.LastSeenAt) {
		t.Errorf("ResolveSession() LastSeenAt changed from %v to %v, want it unchanged (within debounce window)", first.LastSeenAt, second.LastSeenAt)
	}
}

func TestUserStore_DeleteSession(t *testing.T) {
	u, err := GlobalUsers.CreateUser("gandalf", "hash1")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	sess, err := GlobalUsers.CreateSession(u.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if err := GlobalUsers.DeleteSession(sess.Token); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	got, err := GlobalUsers.ResolveSession(sess.Token)
	if err != nil {
		t.Fatalf("ResolveSession() error = %v", err)
	}
	if got != nil {
		t.Errorf("ResolveSession() after DeleteSession() = %+v, want nil", got)
	}

	// Deleting an already-deleted (or never-existing) session is a no-op, not an error.
	if err := GlobalUsers.DeleteSession(sess.Token); err != nil {
		t.Errorf("DeleteSession() on missing session error = %v, want nil", err)
	}
}
