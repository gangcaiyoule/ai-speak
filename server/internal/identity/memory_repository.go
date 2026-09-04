package identity

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	users    map[string]StoredUser
	byEmail  map[string]string
	sessions map[string]AuthSession
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: map[string]StoredUser{}, byEmail: map[string]string{}, sessions: map[string]AuthSession{}}
}
func (r *MemoryRepository) CreateWithPasswordHash(ctx context.Context, u StoredUser) (User, error) {
	if e := ctx.Err(); e != nil {
		return User{}, e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byEmail[u.User.Email]; ok {
		return User{}, ErrConflict
	}
	if u.User.ID == "" {
		t, e := newToken()
		if e != nil {
			return User{}, e
		}
		u.User.ID = t[5:]
	}
	r.users[u.User.ID] = u
	r.byEmail[u.User.Email] = u.User.ID
	return u.User, nil
}
func (r *MemoryRepository) FindByEmailWithPasswordHash(ctx context.Context, e string) (StoredUser, error) {
	if x := ctx.Err(); x != nil {
		return StoredUser{}, x
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[e]
	if !ok {
		return StoredUser{}, ErrInvalidCredentials
	}
	return r.users[id], nil
}
func (r *MemoryRepository) FindByID(ctx context.Context, id string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u.User, nil
}
func (r *MemoryRepository) CreateSession(ctx context.Context, s AuthSession) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
	return nil
}
func (r *MemoryRepository) FindSessionByTokenDigest(ctx context.Context, d string) (AuthSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		if s.TokenDigest == d {
			return s, nil
		}
	}
	return AuthSession{}, ErrUnauthorized
}
func (r *MemoryRepository) RevokeSession(ctx context.Context, id string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil
	}
	if s.RevokedAt == nil {
		s.RevokedAt = &now
		r.sessions[id] = s
	}
	return nil
}
