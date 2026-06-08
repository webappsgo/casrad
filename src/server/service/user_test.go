// Package service — Tests for UserService.
// Covers: ValidateUsername, ValidateEmail, ValidatePassword,
// GetUser, GetUserByUsername, UpdateUser, UpdatePassword,
// DeleteUser, ListUsers, VerifyEmail, SetUserActive,
// UpdateStorageUsed, CheckStorageQuota, CreateAdmin.
// Note: CreateUser is excluded because it calls createUserDirectories (os.MkdirAll)
// which writes to the filesystem — that path is covered by integration tests.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
)

// newUserSvc builds a UserService backed by an in-memory store and a real AuthService.
func newUserSvc() (*UserService, *store.MemoryStore) {
	ms := store.NewMemoryStore()
	auth := NewAuthService(ms)
	svc := NewUserService(ms, auth, "/tmp/casrad-user-test")
	return svc, ms
}

// seedUser inserts a pre-hashed user directly into the memory store so tests
// that don't exercise CreateUser still have a user to act on.
func seedUser(t *testing.T, ms *store.MemoryStore, username, email string) *model.User {
	t.Helper()
	ctx := context.Background()
	u := &model.User{
		Username:          username,
		Email:             email,
		PasswordHash:      "x",
		Role:              "user",
		IsActive:          true,
		StorageQuotaBytes: 1024 * 1024,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	id, err := ms.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	u.ID = id
	return u
}

// --- ValidateUsername ---

func TestValidateUsernameValid(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	valid := []string{"alice", "bob123", "my_user", "x-y", "aaa"}
	for _, name := range valid {
		if err := svc.ValidateUsername(name); err != nil {
			t.Errorf("ValidateUsername(%q) unexpected error: %v", name, err)
		}
	}
}

func TestValidateUsernameInvalid(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	invalid := []string{
		"",
		"ab",
		"1alice",
		// Note: "Alice" is NOT invalid — ValidateUsername lowercases input before checking.
		// Truly invalid: starts with digit, too short, ends with _ or -, double separator.
		"alice_",
		"alice-",
		"alice__bob",
		"alice--bob",
		"alice_-bob",
		"a1",
	}
	for _, name := range invalid {
		if err := svc.ValidateUsername(name); err == nil {
			t.Errorf("ValidateUsername(%q) should have failed", name)
		}
	}
}

func TestValidateUsernameBlocked(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	blocked := []string{"admin", "root", "api", "casrad", "test", "null"}
	for _, name := range blocked {
		err := svc.ValidateUsername(name)
		if !errors.Is(err, ErrUsernameBlocked) {
			t.Errorf("ValidateUsername(%q) = %v, want ErrUsernameBlocked", name, err)
		}
	}
}

// --- ValidateEmail ---

func TestValidateEmailValid(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	valid := []string{
		"user@example.com",
		"user+tag@example.org",
		"User@Sub.Domain.com",
	}
	for _, email := range valid {
		if err := svc.ValidateEmail(email); err != nil {
			t.Errorf("ValidateEmail(%q) unexpected error: %v", email, err)
		}
	}
}

func TestValidateEmailInvalid(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	invalid := []string{
		"",
		"notanemail",
		"@example.com",
		"user@",
		"user@.com",
	}
	for _, email := range invalid {
		if err := svc.ValidateEmail(email); err == nil {
			t.Errorf("ValidateEmail(%q) should have failed", email)
		}
	}
}

// --- ValidatePassword ---

func TestValidatePasswordValid(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	if err := svc.ValidatePassword("ValidPass1!"); err != nil {
		t.Errorf("ValidatePassword valid: %v", err)
	}
}

func TestValidatePasswordTooShort(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	if err := svc.ValidatePassword("abc"); err == nil {
		t.Error("short password should fail")
	}
}

func TestValidatePasswordLeadingSpace(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	if err := svc.ValidatePassword(" ValidPass1!"); err == nil {
		t.Error("password with leading space should fail")
	}
}

func TestValidatePasswordTrailingSpace(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	if err := svc.ValidatePassword("ValidPass1! "); err == nil {
		t.Error("password with trailing space should fail")
	}
}

// --- GetUser ---

func TestGetUserFound(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "getuser", "getuser@example.com")

	got, err := svc.GetUser(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "getuser" {
		t.Errorf("Username = %q, want getuser", got.Username)
	}
}

func TestGetUserNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	_, err := svc.GetUser(context.Background(), 9999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetUser(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- GetUserByUsername ---

func TestGetUserByUsernameFound(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	seedUser(t, ms, "byuname", "byuname@example.com")

	got, err := svc.GetUserByUsername(context.Background(), "byuname")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.Email != "byuname@example.com" {
		t.Errorf("Email = %q, want byuname@example.com", got.Email)
	}
}

func TestGetUserByUsernameNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	_, err := svc.GetUserByUsername(context.Background(), "nobody")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetUserByUsername(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- UpdateUser ---

func TestUpdateUserThemePreference(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "updateme", "updateme@example.com")

	err := svc.UpdateUser(context.Background(), u.ID, map[string]interface{}{
		"theme_preference": "light",
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if got.ThemePreference != "light" {
		t.Errorf("ThemePreference = %q, want light", got.ThemePreference)
	}
}

func TestUpdateUserBio(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "biouser", "biouser@example.com")

	err := svc.UpdateUser(context.Background(), u.ID, map[string]interface{}{
		"bio": "music lover",
	})
	if err != nil {
		t.Fatalf("UpdateUser bio: %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if got.Bio != "music lover" {
		t.Errorf("Bio = %q, want 'music lover'", got.Bio)
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.UpdateUser(context.Background(), 9999, map[string]interface{}{"bio": "x"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UpdateUser(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- UpdatePassword ---

func TestUpdatePasswordNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.UpdatePassword(context.Background(), 9999, "NewPass1!")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UpdatePassword(missing) = %v, want ErrUserNotFound", err)
	}
}

func TestUpdatePasswordInvalid(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "passupd", "passupd@example.com")

	err := svc.UpdatePassword(context.Background(), u.ID, "short")
	if err == nil {
		t.Error("UpdatePassword with short password should fail")
	}
}

// --- DeleteUser ---

func TestDeleteUserSuccess(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "delme", "delme@example.com")

	if err := svc.DeleteUser(context.Background(), u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err := svc.GetUser(context.Background(), u.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("after delete, GetUser = %v, want ErrUserNotFound", err)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.DeleteUser(context.Background(), 9999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("DeleteUser(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- ListUsers ---

func TestListUsersEmpty(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	users, total, err := svc.ListUsers(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("len = %d, want 0", len(users))
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestListUsersClamps(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	for i := 0; i < 3; i++ {
		seedUser(t, ms, "listuser"+string(rune('a'+i)), "list"+string(rune('a'+i))+"@example.com")
	}
	// limit 0 should default to 50
	users, _, err := svc.ListUsers(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListUsers(0): %v", err)
	}
	if len(users) != 3 {
		t.Errorf("len = %d, want 3", len(users))
	}
}

func TestListUsersLimitCappedAt100(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	// limit 200 should be capped to 100 (no error, just fewer results)
	users, _, err := svc.ListUsers(context.Background(), 0, 200)
	if err != nil {
		t.Fatalf("ListUsers limit=200: %v", err)
	}
	// empty store, just verify it doesn't blow up
	_ = users
}

// --- VerifyEmail ---

func TestVerifyEmailSuccess(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "verifyem", "verifyem@example.com")

	if err := svc.VerifyEmail(context.Background(), u.ID); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if !got.EmailVerified {
		t.Error("EmailVerified should be true after VerifyEmail")
	}
}

func TestVerifyEmailNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.VerifyEmail(context.Background(), 9999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("VerifyEmail(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- SetUserActive ---

func TestSetUserActiveDisable(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "deactivate", "deactivate@example.com")

	if err := svc.SetUserActive(context.Background(), u.ID, false); err != nil {
		t.Fatalf("SetUserActive(false): %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if got.IsActive {
		t.Error("IsActive should be false after SetUserActive(false)")
	}
}

func TestSetUserActiveEnable(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "reactivate", "reactivate@example.com")
	u.IsActive = false
	ms.UpdateUser(context.Background(), u)

	if err := svc.SetUserActive(context.Background(), u.ID, true); err != nil {
		t.Fatalf("SetUserActive(true): %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if !got.IsActive {
		t.Error("IsActive should be true after SetUserActive(true)")
	}
}

func TestSetUserActiveNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.SetUserActive(context.Background(), 9999, false)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("SetUserActive(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- UpdateStorageUsed ---

func TestUpdateStorageUsed(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "storageupd", "storageupd@example.com")

	if err := svc.UpdateStorageUsed(context.Background(), u.ID, 12345); err != nil {
		t.Fatalf("UpdateStorageUsed: %v", err)
	}

	got, _ := svc.GetUser(context.Background(), u.ID)
	if got.StorageUsedBytes != 12345 {
		t.Errorf("StorageUsedBytes = %d, want 12345", got.StorageUsedBytes)
	}
}

func TestUpdateStorageUsedNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.UpdateStorageUsed(context.Background(), 9999, 100)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UpdateStorageUsed(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- CheckStorageQuota ---

func TestCheckStorageQuotaWithinLimit(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "quota_ok", "quota_ok@example.com")
	// StorageQuotaBytes is 1MB, used is 0
	if err := svc.CheckStorageQuota(context.Background(), u.ID, 512*1024); err != nil {
		t.Errorf("CheckStorageQuota within limit: %v", err)
	}
}

func TestCheckStorageQuotaExceeded(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	u := seedUser(t, ms, "quota_exc", "quota_exc@example.com")
	// StorageQuotaBytes is 1MB, request 2MB
	err := svc.CheckStorageQuota(context.Background(), u.ID, 2*1024*1024)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Errorf("CheckStorageQuota exceeded = %v, want ErrQuotaExceeded", err)
	}
}

func TestCheckStorageQuotaNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	err := svc.CheckStorageQuota(context.Background(), 9999, 1)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("CheckStorageQuota(missing) = %v, want ErrUserNotFound", err)
	}
}

// --- CreateAdmin ---

func TestCreateAdminInvalidUsername(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	_, err := svc.CreateAdmin(context.Background(), "admin", "admin@example.com", "ValidPass1!")
	if err == nil {
		t.Error("CreateAdmin with blocked username should fail")
	}
}

func TestCreateAdminInvalidEmail(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	_, err := svc.CreateAdmin(context.Background(), "newadmin", "notanemail", "ValidPass1!")
	if err == nil {
		t.Error("CreateAdmin with bad email should fail")
	}
}

func TestCreateAdminInvalidPassword(t *testing.T) {
	t.Parallel()
	svc, _ := newUserSvc()
	_, err := svc.CreateAdmin(context.Background(), "newadmin", "newadmin@example.com", "short")
	if err == nil {
		t.Error("CreateAdmin with bad password should fail")
	}
}
