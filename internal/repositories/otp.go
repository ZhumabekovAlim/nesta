package repositories

import (
	"context"
	"database/sql"
	"time"
)

type OTPCode struct {
	ID           string
	Phone        string
	CodeHash     string
	ExpiresAt    time.Time
	Attempts     int
	BlockedUntil sql.NullTime
	CreatedAt    time.Time
}

type OTPRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

func (r *OTPRepository) Create(ctx context.Context, code OTPCode) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO otp_codes (id, phone, code_hash, expires_at, attempts, blocked_until)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, code.ID, code.Phone, code.CodeHash, code.ExpiresAt, code.Attempts, code.BlockedUntil)
	return err
}

func (r *OTPRepository) LatestByPhone(ctx context.Context, phone string) (OTPCode, error) {
	var code OTPCode
	err := r.db.QueryRowContext(ctx, `
		SELECT id, phone, code_hash, expires_at, attempts, blocked_until, created_at
		FROM otp_codes
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, phone).Scan(&code.ID, &code.Phone, &code.CodeHash, &code.ExpiresAt, &code.Attempts, &code.BlockedUntil, &code.CreatedAt)
	return code, err
}

func (r *OTPRepository) IncrementAttempts(ctx context.Context, id string, attempts int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE otp_codes SET attempts = $2 WHERE id = $1
	`, id, attempts)
	return err
}

func (r *OTPRepository) Block(ctx context.Context, id string, until time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE otp_codes SET blocked_until = $2 WHERE id = $1
	`, id, until)
	return err
}
