package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Address struct {
	ID           string
	UserID       string
	Name         string
	ComplexID    string
	CityID       string
	TimeWindowID sql.NullString
	TimeWindow   *TimeWindow
	AddressJSON  []byte
	CreatedAt    time.Time
}

type AddressRepository struct {
	db *sql.DB
}

func NewAddressRepository(db *sql.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(ctx context.Context, address Address) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO addresses (id, user_id, name, complex_id, city_id, time_window_id, address_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.CityID, address.TimeWindowID, address.AddressJSON)
	return err
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.name,
			a.complex_id,
			a.city_id,
			a.time_window_id,
			a.address_json,
			a.created_at,
			tw.id,
			tw.label,
			tw.start_time,
			tw.end_time,
			tw.is_active
		FROM addresses a
		LEFT JOIN time_windows tw ON tw.id = a.time_window_id
		WHERE user_id = $1
		ORDER BY a.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Address
	for rows.Next() {
		var address Address
		var timeWindowID, label, startTime, endTime sql.NullString
		var isActive sql.NullBool
		if err := rows.Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.CityID, &address.TimeWindowID, &address.AddressJSON, &address.CreatedAt, &timeWindowID, &label, &startTime, &endTime, &isActive); err != nil {
			return nil, err
		}
		if timeWindowID.Valid {
			address.TimeWindow = &TimeWindow{ID: timeWindowID.String, Label: label.String, StartTime: startTime.String, EndTime: endTime.String, IsActive: isActive.Bool}
		}
		items = append(items, address)
	}
	return items, rows.Err()
}

func (r *AddressRepository) Get(ctx context.Context, id string) (Address, error) {
	var address Address
	var timeWindowID, label, startTime, endTime sql.NullString
	var isActive sql.NullBool
	err := r.db.QueryRowContext(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.name,
			a.complex_id,
			a.city_id,
			a.time_window_id,
			a.address_json,
			a.created_at,
			tw.id,
			tw.label,
			tw.start_time,
			tw.end_time,
			tw.is_active
		FROM addresses a
		LEFT JOIN time_windows tw ON tw.id = a.time_window_id
		WHERE a.id = $1
	`, id).Scan(&address.ID, &address.UserID, &address.Name, &address.ComplexID, &address.CityID, &address.TimeWindowID, &address.AddressJSON, &address.CreatedAt, &timeWindowID, &label, &startTime, &endTime, &isActive)
	if err != nil {
		return address, err
	}
	if timeWindowID.Valid {
		address.TimeWindow = &TimeWindow{ID: timeWindowID.String, Label: label.String, StartTime: startTime.String, EndTime: endTime.String, IsActive: isActive.Bool}
	}
	return address, nil
}

func (r *AddressRepository) Update(ctx context.Context, address Address) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE addresses
		SET name = $3, complex_id = $4, city_id = $5, time_window_id = $6, address_json = $7
		WHERE id = $1 AND user_id = $2
	`, address.ID, address.UserID, address.Name, address.ComplexID, address.CityID, address.TimeWindowID, address.AddressJSON)
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
