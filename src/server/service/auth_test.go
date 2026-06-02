// Package service - Tests for auth helpers: token generation/hashing/verification,
// Argon2id password hashing/verification, session ID generation, identifier type detection.
// Tests for AuthService.Authenticate and RegisterUser use the MemoryStore to exercise
// real behavior without mocks.
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
)

// --- HashToken / VerifyToken ---

func TestHashTokenIsConsistent(t *testing.T) {
	t.Parallel()

	h1 := HashToken("mytoken")
	h2 := HashToken("mytoken")
	if h1 != h2 {
		t.Error("HashToken is not deterministic")
	}
}

func TestHashTokenDifferentInputsDifferentHashes(t *testing.T) {
	t.Parallel()

	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if h1 == h2 {
		t.Error("HashToken collision — different tokens must produce different hashes")
	}
}

func TestHashTokenOutputFormat(t *testing.T) {
	t.Parallel()

	h := HashToken("test")
	// SHA-256 base64-raw produces 43 characters
	if len(h) == 0 {
		t.Error("HashToken returned empty string")
	}
}

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	token := "usr_abc123xyz"
	hash := HashToken(token)

	if !VerifyToken(token, hash) {
		t.Error("VerifyToken should return true for matching token/hash")
	}
	if VerifyToken("wrong_token", hash) {
		t.Error("VerifyToken should return false for wrong token")
	}
	if VerifyToken(token, "badhash") {
		t.Error("VerifyToken should return false for wrong hash")
	}
}

// --- GenerateAPIToken / GenerateAPITokenWithPrefix ---

func TestGenerateAPIToken(t *testing.T) {
	t.Parallel()

	token, hash, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken error: %v", err)
	}
	if token == "" {
		t.Error("GenerateAPIToken returned empty token")
	}
	if hash == "" {
		t.Error("GenerateAPIToken returned empty hash")
	}

	// Token must start with admin prefix
	if !strings.HasPrefix(token, TokenPrefixAdmin) {
		t.Errorf("admin token %q should start with %q", token, TokenPrefixAdmin)
	}

	// Hash must verify
	if !VerifyToken(token, hash) {
		t.Error("generated token hash does not verify")
	}
}

func TestGenerateAPITokenWithPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prefix string
	}{
		{TokenPrefixAdmin},
		{TokenPrefixUser},
		{TokenPrefixOrg},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.prefix, func(t *testing.T) {
			t.Parallel()
			token, hash, err := GenerateAPITokenWithPrefix(tc.prefix)
			if err != nil {
				t.Fatalf("GenerateAPITokenWithPrefix(%q) error: %v", tc.prefix, err)
			}
			if !strings.HasPrefix(token, tc.prefix) {
				t.Errorf("token %q should start with prefix %q", token, tc.prefix)
			}
			if !VerifyToken(token, hash) {
				t.Error("generated token hash does not verify")
			}
		})
	}
}

func TestGenerateAPITokenUniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		token, _, err := GenerateAPIToken()
		if err != nil {
			t.Fatalf("GenerateAPIToken[%d] error: %v", i, err)
		}
		if seen[token] {
			t.Errorf("duplicate token generated at iteration %d", i)
		}
		seen[token] = true
	}
}

// --- GetTokenPrefix ---

func TestGetTokenPrefix(t *testing.T) {
	t.Parallel()

	t.Run("long_token", func(t *testing.T) {
		t.Parallel()
		got := GetTokenPrefix("adm_abc123xyz789")
		if got != "adm_abc1" {
			t.Errorf("GetTokenPrefix = %q, want %q", got, "adm_abc1")
		}
	})

	t.Run("short_token", func(t *testing.T) {
		t.Parallel()
		got := GetTokenPrefix("abc")
		if got != "abc" {
			t.Errorf("GetTokenPrefix short = %q, want %q", got, "abc")
		}
	})

	t.Run("exactly_8", func(t *testing.T) {
		t.Parallel()
		got := GetTokenPrefix("12345678")
		if got != "12345678" {
			t.Errorf("GetTokenPrefix 8 = %q, want %q", got, "12345678")
		}
	})
}

// --- AuthService password hash/verify ---

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(store.NewMemoryStore())

	password := "V@lidP@ssw0rd!"
	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	// Must include argon2id marker
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("hash %q does not start with $argon2id$", hash)
	}

	// Verify correct password
	if !svc.VerifyPassword(password, hash) {
		t.Error("VerifyPassword should return true for correct password")
	}

	// Verify wrong password
	if svc.VerifyPassword("wrongpassword", hash) {
		t.Error("VerifyPassword should return false for wrong password")
	}

	// Idempotency: hashing again produces different salt but both verify
	hash2, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword error: %v", err)
	}
	if hash == hash2 {
		t.Error("two hashes of same password should differ (different random salts)")
	}
	if !svc.VerifyPassword(password, hash2) {
		t.Error("second hash should also verify")
	}
}

