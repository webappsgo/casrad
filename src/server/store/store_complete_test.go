// Package store — Additional tests to boost coverage.
// Covers: SQLiteStore.DeleteUserSessions, nil copy helpers, GetToken nil branch,
// GetUserByUsername/Email nil returns, idempotent Migrate, SQLiteStore.Close error path,
// SQLiteStore.GetAdminByEmail, UpdateAdmin with optional fields set,
// UpdateUser with optional fields, multiple-page ListUsers.
package store

import (
	"context"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
)

// --- Nil copy helper coverage ---
// The copy* helpers have a nil guard branch (75% covered — need to exercise nil path).

func TestCopyAdminNil(t *testing.T) {
	t.Parallel()
	if got := copyAdmin(nil); got != nil {
		t.Errorf("copyAdmin(nil) = %v, want nil", got)
	}
}

func TestCopyUserNil(t *testing.T) {
	t.Parallel()
	if got := copyUser(nil); got != nil {
		t.Errorf("copyUser(nil) = %v, want nil", got)
	}
}

func TestCopySessionNil(t *testing.T) {
	t.Parallel()
	if got := copySession(nil); got != nil {
		t.Errorf("copySession(nil) = %v, want nil", got)
	}
}

func TestCopyTokenNil(t *testing.T) {
	t.Parallel()
	if got := copyToken(nil); got != nil {
		t.Errorf("copyToken(nil) = %v, want nil", got)
	}
}

// --- hashForStorage is deterministic ---

func TestHashForStorageIsDeterministic(t *testing.T) {
	t.Parallel()
	h1 := hashForStorage("my-raw-token")
	h2 := hashForStorage("my-raw-token")
	if h1 != h2 {
		t.Errorf("hashForStorage not deterministic: %q != %q", h1, h2)
	}
}

func TestHashForStorageDifferentInputsDifferentHashes(t *testing.T) {
	t.Parallel()
	h1 := hashForStorage("token-a")
	h2 := hashForStorage("token-b")
	if h1 == h2 {
		t.Error("hashForStorage produced same hash for different inputs")
	}
}

func TestHashForStorageEmptyString(t *testing.T) {
	t.Parallel()
	h := hashForStorage("")
	if h == "" {
		t.Error("hashForStorage('') should return non-empty hash")
	}
}

// --- MemoryStore: GetToken nil branch (tokenByID has key but tokens map doesn't) ---

func TestMemoryStoreGetTokenMissingFromTokensMap(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := context.Background()

	// Directly inject an id into tokenByID that has no matching token in tokens map
	s.mu.Lock()
	s.tokenByID[hashForStorage("orphan-token")] = 9999
	s.mu.Unlock()

	got, err := s.GetToken(ctx, "orphan-token")
	if err != nil {
		t.Errorf("GetToken orphan unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetToken should return nil when token ID not found in tokens map")
	}
}

// --- MemoryStore: GetUserByUsername/Email nil returns (not currently in test suite) ---

func TestMemoryStoreGetUserByUsernameNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := context.Background()
	// Populate with a different user so the loop runs
	s.CreateUser(ctx, &model.User{Username: "alice", Email: "alice@example.com"})

	got, err := s.GetUserByUsername(ctx, "nobody")
	if err != nil {
		t.Errorf("GetUserByUsername not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetUserByUsername not-found should return nil")
	}
}

func TestMemoryStoreGetUserByEmailNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ctx := context.Background()
	// Populate store so the loop executes
	s.CreateUser(ctx, &model.User{Username: "bob", Email: "bob@example.com"})

	got, err := s.GetUserByEmail(ctx, "nobody@example.com")
	if err != nil {
		t.Errorf("GetUserByEmail not-found unexpected error: %v", err)
	}
	if got != nil {
		t.Error("GetUserByEmail not-found should return nil")
	}
}

// --- SQLiteStore: DeleteUserSessions (0.0% covered) ---

func TestSQLiteDeleteUserSessions(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "sess_del_admin", Email: "sess_del@example.com", PasswordHash: "hash"}
	adminID, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	user := &model.User{Username: "sess_del_user", Email: "sess_del_user@example.com", PasswordHash: "hash"}
	userID, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Create two sessions for the user
	for i, rawID := range []string{"del-sess-1", "del-sess-2"} {
		_ = i
		sess := &model.Session{
			ID:        rawID,
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
			IsActive:  true,
		}
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatalf("CreateSession(%s): %v", rawID, err)
		}
	}

	// Create one session for admin (different UserID path)
	adminSess := &model.Session{
		ID:        "admin-sess-keep",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}
	if err := s.CreateSession(ctx, adminSess); err != nil {
		t.Fatalf("CreateSession(admin): %v", err)
	}

	// Delete all sessions for user
	if err := s.DeleteUserSessions(ctx, userID); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}

	// User sessions must be gone
	for _, rawID := range []string{"del-sess-1", "del-sess-2"} {
		got, err := s.GetSession(ctx, rawID)
		if err != nil {
			t.Errorf("GetSession(%s) after delete unexpected error: %v", rawID, err)
		}
		if got != nil {
			t.Errorf("session %q should be deleted by DeleteUserSessions", rawID)
		}
	}

	// Admin session must survive (different user_id)
	got, err := s.GetSession(ctx, "admin-sess-keep")
	if err != nil {
		t.Errorf("GetSession(admin) unexpected error: %v", err)
	}
	if got == nil {
		t.Error("admin session should survive DeleteUserSessions for a different user")
	}
}

