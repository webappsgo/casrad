// Package service — Additional auth tests covering untested functions.
//
// Coverage targets:
//   - detectIdentifierType: userID branch (numeric string) → was 80%
//   - parseArgon2Hash: bad algorithm, bad version, bad params, bad salt, bad hash fields
//   - ChangePassword: happy path, wrong current password, mismatch confirm, user not found
//   - InvalidateAllUserSessions: happy path
//   - NewAuthServiceWithStore: constructor smoke test (untested at 0%)
//
// All tests in this file use MemoryStore (no network, no filesystem, no database).
// Tests are in package service (same package) to access unexported functions.
package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
)

// --- detectIdentifierType ---

// TestDetectIdentifierTypeEmail verifies that a valid email address is classified as email.
func TestDetectIdentifierTypeEmail(t *testing.T) {
	t.Parallel()

	got := detectIdentifierType("user@example.com")
	if got != identifierEmail {
		t.Errorf("detectIdentifierType(email) = %d, want identifierEmail (%d)", got, identifierEmail)
	}
}

// TestDetectIdentifierTypeUserID verifies that a numeric string is classified as userID.
func TestDetectIdentifierTypeUserID(t *testing.T) {
	t.Parallel()

	cases := []string{"1", "42", "9999", "0"}
	for _, id := range cases {
		got := detectIdentifierType(id)
		if got != identifierUserID {
			t.Errorf("detectIdentifierType(%q) = %d, want identifierUserID (%d)", id, got, identifierUserID)
		}
	}
}

// TestDetectIdentifierTypeUsername verifies that a plain string is classified as username.
func TestDetectIdentifierTypeUsername(t *testing.T) {
	t.Parallel()

	cases := []string{"alice", "bob-42", "user_name", "john.doe"}
	for _, u := range cases {
		got := detectIdentifierType(u)
		if got != identifierUsername {
			t.Errorf("detectIdentifierType(%q) = %d, want identifierUsername (%d)", u, got, identifierUsername)
		}
	}
}

// --- parseArgon2Hash ---

// TestParseArgon2HashValid verifies a correctly formatted hash is parsed without error.
func TestParseArgon2HashValid(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(store.NewMemoryStore())
	hash, err := svc.HashPassword("testpassword!")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	params, salt, h, err := parseArgon2Hash(hash)
	if err != nil {
		t.Fatalf("parseArgon2Hash valid hash error: %v", err)
	}
	if params.memory == 0 {
		t.Error("parsed memory param is 0")
	}
	if params.time == 0 {
		t.Error("parsed time param is 0")
	}
	if params.threads == 0 {
		t.Error("parsed threads param is 0")
	}
	if len(salt) == 0 {
		t.Error("parsed salt is empty")
	}
	if len(h) == 0 {
		t.Error("parsed hash is empty")
	}
}

// TestParseArgon2HashWrongPartCount verifies that hashes with wrong segment count fail.
func TestParseArgon2HashWrongPartCount(t *testing.T) {
	t.Parallel()

	bad := []string{
		"",
		"$argon2id$v=19$m=65536,t=3,p=4$salt",
		"$argon2id$v=19$m=65536,t=3,p=4$salt$hash$extra",
		"notahash",
	}
	for _, b := range bad {
		_, _, _, err := parseArgon2Hash(b)
		if err == nil {
			t.Errorf("parseArgon2Hash(%q) should return error for wrong format", b)
		}
	}
}

// TestParseArgon2HashWrongAlgorithm verifies that a non-argon2id algorithm field fails.
func TestParseArgon2HashWrongAlgorithm(t *testing.T) {
	t.Parallel()

	// Construct a hash with bcrypt algorithm identifier
	h := "$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA"
	_, _, _, err := parseArgon2Hash(h)
	if err == nil {
		t.Error("parseArgon2Hash with wrong algorithm should return error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' error, got: %v", err)
	}
}

// TestParseArgon2HashBadVersion verifies that an unparseable version field fails.
func TestParseArgon2HashBadVersion(t *testing.T) {
	t.Parallel()

	// Version field is not in v=N format
	h := "$argon2id$notaversion$m=65536,t=3,p=4$c2FsdA$aGFzaA"
	_, _, _, err := parseArgon2Hash(h)
	if err == nil {
		t.Error("parseArgon2Hash with bad version should return error")
	}
}

