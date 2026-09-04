package identity

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestServiceRegistersOnlyOneCanonicalEmailConcurrently(t *testing.T) {
	service := NewService(NewMemoryRepository())
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, email := range []string{"User@example.com", " user@example.com "} {
		wg.Add(1)
		go func(email string) {
			defer wg.Done()
			_, err := service.Register(context.Background(), RegisterInput{Email: email, Password: "password123"})
			results <- err
		}(email)
	}
	wg.Wait()
	close(results)
	var success, conflicts int
	for err := range results {
		if err == nil {
			success++
			continue
		}
		if errors.Is(err, ErrConflict) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected registration error: %v", err)
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
}

func TestNewPostgresRepositoryAcceptsNilDatabase(t *testing.T) {
	if _, err := NewPostgresRepository(nil).FindByID(context.Background(), "user"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
