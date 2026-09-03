package identity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
)

// MemoryRepository 是本 Issue 使用的线程安全内存实现，后续可替换为 PostgreSQL。
type MemoryRepository struct {
	mu       sync.RWMutex
	users    map[string]StoredUser
	byEmail  map[string]string
	sessions map[string]Session
	digests  map[string]string
}

// MemorySessionRepository 将内存用户仓库的会话能力适配为独立端口。
type MemorySessionRepository struct{ owner *MemoryRepository }

// NewMemorySessionRepository 创建与用户仓库共享状态的会话仓库。
func NewMemorySessionRepository(owner *MemoryRepository) *MemorySessionRepository {
	if owner == nil {
		panic("identity: memory repository is required")
	}
	return &MemorySessionRepository{owner: owner}
}

// NewMemoryRepository 创建空的内存仓库。
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: map[string]StoredUser{}, byEmail: map[string]string{}, sessions: map[string]Session{}, digests: map[string]string{}}
}

func (r *MemoryRepository) Create(ctx context.Context, user User) (User, error) {
	return r.CreateWithPasswordHash(ctx, StoredUser{User: user})
}

func (r *MemoryRepository) CreateWithPasswordHash(ctx context.Context, user StoredUser) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[user.User.Email]; exists {
		return User{}, ErrConflict
	}
	r.users[user.User.ID] = user
	r.byEmail[user.User.Email] = user.User.ID
	return user.User, nil
}

func (r *MemoryRepository) FindByID(ctx context.Context, id string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user.User, nil
}

func (r *MemoryRepository) FindByEmailWithPasswordHash(ctx context.Context, email string) (StoredUser, error) {
	if err := ctx.Err(); err != nil {
		return StoredUser{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return StoredUser{}, ErrNotFound
	}
	return r.users[id], nil
}

func (r *MemoryRepository) CreateSession(ctx context.Context, session Session) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.digests[session.ID] = tokenDigest(session.Token)
	stored := session
	stored.Token = ""
	r.sessions[session.ID] = stored
	return session, nil
}

// Create 保存会话。
func (r *MemorySessionRepository) Create(ctx context.Context, session Session) (Session, error) {
	return r.owner.CreateSession(ctx, session)
}

// Delete 删除会话。
func (r *MemorySessionRepository) Delete(ctx context.Context, id string) error {
	return r.owner.Delete(ctx, id)
}

// FindByTokenDigest 按原始 Token 查询会话。
func (r *MemorySessionRepository) FindByTokenDigest(ctx context.Context, token string) (Session, error) {
	return r.owner.FindByTokenDigest(ctx, token)
}

func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
	delete(r.digests, id)
	return nil
}

func (r *MemoryRepository) FindByTokenDigest(ctx context.Context, digest string) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, session := range r.sessions {
		if r.digests[id] == tokenDigest(digest) {
			session.ID = id
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func tokenDigest(token string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(token))) }