// TestParseArgon2HashBadParams verifies that an unparseable params field fails.
func TestParseArgon2HashBadParams(t *testing.T) {
	t.Parallel()

	// Params field is garbled
	h := "$argon2id$v=19$notparams$c2FsdA$aGFzaA"
	_, _, _, err := parseArgon2Hash(h)
	if err == nil {
		t.Error("parseArgon2Hash with bad params should return error")
	}
}

// TestParseArgon2HashBadSaltEncoding verifies that invalid base64 salt fails.
func TestParseArgon2HashBadSaltEncoding(t *testing.T) {
	t.Parallel()

	// Salt is not valid base64-raw
	h := "$argon2id$v=19$m=65536,t=3,p=4$!!!invalid!!!$aGFzaA"
	_, _, _, err := parseArgon2Hash(h)
	if err == nil {
		t.Error("parseArgon2Hash with invalid salt encoding should return error")
	}
}

// TestParseArgon2HashBadHashEncoding verifies that invalid base64 hash body fails.
func TestParseArgon2HashBadHashEncoding(t *testing.T) {
	t.Parallel()

	// Hash body is not valid base64-raw (salt is valid: "c2FsdA" = "salt")
	h := "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$!!!invalid!!!"
	_, _, _, err := parseArgon2Hash(h)
	if err == nil {
		t.Error("parseArgon2Hash with invalid hash encoding should return error")
	}
}

// --- ChangePassword ---

// newAuthServiceWithUser creates an AuthService backed by MemoryStore with one active user pre-seeded.
// Returns the service, memory store, and the seeded userID.
func newAuthServiceWithUser(t *testing.T, password string) (*AuthService, *store.MemoryStore, int64) {
	t.Helper()
	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword setup error: %v", err)
	}

	id, err := ms.CreateUser(ctx, &model.User{
		Username:     "pwuser",
		Email:        "pwuser@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateUser setup error: %v", err)
	}

	return svc, ms, id
}

// TestChangePasswordHappyPath verifies a valid current password with matching new passwords succeeds.
func TestChangePasswordHappyPath(t *testing.T) {
	t.Parallel()

	oldPw := "Old@P@ss1!"
	newPw := "New@P@ss1!"
	svc, _, userID := newAuthServiceWithUser(t, oldPw)
	ctx := context.Background()

	result, err := svc.ChangePassword(ctx, userID, oldPw, newPw, newPw)
	if err != nil {
		t.Fatalf("ChangePassword error: %v", err)
	}
	if result != nil && result.HasErrors() {
		t.Errorf("ChangePassword returned validation errors: %v", result.Errors)
	}

	// Verify old password no longer works
	user, _ := svc.store.GetUserByID(ctx, userID)
	if svc.VerifyPassword(oldPw, user.PasswordHash) {
		t.Error("old password should no longer verify after ChangePassword")
	}

	// Verify new password works
	if !svc.VerifyPassword(newPw, user.PasswordHash) {
		t.Error("new password should verify after ChangePassword")
	}
}

// TestChangePasswordWrongCurrent verifies that supplying the wrong current password fails gracefully.
func TestChangePasswordWrongCurrent(t *testing.T) {
	t.Parallel()

	svc, _, userID := newAuthServiceWithUser(t, "Correct@P@ss1!")
	ctx := context.Background()

	result, err := svc.ChangePassword(ctx, userID, "wrong_password!", "New@P@ss1!", "New@P@ss1!")
	if err != nil {
		t.Fatalf("ChangePassword unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ChangePassword should return ValidationResult with errors for wrong current password")
	}
	if !result.HasErrors() {
		t.Error("ChangePassword with wrong current password should have validation errors")
	}
}

// TestChangePasswordConfirmMismatch verifies that mismatched new passwords fail validation.
func TestChangePasswordConfirmMismatch(t *testing.T) {
	t.Parallel()

	svc, _, userID := newAuthServiceWithUser(t, "Current@P@ss1!")
	ctx := context.Background()

	result, err := svc.ChangePassword(ctx, userID, "Current@P@ss1!", "New@P@ss1!", "Different@P@ss1!")
	if err != nil {
		t.Fatalf("ChangePassword unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ChangePassword should return ValidationResult for mismatched passwords")
	}
	if !result.HasErrors() {
		t.Error("ChangePassword with mismatched confirm should have errors")
	}
}