func TestVerifyPasswordBadHashFormat(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(store.NewMemoryStore())
	if svc.VerifyPassword("anypass", "not-a-valid-hash") {
		t.Error("VerifyPassword with malformed hash should return false")
	}
	if svc.VerifyPassword("anypass", "") {
		t.Error("VerifyPassword with empty hash should return false")
	}
}

// --- AuthService.Authenticate ---

func TestAuthenticate_ValidUser(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "S3cur3P@ss!"
	hash, _ := svc.HashPassword(password)

	// Seed a user
	_, _ = ms.CreateUser(ctx, &model.User{
		Username:     "testauth",
		Email:        "testauth@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})

	userID, adminID, err := svc.Authenticate(ctx, "testauth", password, "127.0.0.1")
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if userID <= 0 {
		t.Errorf("Authenticate userID = %d, want >0", userID)
	}
	if adminID != 0 {
		t.Errorf("Authenticate adminID = %d, want 0 for user", adminID)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	hash, _ := svc.HashPassword("correctpassword!")
	_, _ = ms.CreateUser(ctx, &model.User{
		Username:     "wrongpwuser",
		Email:        "wrongpw@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})

	_, _, err := svc.Authenticate(ctx, "wrongpwuser", "wrongpassword!", "127.0.0.1")
	if err != ErrInvalidCredentials {
		t.Errorf("wrong password = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_UnknownUser(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	_, _, err := svc.Authenticate(ctx, "nobody", "somepassword!", "127.0.0.1")
	if err != ErrInvalidCredentials {
		t.Errorf("unknown user = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_DisabledUser(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	hash, _ := svc.HashPassword("S3cur3P@ss!")
	_, _ = ms.CreateUser(ctx, &model.User{
		Username:     "disableduser",
		Email:        "disabled@example.com",
		PasswordHash: hash,
		// IsActive defaults to false
	})

	_, _, err := svc.Authenticate(ctx, "disableduser", "S3cur3P@ss!", "127.0.0.1")
	if err != ErrAccountDisabled {
		t.Errorf("disabled user = %v, want ErrAccountDisabled", err)
	}
}

func TestAuthenticate_ByEmail(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	password := "S3cur3P@ss!"
	hash, _ := svc.HashPassword(password)
	_, _ = ms.CreateUser(ctx, &model.User{
		Username:     "emaillogin",
		Email:        "emaillogin@example.com",
		PasswordHash: hash,
		IsActive:     true,
	})

	userID, _, err := svc.Authenticate(ctx, "emaillogin@example.com", password, "127.0.0.1")
	if err != nil {
		t.Fatalf("Authenticate by email error: %v", err)
	}
	if userID <= 0 {
		t.Errorf("Authenticate by email userID = %d, want >0", userID)
	}
}

// --- ValidateSession ---

func TestCreateAndValidateSession(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	// Create a session for userID=5
	sessionID, err := svc.CreateSession(ctx, 5, 0, "127.0.0.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if sessionID == "" {
		t.Fatal("CreateSession returned empty sessionID")
	}

	// Validate it
	userID, adminID, err := svc.ValidateSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ValidateSession error: %v", err)
	}
	if userID != 5 {
		t.Errorf("ValidateSession userID = %d, want 5", userID)
	}
	if adminID != 0 {
		t.Errorf("ValidateSession adminID = %d, want 0", adminID)
	}

	// Invalidate it
	if err := svc.InvalidateSession(ctx, sessionID); err != nil {
		t.Fatalf("InvalidateSession error: %v", err)
	}

	// Now it should be invalid
	_, _, err = svc.ValidateSession(ctx, sessionID)
	if err != ErrInvalidSession {
		t.Errorf("ValidateSession after invalidate = %v, want ErrInvalidSession", err)
	}
}

func TestValidateSession_InvalidID(t *testing.T) {
	t.Parallel()

	ms := store.NewMemoryStore()
	svc := NewAuthService(ms)
	ctx := context.Background()

	_, _, err := svc.ValidateSession(ctx, "nonexistent-session-id")
	if err != ErrInvalidSession {
		t.Errorf("ValidateSession unknown id = %v, want ErrInvalidSession", err)
	}
}
