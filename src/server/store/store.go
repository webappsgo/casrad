// Package store provides data access layer
// See AI.md PART 10 for database specification
package store

import (
	"context"

	"github.com/casapps/casrad/src/server/model"
)

// Store defines the data access interface
type Store interface {
	// Admin operations
	GetAdminByID(ctx context.Context, id int64) (*model.Admin, error)
	GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error)
	GetAdminByEmail(ctx context.Context, email string) (*model.Admin, error)
	CreateAdmin(ctx context.Context, admin *model.Admin) (int64, error)
	UpdateAdmin(ctx context.Context, admin *model.Admin) error

	// User operations
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) (int64, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error)

	// Session operations
	GetSession(ctx context.Context, id string) (*model.Session, error)
	CreateSession(ctx context.Context, session *model.Session) error
	UpdateSession(ctx context.Context, session *model.Session) error
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID int64) error

	// Token operations
	GetToken(ctx context.Context, token string) (*model.APIToken, error)
	GetTokenByID(ctx context.Context, id int64) (*model.APIToken, error)
	CreateToken(ctx context.Context, token *model.APIToken) (int64, error)
	DeleteToken(ctx context.Context, id int64) error
	ListUserTokens(ctx context.Context, userID int64) ([]*model.APIToken, error)

	// Database operations
	Close() error
	Ping(ctx context.Context) error
	Migrate(ctx context.Context) error
}
