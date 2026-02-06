package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Address struct {
	ID          string
	UserID      string
	Name        string
	ComplexID   string
	AddressJSON []byte
	CreatedAt   time.Time
}

type AddressRepository struct {
	db *sql.DB
}

func NewAddressRepository(db *sql.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(ctx context.Context, address Address) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO addresses (id, user_id, name, complex_id, address_json)
		VALUES ($1, $2, $3, $4, $5)
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.AddressJSON)
	return err
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, complex_id, address_json, created_at
		FROM addresses
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Address
	for rows.Next() {
		var address Address
		if err := rows.Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.AddressJSON, &address.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, address)
	}
	return items, rows.Err()
}

func (r *AddressRepository) Get(ctx context.Context, id string) (Address, error) {
	var address Address
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, complex_id, address_json, created_at
		FROM addresses
		WHERE id = $1
	`, id).Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.AddressJSON, &address.CreatedAt)
	return address, err
}

func (r *AddressRepository) Update(ctx context.Context, address Address) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE addresses
		SET name = $3, complex_id = $4, address_json = $5
		WHERE id = $1 AND user_id = $2
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.AddressJSON)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *AddressRepository) Delete(ctx context.Context, id, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM addresses
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