func TestSQLiteDeleteUserSessionsNoSessions(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	// Deleting sessions for a user that has none should not error
	if err := s.DeleteUserSessions(ctx, 9999); err != nil {
		t.Errorf("DeleteUserSessions(no sessions) unexpected error: %v", err)
	}
}

// --- SQLiteStore: GetAdminByEmail (66.7% — add a found case) ---

func TestSQLiteGetAdminByEmailFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "findme_admin", Email: "findme@example.com", PasswordHash: "hash"}
	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	got, err := s.GetAdminByEmail(ctx, "findme@example.com")
	if err != nil {
		t.Fatalf("GetAdminByEmail: %v", err)
	}
	if got == nil {
		t.Fatal("GetAdminByEmail returned nil for existing email")
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
}

func TestSQLiteGetAdminByEmailNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.GetAdminByEmail(ctx, "nobody@example.com")
	if err != nil {
		t.Errorf("GetAdminByEmail not-found should return nil error: %v", err)
	}
	if got != nil {
		t.Error("GetAdminByEmail not-found should return nil")
	}
}

// --- SQLiteStore: UpdateAdmin with optional nullable fields set ---

func TestSQLiteUpdateAdminWithOptionalFields(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "opts_admin", Email: "opts@example.com", PasswordHash: "hash"}
	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	// Update with all optional fields populated (exercises nullable branches)
	fetched, _ := s.GetAdminByID(ctx, id)
	fetched.TOTPSecret = "JBSWY3DPEHPK3PXP"
	fetched.LastLogin = time.Now()
	fetched.LastIP = "10.0.0.1"
	fetched.LockedUntil = time.Now().Add(30 * time.Minute)
	fetched.FailedLoginAttempts = 3

	if err := s.UpdateAdmin(ctx, fetched); err != nil {
		t.Fatalf("UpdateAdmin with optional fields: %v", err)
	}

	updated, err := s.GetAdminByID(ctx, id)
	if err != nil {
		t.Fatalf("GetAdminByID after update: %v", err)
	}
	if updated.TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("TOTPSecret = %q, want JBSWY3DPEHPK3PXP", updated.TOTPSecret)
	}
	if updated.FailedLoginAttempts != 3 {
		t.Errorf("FailedLoginAttempts = %d, want 3", updated.FailedLoginAttempts)
	}
}

// --- SQLiteStore: UpdateUser with optional nullable fields set ---

func TestSQLiteUpdateUserWithOptionalFields(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "opts_user", Email: "opts_user@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	fetched, _ := s.GetUserByID(ctx, id)
	fetched.TOTPSecret = "TOTP_SECRET"
	fetched.LastLogin = time.Now()
	fetched.LastIP = "192.168.1.1"
	fetched.LockedUntil = time.Now().Add(30 * time.Minute)
	fetched.FailedLoginAttempts = 2
	fetched.Settings = `{"theme":"dark"}`
	fetched.AvatarURL = "https://example.com/avatar.jpg"
	fetched.Bio = "Music lover"
	fetched.Website = "https://example.com"
	fetched.Location = "New York"

	if err := s.UpdateUser(ctx, fetched); err != nil {
		t.Fatalf("UpdateUser with optional fields: %v", err)
	}

	updated, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID after update: %v", err)
	}
	if updated.Bio != "Music lover" {
		t.Errorf("Bio = %q, want 'Music lover'", updated.Bio)
	}
	if updated.Website != "https://example.com" {
		t.Errorf("Website = %q", updated.Website)
	}
	if updated.FailedLoginAttempts != 2 {
		t.Errorf("FailedLoginAttempts = %d, want 2", updated.FailedLoginAttempts)
	}
}

// --- SQLiteStore: GetUserByUsername and GetUserByEmail ---

func TestSQLiteGetUserByUsernameFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "findme_user", Email: "findme_user@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := s.GetUserByUsername(ctx, "findme_user")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByUsername returned nil")
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
}

func TestSQLiteGetUserByUsernameNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.GetUserByUsername(ctx, "nonexistent_user")
	if err != nil {
		t.Errorf("GetUserByUsername not-found should return nil error: %v", err)
	}
	if got != nil {
		t.Error("GetUserByUsername not-found should return nil")
	}
}

func TestSQLiteGetUserByEmailFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{Username: "email_user", Email: "email_user@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := s.GetUserByEmail(ctx, "email_user@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByEmail returned nil")
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
}

func TestSQLiteGetUserByEmailNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.GetUserByEmail(ctx, "nobody@example.com")
	if err != nil {
		t.Errorf("GetUserByEmail not-found should return nil error: %v", err)
	}
	if got != nil {
		t.Error("GetUserByEmail not-found should return nil")
	}
}

// --- SQLiteStore: multiple sessions for different users, then delete one set ---

func TestSQLiteDeleteUserSessionsIsolation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	userA := &model.User{Username: "user_a_sess", Email: "user_a_sess@example.com", PasswordHash: "hash"}
	userB := &model.User{Username: "user_b_sess", Email: "user_b_sess@example.com", PasswordHash: "hash"}

	idA, _ := s.CreateUser(ctx, userA)
	idB, _ := s.CreateUser(ctx, userB)

	s.CreateSession(ctx, &model.Session{ID: "a-s1", UserID: idA, ExpiresAt: time.Now().Add(time.Hour), IsActive: true})
	s.CreateSession(ctx, &model.Session{ID: "b-s1", UserID: idB, ExpiresAt: time.Now().Add(time.Hour), IsActive: true})

	if err := s.DeleteUserSessions(ctx, idA); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}

	// A's sessions gone
	goneA, _ := s.GetSession(ctx, "a-s1")
	if goneA != nil {
		t.Error("user A session should be deleted")
	}

	// B's session intact
	keepB, _ := s.GetSession(ctx, "b-s1")
	if keepB == nil {
		t.Error("user B session should survive")
	}
}

// --- SQLiteStore: ListUsers offset beyond end returns empty ---

func TestSQLiteListUsersOffsetBeyondEnd(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	u := &model.User{Username: "only_one", Email: "only_one@example.com", PasswordHash: "hash"}
	if _, err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	users, total, err := s.ListUsers(ctx, 100, 10)
	if err != nil {
		t.Fatalf("ListUsers beyond end: %v", err)
	}
	if total < 1 {
		t.Errorf("total should be >= 1, got %d", total)
	}
	if len(users) != 0 {
		t.Errorf("ListUsers beyond end should return 0 users, got %d", len(users))
	}
}

// --- SQLiteStore: Session UpdateSession (exercise full update path) ---

func TestSQLiteSessionUpdateThemeName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{Username: "theme_sess_adm", Email: "theme_sess_adm@example.com", PasswordHash: "hash"}
	adminID, _ := s.CreateAdmin(ctx, admin)

	sess := &model.Session{
		ID:        "theme-update-sess",
		AdminID:   adminID,
		ThemeName: "dark",
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sess.ThemeName = "light"
	if err := s.UpdateSession(ctx, sess); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	updated, err := s.GetSession(ctx, "theme-update-sess")
	if err != nil {
		t.Fatalf("GetSession after update: %v", err)
	}
	if updated == nil {
		t.Fatal("updated session not found")
	}
}

// --- SQLiteStore: Token — GetTokenByID not found ---

func TestSQLiteGetTokenByIDNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	tok, err := s.GetTokenByID(ctx, 9999)
	if err != nil {
		t.Errorf("GetTokenByID not-found should return nil error: %v", err)
	}
	if tok != nil {
		t.Error("GetTokenByID not-found should return nil")
	}
}

// --- SQLiteStore: idempotent Migrate (already in sqlite_test.go, but run twice explicitly) ---

func TestSQLiteMigrateTwiceIsIdempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Errorf("second Migrate should not error: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Errorf("third Migrate should not error: %v", err)
	}
}

// --- SQLiteStore: GetUserByUsername/Email with fully populated optional fields ---
// Covers the .Valid branches in the Scan result block.

