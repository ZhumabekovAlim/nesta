package repositories

import (
	"context"
	"database/sql"
	"time"
)

type PickupLog struct {
	ID             string
	SubscriptionID string
	PickupDate     time.Time
	Status         string
	Comment        sql.NullString
	Reason         sql.NullString
}

type PickupLogRepository struct {
	db *sql.DB
}

func NewPickupLogRepository(db *sql.DB) *PickupLogRepository {
	return &PickupLogRepository{db: db}
}

func (r *PickupLogRepository) ListBySubscription(ctx context.Context, subscriptionID string) ([]PickupLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subscription_id, pickup_date, status, comment, reason
		FROM pickup_logs
		WHERE subscription_id = $1
		ORDER BY pickup_date DESC
	`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []PickupLog
	for rows.Next() {
		var log PickupLog
		if err := rows.Scan(&log.ID, &log.SubscriptionID, &log.PickupDate, &log.Status, &log.Comment, &log.Reason); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *PickupLogRepository) Create(ctx context.Context, log PickupLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pickup_logs (id, subscription_id, pickup_date, status, comment, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.ID, log.SubscriptionID, log.PickupDate, log.Status, log.Comment, log.Reason)
	return err
}

func (r *PickupLogRepository) Update(ctx context.Context, log PickupLog) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE pickup_logs SET status = $2, comment = $3, reason = $4 WHERE id = $1
	`, log.ID, log.Status, log.Comment, log.Reason)
	return err
}
