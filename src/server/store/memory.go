// Package store - Memory-based store for development and testing
// See AI.md PART 3 for cache driver specification
package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/casapps/casrad/src/server/model"
)

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	mu sync.RWMutex

	// Data storage
	admins    map[int64]*model.Admin
	users     map[int64]*model.User
	sessions  map[string]*model.Session
	tokens    map[int64]*model.APIToken
	// token string -> id
	tokenByID map[string]int64

	// ID counters
	nextAdminID int64
	nextUserID  int64
	nextTokenID int64
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		admins:      make(map[int64]*model.Admin),
		users:       make(map[int64]*model.User),
		sessions:    make(map[string]*model.Session),
		tokens:      make(map[int64]*model.APIToken),
		tokenByID:   make(map[string]int64),
		nextAdminID: 1,
		nextUserID:  1,
		nextTokenID: 1,
	}
}

// Close is a no-op for memory store
func (s *MemoryStore) Close() error {
	return nil
}

// Ping always succeeds for memory store
func (s *MemoryStore) Ping(ctx context.Context) error {
	return nil
}

// Migrate is a no-op for memory store
func (s *MemoryStore) Migrate(ctx context.Context) error {
	return nil
}

// Admin operations

func (s *MemoryStore) GetAdminByID(ctx context.Context, id int64) (*model.Admin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	admin, ok := s.admins[id]
	if !ok {
		return nil, nil
	}
	return copyAdmin(admin), nil
}

func (s *MemoryStore) GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, admin := range s.admins {
		if admin.Username == username {
			return copyAdmin(admin), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) GetAdminByEmail(ctx context.Context, email string) (*model.Admin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, admin := range s.admins {
		if admin.Email == email {
			return copyAdmin(admin), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) CreateAdmin(ctx context.Context, admin *model.Admin) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	for _, existing := range s.admins {
		if existing.Username == admin.Username {
			return 0, errors.New("username already exists")
		}
		if existing.Email == admin.Email {
			return 0, errors.New("email already exists")
		}
	}

	id := s.nextAdminID
	s.nextAdminID++

	now := time.Now()
	newAdmin := copyAdmin(admin)
	newAdmin.ID = id
	newAdmin.CreatedAt = now
	newAdmin.UpdatedAt = now

	s.admins[id] = newAdmin
	return id, nil
}

func (s *MemoryStore) UpdateAdmin(ctx context.Context, admin *model.Admin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.admins[admin.ID]; !ok {
		return errors.New("admin not found")
	}

	updated := copyAdmin(admin)
	updated.UpdatedAt = time.Now()
	s.admins[admin.ID] = updated
	return nil
}

// User operations

func (s *MemoryStore) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, nil
	}
	return copyUser(user), nil
}

func (s *MemoryStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Username == username {
			return copyUser(user), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return copyUser(user), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) CreateUser(ctx context.Context, user *model.User) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	for _, existing := range s.users {
		if existing.Username == user.Username {
			return 0, errors.New("username already exists")
		}
		if existing.Email == user.Email {
			return 0, errors.New("email already exists")
		}
	}

	id := s.nextUserID
	s.nextUserID++

	now := time.Now()
	newUser := copyUser(user)
	newUser.ID = id
	newUser.CreatedAt = now
	newUser.UpdatedAt = now

	// Set defaults
	if newUser.Role == "" {
		newUser.Role = "user"
	}
	if newUser.ThemePreference == "" {
		newUser.ThemePreference = "dark"
	}
	if newUser.StorageQuotaBytes == 0 {
		// 50GB default
		newUser.StorageQuotaBytes = 53687091200
	}

	s.users[id] = newUser
	return id, nil
}

func (s *MemoryStore) UpdateUser(ctx context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[user.ID]; !ok {
		return errors.New("user not found")
	}

	updated := copyUser(user)
	updated.UpdatedAt = time.Now()
	s.users[user.ID] = updated
	return nil
}