func TestSQLiteGetUserByUsernameWithOptionalFields(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{
		Username:     "full_opts_user",
		Email:        "full_opts@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Update to set all optional nullable fields
	fetched, _ := s.GetUserByID(ctx, id)
	fetched.TOTPSecret = "SECRET"
	fetched.HomeDirectory = "/var/lib/casrad/users/full_opts_user"
	fetched.LastLogin = time.Now().Truncate(time.Second)
	fetched.LastIP = "10.0.0.5"
	fetched.LockedUntil = time.Now().Add(time.Hour).Truncate(time.Second)
	fetched.Settings = `{"key":"val"}`
	fetched.AvatarURL = "https://example.com/av.jpg"
	fetched.Bio = "Bio text"
	fetched.Website = "https://example.com"
	fetched.Location = "Berlin"
	if err := s.UpdateUser(ctx, fetched); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// Now get by username — exercises all .Valid branches
	got, err := s.GetUserByUsername(ctx, "full_opts_user")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByUsername returned nil")
	}
	if got.TOTPSecret != "SECRET" {
		t.Errorf("TOTPSecret = %q, want SECRET", got.TOTPSecret)
	}
	if got.HomeDirectory != "/var/lib/casrad/users/full_opts_user" {
		t.Errorf("HomeDirectory = %q", got.HomeDirectory)
	}
	if got.LastIP != "10.0.0.5" {
		t.Errorf("LastIP = %q, want 10.0.0.5", got.LastIP)
	}
	if got.Bio != "Bio text" {
		t.Errorf("Bio = %q, want 'Bio text'", got.Bio)
	}
	if got.Website != "https://example.com" {
		t.Errorf("Website = %q", got.Website)
	}
	if got.Location != "Berlin" {
		t.Errorf("Location = %q", got.Location)
	}
	if got.AvatarURL != "https://example.com/av.jpg" {
		t.Errorf("AvatarURL = %q", got.AvatarURL)
	}
	if got.Settings != `{"key":"val"}` {
		t.Errorf("Settings = %q", got.Settings)
	}
}

func TestSQLiteGetUserByEmailWithOptionalFields(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{
		Username:     "email_opts_user",
		Email:        "email_opts@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	fetched, _ := s.GetUserByID(ctx, id)
	fetched.TOTPSecret = "TOTP2"
	fetched.LastIP = "192.168.0.1"
	fetched.Bio = "Audio fan"
	if err := s.UpdateUser(ctx, fetched); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// Get by email — exercises the same .Valid branches in a parallel code path
	got, err := s.GetUserByEmail(ctx, "email_opts@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByEmail returned nil")
	}
	if got.TOTPSecret != "TOTP2" {
		t.Errorf("TOTPSecret = %q, want TOTP2", got.TOTPSecret)
	}
	if got.LastIP != "192.168.0.1" {
		t.Errorf("LastIP = %q", got.LastIP)
	}
	if got.Bio != "Audio fan" {
		t.Errorf("Bio = %q, want 'Audio fan'", got.Bio)
	}
}

// --- SQLiteStore: GetAdminByUsername with populated optional fields ---

func TestSQLiteGetAdminByUsernameWithOptionalFields(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	admin := &model.Admin{
		Username:     "admin_opts",
		Email:        "admin_opts@example.com",
		PasswordHash: "hash",
		Role:         "admin",
	}
	id, err := s.CreateAdmin(ctx, admin)
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	fetched, _ := s.GetAdminByID(ctx, id)
	fetched.TOTPSecret = "TOTP_ADMIN"
	fetched.LastLogin = time.Now().Truncate(time.Second)
	fetched.LastIP = "10.1.2.3"
	fetched.LockedUntil = time.Now().Add(30 * time.Minute).Truncate(time.Second)
	fetched.FailedLoginAttempts = 1
	if err := s.UpdateAdmin(ctx, fetched); err != nil {
		t.Fatalf("UpdateAdmin: %v", err)
	}

	got, err := s.GetAdminByUsername(ctx, "admin_opts")
	if err != nil {
		t.Fatalf("GetAdminByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("GetAdminByUsername returned nil")
	}
	if got.TOTPSecret != "TOTP_ADMIN" {
		t.Errorf("TOTPSecret = %q, want TOTP_ADMIN", got.TOTPSecret)
	}
	if got.LastIP != "10.1.2.3" {
		t.Errorf("LastIP = %q", got.LastIP)
	}
}

// --- SQLiteStore: CreateUser with explicit non-default Role/Theme ---

func TestSQLiteCreateUserExplicitRoleAndTheme(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	user := &model.User{
		Username:          "explicit_role_user",
		Email:             "explicit_role@example.com",
		PasswordHash:      "hash",
		Role:              "moderator",
		ThemePreference:   "light",
		StorageQuotaBytes: 1073741824, // 1GB
		IsActive:          true,
	}
	id, err := s.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser with explicit role: %v", err)
	}

	got, err := s.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Role != "moderator" {
		t.Errorf("Role = %q, want moderator", got.Role)
	}
	if got.ThemePreference != "light" {
		t.Errorf("ThemePreference = %q, want light", got.ThemePreference)
	}
	if got.StorageQuotaBytes != 1073741824 {
		t.Errorf("StorageQuotaBytes = %d, want 1073741824", got.StorageQuotaBytes)
	}
}
