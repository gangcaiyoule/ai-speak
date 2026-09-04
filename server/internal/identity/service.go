package identity

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
	ttl  time.Duration
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, ttl: 30 * 24 * time.Hour}
}
func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	e, err := canonicalEmail(in.Email)
	if err != nil {
		return User{}, err
	}
	if err = validatePassword(in.Password); err != nil {
		return User{}, err
	}
	h, err := hashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	id, err := newIdentifier()
	if err != nil {
		return User{}, err
	}
	return s.repo.CreateWithPasswordHash(ctx, StoredUser{User: User{ID: id, Email: e}, PasswordHash: h})
}
func (s *Service) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
	e, err := canonicalEmail(in.Email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	u, err := s.repo.FindByEmailWithPasswordHash(ctx, e)
	if err != nil || !verifyPassword(u.PasswordHash, in.Password) {
		return LoginResult{}, ErrInvalidCredentials
	}
	raw, err := newToken()
	if err != nil {
		return LoginResult{}, err
	}
	id, err := newIdentifier()
	if err != nil {
		return LoginResult{}, err
	}
	now := s.now()
	exp := now.Add(s.ttl)
	if err = s.repo.CreateSession(ctx, AuthSession{ID: id, UserID: u.User.ID, TokenDigest: tokenDigest(raw), CreatedAt: now, ExpiresAt: exp}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: u.User, Token: raw, TokenType: "Bearer", ExpiresAt: exp}, nil
}
func (s *Service) Authenticate(ctx context.Context, raw string) (Actor, error) {
	if !validToken(raw) {
		return Actor{}, ErrUnauthorized
	}
	sess, err := s.repo.FindSessionByTokenDigest(ctx, tokenDigest(raw))
	if err != nil || sess.RevokedAt != nil || !s.now().Before(sess.ExpiresAt) {
		return Actor{}, ErrUnauthorized
	}
	return Actor{UserID: sess.UserID, SessionID: sess.ID}, nil
}
func (s *Service) CurrentUser(ctx context.Context, id string) (User, error) {
	return s.repo.FindByID(ctx, id)
}
func (s *Service) Logout(ctx context.Context, id string) error {
	return s.repo.RevokeSession(ctx, id, s.now())
}
