package identity

import (
	"context"
	"errors"
	"testing"
)

func newTestService(t *testing.T) (*Service, *MemoryRepository) {
	t.Helper()
	users := NewMemoryRepository()
	service, err := NewService(users, NewMemorySessionRepository(users))
	if err != nil {
		t.Fatal(err)
	}
	return service, users
}

func TestServiceRegisterLoginCurrentUserAndLogout(t *testing.T) {
	service, users := newTestService(t)
	ctx := context.Background()
	session, err := service.Register(ctx, RegisterInput{Email: " User@Example.com ", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if session.Token == "" || session.User.Email != "user@example.com" {
		t.Fatalf("unexpected session: %#v", session)
	}
	stored, err := users.FindByEmailWithPasswordHash(ctx, "user@example.com")
	if err != nil || stored.PasswordHash == "password123" {
		t.Fatalf("password was not hashed: %#v, err = %v", stored, err)
	}
	user, err := service.CurrentUser(ctx, session.Token)
	if err != nil || user.ID != session.User.ID {
		t.Fatalf("current user = %#v, err = %v", user, err)
	}
	if err := service.Logout(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CurrentUser(ctx, session.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("current user after logout err = %v", err)
	}
}

func TestServiceRejectsDuplicateAndInvalidCredentials(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	_, err := service.Register(ctx, RegisterInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, RegisterInput{Email: "user@example.com", Password: "password123"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate err = %v", err)
	}
	if _, err := service.Login(ctx, LoginInput{Email: "user@example.com", Password: "wrongpass"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password err = %v", err)
	}
	if _, err := service.Login(ctx, LoginInput{Email: "missing@example.com", Password: "wrongpass"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user err = %v", err)
	}
}
