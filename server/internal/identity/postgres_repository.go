package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// PostgresRepository persists identity data without ever receiving raw tokens.
type PostgresRepository struct{ db *sql.DB }

func NewPostgresRepository(db *sql.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) CreateWithPasswordHash(ctx context.Context, stored StoredUser) (User, error) {
	if r == nil || r.db == nil || stored.User.ID == "" || stored.User.Email == "" || stored.PasswordHash == "" {
		return User{}, ErrInvalidRequest
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin identity registration: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO users (id, canonical_email, created_at, updated_at) VALUES ($1,$2,$3,$3)`, stored.User.ID, stored.User.Email, now)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO credentials (user_id, password_hash, updated_at) VALUES ($1,$2,$3)`, stored.User.ID, stored.PasswordHash, now); err != nil {
		return User{}, fmt.Errorf("insert credential: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit identity registration: %w", err)
	}
	return stored.User, nil
}

func (r *PostgresRepository) FindByEmailWithPasswordHash(ctx context.Context, email string) (StoredUser, error) {
	if r == nil || r.db == nil {
		return StoredUser{}, ErrInvalidCredentials
	}
	var out StoredUser
	err := r.db.QueryRowContext(ctx, `SELECT u.id,u.canonical_email,c.password_hash FROM users u JOIN credentials c ON c.user_id=u.id WHERE u.canonical_email=$1 AND u.status='active'`, email).Scan(&out.User.ID, &out.User.Email, &out.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return StoredUser{}, ErrInvalidCredentials
	}
	if err != nil {
		return StoredUser{}, fmt.Errorf("find credential: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (User, error) {
	if r == nil || r.db == nil {
		return User{}, ErrNotFound
	}
	var out User
	err := r.db.QueryRowContext(ctx, `SELECT id,canonical_email FROM users WHERE id=$1 AND status='active'`, id).Scan(&out.ID, &out.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session AuthSession) error {
	if r == nil || r.db == nil {
		return ErrUnauthorized
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO auth_sessions (id,user_id,token_digest,created_at,expires_at) VALUES ($1,$2,$3,$4,$5)`, session.ID, session.UserID, session.TokenDigest, session.CreatedAt, session.ExpiresAt)
	if isUniqueViolation(err) {
		return ErrUnauthorized
	}
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindSessionByTokenDigest(ctx context.Context, digest string) (AuthSession, error) {
	if r == nil || r.db == nil {
		return AuthSession{}, ErrUnauthorized
	}
	var out AuthSession
	var revoked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,user_id,token_digest,created_at,expires_at,revoked_at FROM auth_sessions WHERE token_digest=$1`, digest).Scan(&out.ID, &out.UserID, &out.TokenDigest, &out.CreatedAt, &out.ExpiresAt, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthSession{}, ErrUnauthorized
	}
	if err != nil {
		return AuthSession{}, fmt.Errorf("find session: %w", err)
	}
	if revoked.Valid {
		out.RevokedAt = &revoked.Time
	}
	return out, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, id string, now time.Time) error {
	if r == nil || r.db == nil {
		return ErrUnauthorized
	}
	_, err := r.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,$2) WHERE id=$1`, id, now)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var sqlErr *pq.Error
	return errors.As(err, &sqlErr) && string(sqlErr.Code) == "23505"
}
