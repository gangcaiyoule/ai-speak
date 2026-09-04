package identity

import (
	"context"
	"time"
)

type Repository interface {
	CreateWithPasswordHash(context.Context, StoredUser) (User, error)
	FindByEmailWithPasswordHash(context.Context, string) (StoredUser, error)
	FindByID(context.Context, string) (User, error)
	CreateSession(context.Context, AuthSession) error
	FindSessionByTokenDigest(context.Context, string) (AuthSession, error)
	RevokeSession(context.Context, string, time.Time) error
}

type AuthService interface {
	Register(context.Context, RegisterInput) (User, error)
	Login(context.Context, LoginInput) (LoginResult, error)
	Authenticate(context.Context, string) (Actor, error)
	CurrentUser(context.Context, string) (User, error)
	Logout(context.Context, string) error
}
