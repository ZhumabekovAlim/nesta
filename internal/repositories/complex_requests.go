package repositories

import (
	"context"
	"database/sql"
	"time"
)

type ComplexRequest struct {
	ID         string
	ComplexID  string
	Phone      string
	Verified   bool
	CreatedAt  time.Time
	VerifiedAt sql.NullTime
}

type ComplexRequestRepository struct {
	db *sql.DB
}

func NewComplexRequestRepository(db *sql.DB) *ComplexRequestRepository {
	return &ComplexRequestRepository{db: db}
}

func (r *ComplexRequestRepository) Create(ctx context.Context, req ComplexRequest) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO complex_requests (id, complex_id, phone, verified, verified_at)
		VALUES ($1, $2, $3, $4, $5)
	`, req.ID, req.ComplexID, req.Phone, req.Verified, req.VerifiedAt)
	return err
}

func (r *ComplexRequestRepository) FindByComplexAndPhone(ctx context.Context, complexID, phone string) (ComplexRequest, error) {
	var req ComplexRequest
	err := r.db.QueryRowContext(ctx, `
		SELECT id, complex_id, phone, verified, created_at, verified_at
		FROM complex_requests
		WHERE complex_id = $1 AND phone = $2
	`, complexID, phone).Scan(&req.ID, &req.ComplexID, &req.Phone, &req.Verified, &req.CreatedAt, &req.VerifiedAt)
	return req, err
}

func (r *ComplexRequestRepository) Verify(ctx context.Context, id string, verifiedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE complex_requests
		SET verified = TRUE, verified_at = $2
		WHERE id = $1
	`, id, verifiedAt)
	return err
}
