//go:build integration

package identity

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestPostgresRepositoryPersistsAuthenticationFlow(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration test")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(NewPostgresRepository(db))
	ctx := context.Background()
	registered, err := service.Register(ctx, RegisterInput{Email: "persisted@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := service.Login(ctx, LoginInput{Email: registered.Email, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Authenticate(ctx, login.Token); err != nil {
		t.Fatal(err)
	}
	if err = service.Logout(ctx, mustActor(t, service, ctx, login.Token).SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Authenticate(ctx, login.Token); err == nil {
		t.Fatal("revoked token authenticated")
	}
}

func mustActor(t *testing.T, service *Service, ctx context.Context, token string) Actor {
	t.Helper()
	actor, err := service.Authenticate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}
