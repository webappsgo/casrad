// Package store - Tests for MemoryStore CRUD operations.
// Covers: admin/user/session/token lifecycle, duplicate detection, not-found paths,
// list pagination, session hashing, deletion, concurrent access safety.
package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
)

func newCtx() context.Context { return context.Background() }

// --- Infrastructure ---

func TestNewMemoryStore(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	if s == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if err := s.Ping(newCtx()); err != nil {
		t.Errorf("Ping unexpected error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close unexpected error: %v", err)
	}
	if err := s.Migrate(newCtx()); err != nil {
		t.Errorf("Migrate unexpected error: %v", err)
	}
}

// --- Admin operations ---

func TestAdminCRUD(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	admin := &model.Admin{
		Username: "sysadmin",
		Email:    "sysadmin@example.com",
		IsActive: true,
		Role:     "admin",
	}

	// Create
	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin error: %v", err)
	}
	if id <= 0 {
		t.Errorf("CreateAdmin returned invalid id %d", id)
	}

	// GetByID
	got, err := s.GetAdminByID(ctx, id)
	if err != nil {
		t.Fatalf("GetAdminByID error: %v", err)
	}
	if got == nil {
		t.Fatal("GetAdminByID returned nil for existing admin")
	}
	if got.Username != admin.Username {
		t.Errorf("GetAdminByID username = %q, want %q", got.Username, admin.Username)
	}

	// GetByUsername
	byUsername, err := s.GetAdminByUsername(ctx, "sysadmin")
	if err != nil {
		t.Fatalf("GetAdminByUsername error: %v", err)
	}
	if byUsername == nil {
		t.Fatal("GetAdminByUsername returned nil")
	}

	// GetByEmail
	byEmail, err := s.GetAdminByEmail(ctx, "sysadmin@example.com")
	if err != nil {
		t.Fatalf("GetAdminByEmail error: %v", err)
	}
	if byEmail == nil {
		t.Fatal("GetAdminByEmail returned nil")
	}

	// Update
	got.Role = "super_admin"
	got.ID = id
	if err := s.UpdateAdmin(ctx, got); err != nil {
		t.Fatalf("UpdateAdmin error: %v", err)
	}
	updated, _ := s.GetAdminByID(ctx, id)
	if updated.Role != "super_admin" {
		t.Errorf("UpdateAdmin role = %q, want %q", updated.Role, "super_admin")
	}
}

func TestAdminNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	got, err := s.GetAdminByID(ctx, 999)
	if err != nil {
		t.Errorf("GetAdminByID not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetAdminByID not-found should return nil")
	}

	got, err = s.GetAdminByUsername(ctx, "nobody")
	if err != nil {
		t.Errorf("GetAdminByUsername not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetAdminByUsername not-found should return nil")
	}

	got, err = s.GetAdminByEmail(ctx, "nobody@example.com")
	if err != nil {
		t.Errorf("GetAdminByEmail not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetAdminByEmail not-found should return nil")
	}
}

func TestAdminDuplicateRejected(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	a := &model.Admin{Username: "dupuser", Email: "dup@example.com"}
	if _, err := s.CreateAdmin(ctx, a); err != nil {
		t.Fatalf("first CreateAdmin failed: %v", err)
	}

	// Duplicate username
	_, err := s.CreateAdmin(ctx, &model.Admin{Username: "dupuser", Email: "other@example.com"})
	if err == nil {
		t.Error("duplicate username should be rejected")
	}

	// Duplicate email
	_, err = s.CreateAdmin(ctx, &model.Admin{Username: "other", Email: "dup@example.com"})
	if err == nil {
		t.Error("duplicate email should be rejected")
	}
}

func TestAdminUpdateNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	err := s.UpdateAdmin(ctx, &model.Admin{ID: 999, Username: "ghost"})
	if err == nil {
		t.Error("UpdateAdmin for non-existent admin should return error")
	}
}

// --- User operations ---

func TestUserCRUD(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	user := &model.User{
		Username: "alice",
		Email:    "alice@example.com",
		IsActive: true,
	}

	// Create
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if id <= 0 {
		t.Errorf("CreateUser returned invalid id %d", id)
	}

	// GetByID
	got, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByID returned nil")
	}
	if got.Username != "alice" {
		t.Errorf("GetUserByID username = %q, want alice", got.Username)
	}

	// GetByUsername
	byUsername, err := s.GetUserByUsername(ctx, "alice")
	if err != nil || byUsername == nil {
		t.Fatalf("GetUserByUsername error or nil: %v", err)
	}

	// GetByEmail
	byEmail, err := s.GetUserByEmail(ctx, "alice@example.com")
	if err != nil || byEmail == nil {
		t.Fatalf("GetUserByEmail error or nil: %v", err)
	}

	// Defaults applied
	if got.Role != "user" {
		t.Errorf("default role = %q, want user", got.Role)
	}
	if got.ThemePreference != "dark" {
		t.Errorf("default theme = %q, want dark", got.ThemePreference)
	}
	if got.StorageQuotaBytes != 53687091200 {
		t.Errorf("default quota = %d, want 53687091200", got.StorageQuotaBytes)
	}

	// Update
	got.Bio = "Hello world"
	if err := s.UpdateUser(ctx, got); err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	updated, _ := s.GetUserByID(ctx, id)
	if updated.Bio != "Hello world" {
		t.Errorf("UpdateUser bio = %q, want %q", updated.Bio, "Hello world")
	}

	// Delete
	if err := s.DeleteUser(ctx, id); err != nil {
		t.Fatalf("DeleteUser error: %v", err)
	}
	deleted, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Errorf("GetUserByID after delete unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("GetUserByID after delete should return nil")
	}
}