// TestChangePasswordUserNotFound verifies that a non-existent userID returns an error.
func TestChangePasswordUserNotFound(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	// No users seeded — userID 99 does not exist
	_, err := svc.ChangePassword(ctx, 99, "any@pass1!", "new@Pass1!", "new@Pass1!")
	if err == nil {
		t.Error("ChangePassword with non-existent userID should return error")
	}
}

// TestChangePasswordWeakNewPassword verifies that a new password failing policy is rejected.
func TestChangePasswordWeakNewPassword(t *testing.T) {
	t.Parallel()

	svc, _, userID := newAuthServiceWithUser(t, "Strong@P@ss1!")
	ctx := context.Background()

	// "short" is too short and missing complexity
	result, err := svc.ChangePassword(ctx, userID, "Strong@P@ss1!", "short", "short")
	if err != nil {
		t.Fatalf("ChangePassword unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ChangePassword with weak new password should return ValidationResult")
	}
	if !result.HasErrors() {
		t.Error("ChangePassword with weak new password should have validation errors")
	}
}

// --- InvalidateAllUserSessions ---

// TestInvalidateAllUserSessionsRemovesSessions verifies that all sessions for a user are deleted.
func TestInvalidateAllUserSessionsRemovesSessions(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	// Create two sessions for userID=7
	sid1, err := svc.CreateSession(ctx, 7, 0, "1.2.3.4", "agent1")
	if err != nil {
		t.Fatalf("CreateSession 1 error: %v", err)
	}
	sid2, err := svc.CreateSession(ctx, 7, 0, "1.2.3.5", "agent2")
	if err != nil {
		t.Fatalf("CreateSession 2 error: %v", err)
	}

	// Invalidate all sessions for user 7
	if err := svc.InvalidateAllUserSessions(ctx, 7); err != nil {
		t.Fatalf("InvalidateAllUserSessions error: %v", err)
	}

	// Both sessions should now be invalid
	for _, sid := range []string{sid1, sid2} {
		_, _, err := svc.ValidateSession(ctx, sid)
		if err != ErrInvalidSession {
			t.Errorf("session %q should be invalid after InvalidateAllUserSessions, got: %v", sid, err)
		}
	}
}

// --- Authenticate via admin path ---

// TestAuthenticateAdminByUsername verifies admin login by username.
func TestAuthenticateAdminByUsername(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "adminuser",
		Email:        "admin@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	userID, adminID, err := svc.Authenticate(ctx, "adminuser", password, "10.0.0.1")
	if err != nil {
		t.Fatalf("Authenticate admin error: %v", err)
	}
	if userID != 0 {
		t.Errorf("Authenticate admin userID = %d, want 0", userID)
	}
	if adminID <= 0 {
		t.Errorf("Authenticate admin adminID = %d, want >0", adminID)
	}
}

// TestAuthenticateAdminByEmail verifies admin login by email address.
func TestAuthenticateAdminByEmail(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "adminbyemail",
		Email:        "adminbyemail@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	userID, adminID, err := svc.Authenticate(ctx, "adminbyemail@example.com", password, "10.0.0.1")
	if err != nil {
		t.Fatalf("Authenticate admin by email error: %v", err)
	}
	if userID != 0 {
		t.Errorf("Authenticate admin by email userID = %d, want 0", userID)
	}
	if adminID <= 0 {
		t.Errorf("Authenticate admin by email adminID = %d, want >0", adminID)
	}
}

// TestAuthenticateLockedAdmin verifies that a locked admin is rejected.
func TestAuthenticateLockedAdmin(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "lockedadmin",
		Email:        "lockedadmin@example.com",
		PasswordHash: hash,
		IsActive:     true,
		// Locked for the next hour
		LockedUntil: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	_, _, err = svc.Authenticate(ctx, "lockedadmin", password, "10.0.0.1")
	if err != ErrAccountLocked {
		t.Errorf("locked admin = %v, want ErrAccountLocked", err)
	}
}

// TestAuthenticateDisabledAdmin verifies that an inactive admin is rejected.
func TestAuthenticateDisabledAdmin(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "inactiveadmin",
		Email:        "inactiveadmin@example.com",
		PasswordHash: hash,
		// IsActive: false (zero value)
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	_, _, err = svc.Authenticate(ctx, "inactiveadmin", password, "10.0.0.1")
	if err != ErrAccountDisabled {
		t.Errorf("inactive admin = %v, want ErrAccountDisabled", err)
	}
}

