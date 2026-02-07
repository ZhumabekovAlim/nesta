package repositories

import (
	"context"
	"database/sql"
	"time"
)

type City struct {
	ID        string
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

type CityRepository struct {
	db *sql.DB
}

func NewCityRepository(db *sql.DB) *CityRepository {
	return &CityRepository{db: db}
}

func (r *CityRepository) ListActive(ctx context.Context) ([]City, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, is_active, created_at
		FROM cities
		WHERE is_active = TRUE
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []City
	for rows.Next() {
		var city City
		if err := rows.Scan(&city.ID, &city.Name, &city.IsActive, &city.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, city)
	}
	return items, rows.Err()
}

func (r *CityRepository) Get(ctx context.Context, id string) (City, error) {
	var city City
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, is_active, created_at
		FROM cities
		WHERE id = $1
	`, id).Scan(&city.ID, &city.Name, &city.IsActive, &city.CreatedAt)
	return city, err
}
