package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Address struct {
	ID          string
	UserID      string
	Name        string
	ComplexID   string
	CityID      string
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
		INSERT INTO addresses (id, user_id, name, complex_id, city_id, address_json)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.CityID, address.AddressJSON)
	return err
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, complex_id, city_id, address_json, created_at
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
		if err := rows.Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.CityID, &address.AddressJSON, &address.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, address)
	}
	return items, rows.Err()
}

func (r *AddressRepository) Get(ctx context.Context, id string) (Address, error) {
	var address Address
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, complex_id, city_id, address_json, created_at
		FROM addresses
		WHERE id = $1
	`, id).Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.CityID, &address.AddressJSON, &address.CreatedAt)
	return address, err
}

func (r *AddressRepository) Update(ctx context.Context, address Address) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE addresses
		SET name = $3, complex_id = $4, city_id = $5, address_json = $6
		WHERE id = $1 AND user_id = $2
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.CityID, address.AddressJSON)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type AddressSuggestion struct {
	ID          string
	Name        string
	Address     string
	ComplexID   string
	ComplexName string
	ComplexAddr string
	CityID      string
	CityName    string
}

func (r *AddressRepository) Search(ctx context.Context, cityID, query string, limit int) ([]AddressSuggestion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name,
			c.address,
			c.id,
			c.name,
			c.address,
			c.city_id,
			cities.name
		FROM residential_complexes c
		JOIN cities ON cities.id = c.city_id
		WHERE ($1 = '' OR c.city_id = $1)
			AND (
				LOWER(COALESCE(c.name, '')) LIKE $2
				OR LOWER(COALESCE(c.address, '')) LIKE $2
			)
		ORDER BY c.name
		LIMIT $3
	`, cityID, "%"+strings.ToLower(query)+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AddressSuggestion
	for rows.Next() {
		var item AddressSuggestion
		if err := rows.Scan(&item.ID, &item.Name, &item.Address, &item.ComplexID, &item.ComplexName, &item.ComplexAddr, &item.CityID, &item.CityName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
