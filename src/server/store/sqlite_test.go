// Package store — Integration tests for SQLiteStore using an in-memory database.
// Uses modernc.org/sqlite (pure Go, CGO_ENABLED=0) with the ":memory:" DSN.
// Covers: NewSQLiteStore, Ping, Migrate, Close, and all CRUD operations.
package store

import (
	"context"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
)

// newTestStore creates an in-memory SQLite store, runs migrations, and returns it.
func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore(:memory:): %v", err)
	}
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// --- Bootstrap ---

func TestNewSQLiteStore(t *testing.T) {
	t.Parallel()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer s.Close()
}

func TestSQLiteStorePing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestSQLiteStoreMigrateIdempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	// Migrate twice — schema uses IF NOT EXISTS so it must be safe
	if err := s.Migrate(ctx); err != nil {
		t.Errorf("second Migrate should be idempotent: %v", err)
	}
}

func TestSQLiteStoreClose(t *testing.T) {
	t.Parallel()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// --- Admin CRUD ---

func TestSQLiteAdminCreateAndGet(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{
		Username:     "sysadmin",
		Email:        "sysadmin@example.com",
		PasswordHash: "hashed",
	}

	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}
	if id == 0 {
		t.Error("CreateAdmin should return non-zero ID")
	}

	byID, err := s.GetAdminByID(ctx, id)
	if err != nil {
		t.Fatalf("GetAdminByID: %v", err)
	}
	if byID.Username != "sysadmin" {
		t.Errorf("Username = %q, want sysadmin", byID.Username)
	}

	byUsername, err := s.GetAdminByUsername(ctx, "sysadmin")
	if err != nil {
		t.Fatalf("GetAdminByUsername: %v", err)
	}
	if byUsername.ID != id {
		t.Errorf("GetAdminByUsername ID = %d, want %d", byUsername.ID, id)
	}

	byEmail, err := s.GetAdminByEmail(ctx, "sysadmin@example.com")
	if err != nil {
		t.Fatalf("GetAdminByEmail: %v", err)
	}
	if byEmail.ID != id {
		t.Errorf("GetAdminByEmail ID = %d, want %d", byEmail.ID, id)
	}
}

func TestSQLiteAdminNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin, err := s.GetAdminByID(ctx, 9999)
	if err != nil {
		t.Errorf("GetAdminByID not-found should return nil error, got: %v", err)
	}
	if admin != nil {
		t.Error("GetAdminByID not-found should return nil admin")
	}

	admin, err = s.GetAdminByUsername(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetAdminByUsername not-found should return nil error, got: %v", err)
	}
	if admin != nil {
		t.Error("GetAdminByUsername not-found should return nil admin")
	}
}

func TestSQLiteAdminDuplicateRejected(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "dup_admin", Email: "dup@example.com", PasswordHash: "hash"}
	if _, err := s.CreateAdmin(ctx, admin); err != nil {
		t.Fatalf("first CreateAdmin: %v", err)
	}

	dup := &model.Admin{Username: "dup_admin", Email: "other@example.com", PasswordHash: "hash"}
	if _, err := s.CreateAdmin(ctx, dup); err == nil {
		t.Error("CreateAdmin with duplicate username should fail")
	}

	dupEmail := &model.Admin{Username: "other_admin", Email: "dup@example.com", PasswordHash: "hash"}
	if _, err := s.CreateAdmin(ctx, dupEmail); err == nil {
		t.Error("CreateAdmin with duplicate email should fail")
	}
}

func TestSQLiteAdminUpdate(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "admin2", Email: "admin2@example.com", PasswordHash: "old_hash"}
	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	fetched, _ := s.GetAdminByID(ctx, id)
	fetched.PasswordHash = "new_hash"
	if err := s.UpdateAdmin(ctx, fetched); err != nil {
		t.Fatalf("UpdateAdmin: %v", err)
	}

	updated, _ := s.GetAdminByID(ctx, id)
	if updated.PasswordHash != "new_hash" {
		t.Errorf("PasswordHash = %q, want new_hash", updated.PasswordHash)
	}
}

// --- User CRUD ---

