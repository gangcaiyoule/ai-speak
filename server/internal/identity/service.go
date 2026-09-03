package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Service 实现注册、登录、当前用户查询和退出登录用例。
type Service struct {
	users    UserRepository
	sessions SessionRepository
}

// NewService 创建使用指定仓库的认证服务。
func NewService(users UserRepository, sessions SessionRepository) (*Service, error) {
	if users == nil || sessions == nil {
		return nil, errors.New("identity: repository is required")
	}
	return &Service{users: users, sessions: sessions}, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (Session, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || !validPassword(input.Password) {
		return Session{}, ErrInvalidRequest
	}
	hash, err := hashPassword(input.Password)
	if err != nil {
		return Session{}, err
	}
	user := User{ID: newID("usr"), Email: email}
	if _, err = s.users.CreateWithPasswordHash(ctx, StoredUser{User: user, PasswordHash: hash}); err != nil {
		return Session{}, err
	}
	return s.createSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (Session, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || input.Password == "" {
		return Session{}, ErrInvalidRequest
	}
	stored, err := s.users.FindByEmailWithPasswordHash(ctx, email)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	ok, err := verifyPassword(input.Password, stored.PasswordHash)
	if err != nil || !ok {
		return Session{}, ErrInvalidCredentials
	}
	return s.createSession(ctx, stored.User)
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return ErrUnauthorized
	}
	session, err := s.sessions.FindByTokenDigest(ctx, sessionID)
	if err != nil {
		return ErrUnauthorized
	}
	return s.sessions.Delete(ctx, session.ID)
}

func (s *Service) CurrentUser(ctx context.Context, sessionID string) (User, error) {
	if sessionID == "" {
		return User{}, ErrUnauthorized
	}
	session, err := s.sessions.FindByTokenDigest(ctx, sessionID)
	if err != nil || session.User.ID == "" {
		return User{}, ErrUnauthorized
	}
	return s.users.FindByID(ctx, session.User.ID)
}

func (s *Service) createSession(ctx context.Context, user User) (Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Session{}, err
	}
	token := "sess_" + base64.RawURLEncoding.EncodeToString(tokenBytes)
	session := Session{ID: newID("ses"), User: user, Token: token}
	return s.sessions.Create(ctx, session)
}

func newID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(b)
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !utf8.ValidString(value) || len(value) < 3 || !strings.Contains(value, "@") {
		return "", ErrInvalidRequest
	}
	return value, nil
}
func validPassword(value string) bool {
	n := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && n >= 8 && n <= 128
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return "$argon2id$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}
func verifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[1] != "argon2id" {
		return false, ErrInvalidRequest
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
