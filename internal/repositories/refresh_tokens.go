package repositories

import (
	"context"
	"database/sql"
	"time"
)

type RefreshToken struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	RevokedAt sql.NullTime
}

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.Token, token.ExpiresAt, token.RevokedAt)
	return err
}

func (r *RefreshTokenRepository) FindByToken(ctx context.Context, token string) (RefreshToken, error) {
	var row RefreshToken
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token = $1
	`, token).Scan(&row.ID, &row.UserID, &row.Token, &row.ExpiresAt, &row.RevokedAt)
	return row, err
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string, revokedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = $2 WHERE token = $1
	`, token, revokedAt)
	return err
}