func (s *MemoryStore) DeleteUser(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, id)
	return nil
}

func (s *MemoryStore) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := int64(len(s.users))

	// Convert to slice and sort by ID
	users := make([]*model.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, copyUser(user))
	}

	// Apply offset and limit
	start := offset
	if start > len(users) {
		start = len(users)
	}
	end := start + limit
	if end > len(users) {
		end = len(users)
	}

	return users[start:end], total, nil
}

// Session operations

func (s *MemoryStore) GetSession(ctx context.Context, id string) (*model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Hash the raw session ID — never store raw tokens per AI.md PART 11
	session, ok := s.sessions[hashForStorage(id)]
	if !ok {
		return nil, nil
	}
	return copySession(session), nil
}

func (s *MemoryStore) CreateSession(ctx context.Context, session *model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	newSession := copySession(session)
	if newSession.CreatedAt.IsZero() {
		newSession.CreatedAt = now
	}
	if newSession.LastActivity.IsZero() {
		newSession.LastActivity = now
	}

	// Hash the raw session ID — never store raw tokens per AI.md PART 11
	s.sessions[hashForStorage(session.ID)] = newSession
	return nil
}

func (s *MemoryStore) UpdateSession(ctx context.Context, session *model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Hash the raw session ID — never store raw tokens per AI.md PART 11
	key := hashForStorage(session.ID)
	if _, ok := s.sessions[key]; !ok {
		return errors.New("session not found")
	}

	updated := copySession(session)
	updated.LastActivity = time.Now()
	s.sessions[key] = updated
	return nil
}

func (s *MemoryStore) DeleteSession(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Hash the raw session ID — never store raw tokens per AI.md PART 11
	delete(s.sessions, hashForStorage(id))
	return nil
}

func (s *MemoryStore) DeleteUserSessions(ctx context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, id)
		}
	}
	return nil
}

// Token operations

func (s *MemoryStore) GetToken(ctx context.Context, token string) (*model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Hash the raw token — never store raw tokens per AI.md PART 11
	id, ok := s.tokenByID[hashForStorage(token)]
	if !ok {
		return nil, nil
	}

	apiToken, ok := s.tokens[id]
	if !ok {
		return nil, nil
	}
	return copyToken(apiToken), nil
}

func (s *MemoryStore) GetTokenByID(ctx context.Context, id int64) (*model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[id]
	if !ok {
		return nil, nil
	}
	return copyToken(token), nil
}

func (s *MemoryStore) CreateToken(ctx context.Context, token *model.APIToken) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextTokenID
	s.nextTokenID++

	newToken := copyToken(token)
	newToken.ID = id
	newToken.CreatedAt = time.Now()

	s.tokens[id] = newToken
	// Hash the raw token — never store raw tokens per AI.md PART 11
	s.tokenByID[hashForStorage(token.Token)] = id
	return id, nil
}

func (s *MemoryStore) DeleteToken(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token, ok := s.tokens[id]; ok {
		// tokenByID is keyed by hash — delete using hash of stored token
		delete(s.tokenByID, hashForStorage(token.Token))
	}
	delete(s.tokens, id)
	return nil
}

func (s *MemoryStore) ListUserTokens(ctx context.Context, userID int64) ([]*model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tokens []*model.APIToken
	for _, token := range s.tokens {
		if token.UserID == userID {
			tokens = append(tokens, copyToken(token))
		}
	}
	return tokens, nil
}

// Helper functions for deep copying

func copyAdmin(a *model.Admin) *model.Admin {
	if a == nil {
		return nil
	}
	copy := *a
	return &copy
}

func copyUser(u *model.User) *model.User {
	if u == nil {
		return nil
	}
	copy := *u
	return &copy
}

func copySession(s *model.Session) *model.Session {
	if s == nil {
		return nil
	}
	copy := *s
	return &copy
}

func copyToken(t *model.APIToken) *model.APIToken {
	if t == nil {
		return nil
	}
	copy := *t
	return &copy
}