func TestUserDefaultsWhenZero(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	// Provide explicit role and theme — should not be overwritten
	u := &model.User{
		Username:          "bob",
		Email:             "bob@example.com",
		Role:              "moderator",
		ThemePreference:   "light",
		StorageQuotaBytes: 1024,
	}
	id, err := s.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	got, _ := s.GetUserByID(ctx, id)
	if got.Role != "moderator" {
		t.Errorf("explicit role = %q, want moderator", got.Role)
	}
	if got.ThemePreference != "light" {
		t.Errorf("explicit theme = %q, want light", got.ThemePreference)
	}
	if got.StorageQuotaBytes != 1024 {
		t.Errorf("explicit quota = %d, want 1024", got.StorageQuotaBytes)
	}
}

func TestUserDuplicateRejected(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	_, err := s.CreateUser(ctx, &model.User{Username: "charlie", Email: "charlie@example.com"})
	if err != nil {
		t.Fatalf("first CreateUser error: %v", err)
	}

	_, err = s.CreateUser(ctx, &model.User{Username: "charlie", Email: "other@example.com"})
	if err == nil {
		t.Error("duplicate username should fail")
	}

	_, err = s.CreateUser(ctx, &model.User{Username: "other", Email: "charlie@example.com"})
	if err == nil {
		t.Error("duplicate email should fail")
	}
}

func TestUserListPagination(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	// Create 5 users
	for i := 0; i < 5; i++ {
		username := "user" + string(rune('a'+i))
		email := username + "@example.com"
		_, err := s.CreateUser(ctx, &model.User{Username: username, Email: email})
		if err != nil {
			t.Fatalf("CreateUser[%d] error: %v", i, err)
		}
	}

	// List all
	all, total, err := s.ListUsers(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListUsers error: %v", err)
	}
	if total != 5 {
		t.Errorf("ListUsers total = %d, want 5", total)
	}
	if len(all) != 5 {
		t.Errorf("ListUsers len = %d, want 5", len(all))
	}

	// First page of 2
	page1, total2, err := s.ListUsers(ctx, 0, 2)
	if err != nil {
		t.Fatalf("ListUsers page1 error: %v", err)
	}
	if total2 != 5 {
		t.Errorf("ListUsers page1 total = %d, want 5", total2)
	}
	if len(page1) != 2 {
		t.Errorf("ListUsers page1 len = %d, want 2", len(page1))
	}

	// Second page of 2
	page2, _, err := s.ListUsers(ctx, 2, 2)
	if err != nil {
		t.Fatalf("ListUsers page2 error: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("ListUsers page2 len = %d, want 2", len(page2))
	}

	// Offset beyond end returns empty
	empty, _, err := s.ListUsers(ctx, 100, 10)
	if err != nil {
		t.Fatalf("ListUsers empty page error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("ListUsers beyond end len = %d, want 0", len(empty))
	}
}

func TestUserUpdateNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	err := s.UpdateUser(ctx, &model.User{ID: 999})
	if err == nil {
		t.Error("UpdateUser for non-existent user should return error")
	}
}

// --- Session operations ---

func TestSessionCRUD(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	session := &model.Session{
		ID:        "raw-session-id-abc123",
		UserID:    1,
		IPAddress: "127.0.0.1",
		IsActive:  true,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	// Create
	if err := s.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Get by raw ID
	got, err := s.GetSession(ctx, "raw-session-id-abc123")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if got == nil {
		t.Fatal("GetSession returned nil for existing session")
	}
	if got.UserID != 1 {
		t.Errorf("GetSession UserID = %d, want 1", got.UserID)
	}

	// Update
	if err := s.UpdateSession(ctx, got); err != nil {
		t.Fatalf("UpdateSession error: %v", err)
	}

	// Delete
	if err := s.DeleteSession(ctx, "raw-session-id-abc123"); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	// Verify gone
	gone, err := s.GetSession(ctx, "raw-session-id-abc123")
	if err != nil {
		t.Errorf("GetSession after delete unexpected error: %v", err)
	}
	if gone != nil {
		t.Error("GetSession after delete should return nil")
	}
}

func TestSessionHashingIsTransparent(t *testing.T) {
	// The same raw ID used to create must retrieve the same record.
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	rawID := "super-secret-session-token-xyz"
	session := &model.Session{
		ID:       rawID,
		UserID:   42,
		IsActive: true,
	}

	if err := s.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	got, err := s.GetSession(ctx, rawID)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if got == nil {
		t.Fatal("session not found after create — hashing mismatch")
	}
}

func TestSessionNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	got, err := s.GetSession(ctx, "nonexistent-session")
	if err != nil {
		t.Errorf("GetSession not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetSession not-found should return nil")
	}
}

func TestSessionUpdateNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	err := s.UpdateSession(ctx, &model.Session{ID: "nonexistent"})
	if err == nil {
		t.Error("UpdateSession for non-existent session should return error")
	}
}