// TestAuthenticateAdminWrongPassword verifies that wrong admin password returns ErrInvalidCredentials.
func TestAuthenticateAdminWrongPassword(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "wrongpwadmin",
		Email:        "wrongpwadmin@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	_, _, err = svc.Authenticate(ctx, "wrongpwadmin", "wrongpassword!", "10.0.0.1")
	if err != ErrInvalidCredentials {
		t.Errorf("admin wrong password = %v, want ErrInvalidCredentials", err)
	}
}

// TestAuthenticateLockedUser verifies that a locked user is rejected.
func TestAuthenticateLockedUser(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "User@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateUser(ctx, &model.User{
		Username:     "lockeduser",
		Email:        "lockeduser@example.com",
		PasswordHash: hash,
		IsActive:     true,
		LockedUntil:  time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateUser setup error: %v", err)
	}

	_, _, err = svc.Authenticate(ctx, "lockeduser", password, "10.0.0.1")
	if err != ErrAccountLocked {
		t.Errorf("locked user = %v, want ErrAccountLocked", err)
	}
}

// TestAuthenticateBruteForceLocksAdmin verifies that 5 failed admin login attempts trigger lockout.
func TestAuthenticateBruteForceLocksAdmin(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "Admin@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateAdmin(ctx, &model.Admin{
		Username:     "bruteadmin",
		Email:        "bruteadmin@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateAdmin setup error: %v", err)
	}

	// 5 failed attempts
	for i := 0; i < 5; i++ {
		_, _, _ = svc.Authenticate(ctx, "bruteadmin", "wrongpassword!", "10.0.0.1")
	}

	// Now correct password should return locked
	_, _, err = svc.Authenticate(ctx, "bruteadmin", password, "10.0.0.1")
	if err != ErrAccountLocked {
		t.Errorf("after 5 failures admin = %v, want ErrAccountLocked", err)
	}
}

// TestAuthenticateBruteForceLocksUser verifies that 5 failed user login attempts trigger lockout.
func TestAuthenticateBruteForceLocksUser(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "User@P@ss1!"
	hash, _ := svc.HashPassword(password)

	_, err := ms.CreateUser(ctx, &model.User{
		Username:     "bruteuser",
		Email:        "bruteuser@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreateUser setup error: %v", err)
	}

	// 5 failed attempts
	for i := 0; i < 5; i++ {
		_, _, _ = svc.Authenticate(ctx, "bruteuser", "wrongpassword!", "10.0.0.1")
	}

	// Now correct password should return locked
	_, _, err = svc.Authenticate(ctx, "bruteuser", password, "10.0.0.1")
	if err != ErrAccountLocked {
		t.Errorf("after 5 failures user = %v, want ErrAccountLocked", err)
	}
}

// TestCreateAdminSession verifies session creation for an admin (adminID != 0, userID == 0).
func TestCreateAdminSession(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	sessionID, err := svc.CreateSession(ctx, 0, 42, "10.0.0.1", "AdminClient/1.0")
	if err != nil {
		t.Fatalf("CreateSession admin error: %v", err)
	}
	if sessionID == "" {
		t.Fatal("CreateSession returned empty sessionID")
	}

	userID, adminID, err := svc.ValidateSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ValidateSession error: %v", err)
	}
	if userID != 0 {
		t.Errorf("admin session userID = %d, want 0", userID)
	}
	if adminID != 42 {
		t.Errorf("admin session adminID = %d, want 42", adminID)
	}
}

// TestGenerateAPITokenWithPrefixUsr verifies tokens with user prefix start correctly and hash verifies.
func TestGenerateAPITokenWithPrefixUsr(t *testing.T) {
	t.Parallel()

	token, hash, err := GenerateAPITokenWithPrefix(TokenPrefixUser)
	if err != nil {
		t.Fatalf("GenerateAPITokenWithPrefix(usr) error: %v", err)
	}

	if !strings.HasPrefix(token, TokenPrefixUser) {
		t.Errorf("token %q should start with %q", token, TokenPrefixUser)
	}

	if !VerifyToken(token, hash) {
		t.Error("generated user token hash does not verify")
	}
}

// TestHashTokenLength verifies SHA-256 base64-raw output is 43 characters.
func TestHashTokenLength(t *testing.T) {
	t.Parallel()

	h := HashToken("anytokentouse")
	// SHA-256 = 32 bytes → base64-raw = ceil(32*8/6) = 43 chars
	if len(h) != 43 {
		t.Errorf("HashToken length = %d, want 43", len(h))
	}
}
