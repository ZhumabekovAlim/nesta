package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

type ResidentialComplex struct {
	ID              string
	Name            string
	City            string
	Status          string
	Threshold       int
	CurrentRequests int
}

type ComplexRepository struct {
	db *sql.DB
}

func NewComplexRepository(db *sql.DB) *ComplexRepository {
	return &ComplexRepository{db: db}
}

func (r *ComplexRepository) List(ctx context.Context, search, status, city string, onlyActive bool, limit, offset int) ([]ResidentialComplex, error) {
	filters := []string{"1=1"}
	args := []any{}
	idx := 1

	if search != "" {
		filters = append(filters, "LOWER(name) LIKE $"+itoa(idx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		idx++
	}
	if status != "" {
		filters = append(filters, "status = $"+itoa(idx))
		args = append(args, status)
		idx++
	}
	if city != "" {
		filters = append(filters, "city = $"+itoa(idx))
		args = append(args, city)
		idx++
	}
	if onlyActive {
		filters = append(filters, "status = 'ACTIVE'")
	}

	query := `SELECT id, name, city, status, threshold_n, current_requests FROM residential_complexes WHERE ` + strings.Join(filters, " AND ") + ` ORDER BY name LIMIT $` + itoa(idx) + ` OFFSET $` + itoa(idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complexes []ResidentialComplex
	for rows.Next() {
		var item ResidentialComplex
		if err := rows.Scan(&item.ID, &item.Name, &item.City, &item.Status, &item.Threshold, &item.CurrentRequests); err != nil {
			return nil, err
		}
		complexes = append(complexes, item)
	}
	return complexes, rows.Err()
}

func (r *ComplexRepository) Get(ctx context.Context, id string) (ResidentialComplex, error) {
	var item ResidentialComplex
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, city, status, threshold_n, current_requests
		FROM residential_complexes
		WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.City, &item.Status, &item.Threshold, &item.CurrentRequests)
	return item, err
}

func (r *ComplexRepository) UpdateStatusAndRequests(ctx context.Context, id, status string, current int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE residential_complexes SET status = $2, current_requests = $3 WHERE id = $1
	`, id, status, current)
	return err
}

func (r *ComplexRepository) Create(ctx context.Context, complex ResidentialComplex) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO residential_complexes (id, name, city, status, threshold_n, current_requests)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, complex.ID, complex.Name, complex.City, complex.Status, complex.Threshold, complex.CurrentRequests)
	return err
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
