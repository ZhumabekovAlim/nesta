package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Subscription struct {
	ID                 string
	UserID             string
	ComplexID          string
	PlanID             string
	Status             string
	AddressJSON        []byte
	TimeWindow         sql.NullString
	Instructions       sql.NullString
	CurrentPeriodStart sql.NullTime
	CurrentPeriodEnd   sql.NullTime
	CreatedAt          time.Time
}

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub Subscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, user_id, complex_id, plan_id, status, address_json, time_window, instructions, current_period_start, current_period_end
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, sub.ID, sub.UserID, sub.ComplexID, sub.PlanID, sub.Status, sub.AddressJSON, sub.TimeWindow, sub.Instructions, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	return err
}

func (r *SubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, complex_id, plan_id, status, address_json, time_window, instructions, current_period_start, current_period_end, created_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.ComplexID, &sub.PlanID, &sub.Status, &sub.AddressJSON, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions SET status = $2 WHERE id = $1
	`, id, status)
	return err
}

func (r *SubscriptionRepository) ListAll(ctx context.Context) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, complex_id, plan_id, status, address_json, time_window, instructions, current_period_start, current_period_end, created_at
		FROM subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.ComplexID, &sub.PlanID, &sub.Status, &sub.AddressJSON, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) Get(ctx context.Context, id string) (Subscription, error) {
	var sub Subscription
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, complex_id, plan_id, status, address_json, time_window, instructions, current_period_start, current_period_end, created_at
		FROM subscriptions
		WHERE id = $1
	`, id).Scan(&sub.ID, &sub.UserID, &sub.ComplexID, &sub.PlanID, &sub.Status, &sub.AddressJSON, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt)
	return sub, err
}