func TestSQLiteUserCreateAndGet(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed",
	}

	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id == 0 {
		t.Error("CreateUser should return non-zero ID")
	}

	byID, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Username != "alice" {
		t.Errorf("Username = %q, want alice", byID.Username)
	}

	byUsername, err := s.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if byUsername.ID != id {
		t.Errorf("GetUserByUsername ID = %d, want %d", byUsername.ID, id)
	}

	byEmail, err := s.GetUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if byEmail.ID != id {
		t.Errorf("GetUserByEmail ID = %d, want %d", byEmail.ID, id)
	}
}

func TestSQLiteUserDefaults(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	// SQLiteStore applies Role/ThemePreference/StorageQuotaBytes defaults when zero.
	// IsActive must be set by the caller explicitly (SQLite INSERT passes the value directly).
	user := &model.User{Username: "bob", Email: "bob@example.com", PasswordHash: "hash", IsActive: true}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	fetched, _ := s.GetUserByID(ctx, id)
	if fetched.Role != "user" {
		t.Errorf("default Role = %q, want user", fetched.Role)
	}
	if fetched.ThemePreference != "dark" {
		t.Errorf("default ThemePreference = %q, want dark", fetched.ThemePreference)
	}
	if fetched.StorageQuotaBytes != 53687091200 {
		t.Errorf("default StorageQuotaBytes = %d, want 53687091200", fetched.StorageQuotaBytes)
	}
	if !fetched.IsActive {
		t.Error("IsActive should be true (set on creation)")
	}
}

func TestSQLiteUserNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	u, err := s.GetUserByID(ctx, 9999)
	if err != nil {
		t.Errorf("GetUserByID not-found should return nil error, got: %v", err)
	}
	if u != nil {
		t.Error("GetUserByID not-found should return nil")
	}
}

func TestSQLiteUserDuplicateRejected(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "dup_user", Email: "dup_user@example.com", PasswordHash: "hash"}
	if _, err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	dup := &model.User{Username: "dup_user", Email: "other@example.com", PasswordHash: "hash"}
	if _, err := s.CreateUser(ctx, dup); err == nil {
		t.Error("duplicate username should be rejected")
	}
}

func TestSQLiteUserUpdate(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "charlie", Email: "charlie@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	fetched, _ := s.GetUserByID(ctx, id)
	fetched.Bio = "Music lover"
	fetched.UpdatedAt = time.Now()
	if err := s.UpdateUser(ctx, fetched); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	updated, _ := s.GetUserByID(ctx, id)
	if updated.Bio != "Music lover" {
		t.Errorf("Bio = %q, want 'Music lover'", updated.Bio)
	}
}

func TestSQLiteUserDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "dave", Email: "dave@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := s.DeleteUser(ctx, id); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	gone, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Errorf("GetUserByID after delete should return nil error, got: %v", err)
	}
	if gone != nil {
		t.Error("GetUserByID after delete should return nil")
	}
}

func TestSQLiteUserListPagination(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &model.User{
			Username:     "user_list_" + string(rune('a'+i)),
			Email:        "list_" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
		}
		if _, err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser %d: %v", i, err)
		}
	}

	users, total, err := s.ListUsers(ctx, 0, 3)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total < 5 {
		t.Errorf("total = %d, want >= 5", total)
	}
	if len(users) != 3 {
		t.Errorf("page size = %d, want 3", len(users))
	}

	// Second page
	users2, _, err := s.ListUsers(ctx, 3, 3)
	if err != nil {
		t.Fatalf("ListUsers page 2: %v", err)
	}
	if len(users2) < 2 {
		t.Errorf("second page = %d users, want >= 2", len(users2))
	}
}

// --- Session CRUD ---

func TestSQLiteSessionCreateAndGet(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	// Create admin to own the session
	admin := &model.Admin{Username: "sess_admin", Email: "sess_admin@example.com", PasswordHash: "hash"}
	adminID, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	session := &model.Session{
		ID:        "raw-session-id-12345",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}

	if err := s.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	fetched, err := s.GetSession(ctx, "raw-session-id-12345")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if fetched == nil {
		t.Fatal("GetSession returned nil")
	}
	if fetched.AdminID != adminID {
		t.Errorf("AdminID = %d, want %d", fetched.AdminID, adminID)
	}
}

func TestSQLiteSessionNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.GetSession(ctx, "does-not-exist")
	if err != nil {
		t.Errorf("GetSession not-found should return nil error, got: %v", err)
	}
	if sess != nil {
		t.Error("GetSession not-found should return nil")
	}
}

func TestSQLiteSessionDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "del_sess_adm", Email: "del_sess_adm@example.com", PasswordHash: "hash"}
	adminID, _ := s.CreateAdmin(ctx, admin)

	session := &model.Session{
		ID:        "delete-me-session",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}
	if err := s.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.DeleteSession(ctx, "delete-me-session"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	gone, _ := s.GetSession(ctx, "delete-me-session")
	if gone != nil {
		t.Error("session should be gone after DeleteSession")
	}
}

func TestSQLiteSessionUpdate(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "upd_sess_adm", Email: "upd_sess_adm@example.com", PasswordHash: "hash"}
	adminID, _ := s.CreateAdmin(ctx, admin)

	session := &model.Session{
		ID:        "update-me-session",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}
	if err := s.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// UpdateSession persists is_active and expires_at; ip_address is not part of the UPDATE
	session.IsActive = false
	newExpiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	session.ExpiresAt = newExpiry
	if err := s.UpdateSession(ctx, session); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	updated, _ := s.GetSession(ctx, "update-me-session")
	if updated == nil {
		t.Fatal("updated session not found")
	}
	if updated.IsActive {
		t.Error("IsActive should be false after update")
	}
}

// --- Token CRUD ---

func TestSQLiteTokenCreateAndGet(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "token_user", Email: "token_user@example.com", PasswordHash: "hash"}
	userID, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := &model.APIToken{
		UserID:   userID,
		Token:    "raw-token-abc",
		Name:     "test token",
		IsActive: true,
	}
	id, err := s.CreateToken(ctx, token)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if id == 0 {
		t.Error("CreateToken should return non-zero ID")
	}

	// GetToken by raw value
	fetched, err := s.GetToken(ctx, "raw-token-abc")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if fetched == nil {
		t.Fatal("GetToken returned nil")
	}
	if fetched.Name != "test token" {
		t.Errorf("Name = %q, want 'test token'", fetched.Name)
	}

	// GetTokenByID
	byID, err := s.GetTokenByID(ctx, id)
	if err != nil {
		t.Fatalf("GetTokenByID: %v", err)
	}
	if byID.ID != id {
		t.Errorf("GetTokenByID.ID = %d, want %d", byID.ID, id)
	}
}

func TestSQLiteTokenNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	tok, err := s.GetToken(ctx, "nonexistent-raw-token")
	if err != nil {
		t.Errorf("GetToken not-found should return nil error, got: %v", err)
	}
	if tok != nil {
		t.Error("GetToken not-found should return nil")
	}
}

func TestSQLiteTokenDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "del_tok_user", Email: "del_tok_user@example.com", PasswordHash: "hash"}
	userID, _ := s.CreateUser(ctx, user)

	token := &model.APIToken{
		UserID:   userID,
		Token:    "delete-raw-token",
		Name:     "delete me",
		IsActive: true,
	}
	id, err := s.CreateToken(ctx, token)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if err := s.DeleteToken(ctx, id); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	gone, _ := s.GetToken(ctx, "delete-raw-token")
	if gone != nil {
		t.Error("token should be gone after DeleteToken")
	}
}

func TestSQLiteTokenListUserTokens(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "list_tok_user", Email: "list_tok_user@example.com", PasswordHash: "hash"}
	userID, _ := s.CreateUser(ctx, user)

	for i := 0; i < 3; i++ {
		tok := &model.APIToken{
			UserID:   userID,
			Token:    "raw-tok-" + string(rune('a'+i)),
			Name:     "token " + string(rune('a'+i)),
			IsActive: true,
		}
		if _, err := s.CreateToken(ctx, tok); err != nil {
			t.Fatalf("CreateToken %d: %v", i, err)
		}
	}

	tokens, err := s.ListUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("ListUserTokens: %v", err)
	}
	if len(tokens) != 3 {
		t.Errorf("token count = %d, want 3", len(tokens))
	}
}

func TestSQLiteTokenListEmpty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	tokens, err := s.ListUserTokens(ctx, 9999)
	if err != nil {
		t.Fatalf("ListUserTokens empty: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}