func TestDeleteUserSessions(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	// Create two sessions for user 1 and one for user 2
	for i, rawID := range []string{"sess1", "sess2"} {
		_ = i
		s.CreateSession(ctx, &model.Session{ID: rawID, UserID: 1, IsActive: true})
	}
	s.CreateSession(ctx, &model.Session{ID: "sess3", UserID: 2, IsActive: true})

	if err := s.DeleteUserSessions(ctx, 1); err != nil {
		t.Fatalf("DeleteUserSessions error: %v", err)
	}

	// User 1's sessions gone
	for _, rawID := range []string{"sess1", "sess2"} {
		got, _ := s.GetSession(ctx, rawID)
		if got != nil {
			t.Errorf("session %q should be gone after DeleteUserSessions", rawID)
		}
	}

	// User 2's session intact
	got, _ := s.GetSession(ctx, "sess3")
	if got == nil {
		t.Error("user 2 session should survive DeleteUserSessions for user 1")
	}
}

// --- Token operations ---

func TestTokenCRUD(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	rawToken := "usr_abc123xyz"
	tok := &model.APIToken{
		UserID:   1,
		Token:    rawToken,
		Name:     "test token",
		IsActive: true,
	}

	// Create
	id, err := s.CreateToken(ctx, tok)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}
	if id <= 0 {
		t.Errorf("CreateToken returned invalid id %d", id)
	}

	// GetToken by raw value
	got, err := s.GetToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("GetToken error: %v", err)
	}
	if got == nil {
		t.Fatal("GetToken returned nil — hashing mismatch")
	}
	if got.Name != "test token" {
		t.Errorf("GetToken name = %q, want %q", got.Name, "test token")
	}

	// GetTokenByID
	byID, err := s.GetTokenByID(ctx, id)
	if err != nil {
		t.Fatalf("GetTokenByID error: %v", err)
	}
	if byID == nil {
		t.Fatal("GetTokenByID returned nil")
	}

	// ListUserTokens
	tokens, err := s.ListUserTokens(ctx, 1)
	if err != nil {
		t.Fatalf("ListUserTokens error: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("ListUserTokens = %d tokens, want 1", len(tokens))
	}

	// Delete
	if err := s.DeleteToken(ctx, id); err != nil {
		t.Fatalf("DeleteToken error: %v", err)
	}

	// Verify gone by raw value
	gone, err := s.GetToken(ctx, rawToken)
	if err != nil {
		t.Errorf("GetToken after delete unexpected error: %v", err)
	}
	if gone != nil {
		t.Error("GetToken after delete should return nil")
	}

	// Verify gone by ID
	goneByID, err := s.GetTokenByID(ctx, id)
	if err != nil {
		t.Errorf("GetTokenByID after delete unexpected error: %v", err)
	}
	if goneByID != nil {
		t.Error("GetTokenByID after delete should return nil")
	}
}

func TestTokenNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	got, err := s.GetToken(ctx, "nonexistent-token")
	if err != nil {
		t.Errorf("GetToken not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetToken not-found should return nil")
	}

	got2, err := s.GetTokenByID(ctx, 999)
	if err != nil {
		t.Errorf("GetTokenByID not-found unexpected error: %v", err)
	}
	if got2 != nil {
		t.Error("GetTokenByID not-found should return nil")
	}
}

func TestListUserTokensEmpty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	tokens, err := s.ListUserTokens(ctx, 999)
	if err != nil {
		t.Fatalf("ListUserTokens empty unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("ListUserTokens for unknown user = %d, want 0", len(tokens))
	}
}

// --- Concurrency ---

func TestMemoryStoreConcurrentWrites(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := newCtx()

	// 20 goroutines each creating a unique user — no races, no duplicate errors
	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			username := "concurrent" + string(rune('a'+i%26)) + "user" + itoa(i)
			email := username + "@example.com"
			_, err := s.CreateUser(ctx, &model.User{Username: username, Email: email})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent CreateUser error: %v", err)
	}

	// Verify all created
	_, total, _ := s.ListUsers(ctx, 0, 100)
	if total != goroutines {
		t.Errorf("after concurrent writes total = %d, want %d", total, goroutines)
	}
}

// itoa converts an int to its ASCII string for test name generation without fmt import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
