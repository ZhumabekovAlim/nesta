package repositories

import (
	"context"
	"database/sql"
)

type TimeWindow struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsActive  bool   `json:"is_active"`
}

type TimeWindowRepository struct {
	db *sql.DB
}

func NewTimeWindowRepository(db *sql.DB) *TimeWindowRepository {
	return &TimeWindowRepository{db: db}
}

func (r *TimeWindowRepository) ListActive(ctx context.Context) ([]TimeWindow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, label, start_time, end_time, is_active
		FROM time_windows
		WHERE is_active = TRUE
		ORDER BY start_time, label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TimeWindow, 0)
	for rows.Next() {
		var item TimeWindow
		if err := rows.Scan(&item.ID, &item.Label, &item.StartTime, &item.EndTime, &item.IsActive); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *TimeWindowRepository) Get(ctx context.Context, id string) (TimeWindow, error) {
	var item TimeWindow
	err := r.db.QueryRowContext(ctx, `
		SELECT id, label, start_time, end_time, is_active
		FROM time_windows
		WHERE id = $1
	`, id).Scan(&item.ID, &item.Label, &item.StartTime, &item.EndTime, &item.IsActive)
	return item, err
}
