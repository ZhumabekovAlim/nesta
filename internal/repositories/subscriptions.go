package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Subscription struct {
	ID                 string
	UserID             string
	AddressID          string
	PlanID             string
	Status             string
	AddressName        string
	AddressJSON        []byte
	ComplexID          string
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
	err := r.createAddressBased(ctx, sub)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code == "23502" && requiresLegacyColumns(pgErr.ColumnName) {
		return r.createLegacy(ctx, sub)
	}

	if pgErr.Code == "23503" && pgErr.ConstraintName == "subscriptions_plan_id_fkey" {
		if ensureErr := r.ensureCompatPlanFromSubscriptionType(ctx, sub.PlanID); ensureErr != nil {
			return ensureErr
		}

		retryErr := r.createAddressBased(ctx, sub)
		if retryErr == nil {
			return nil
		}
		if errors.As(retryErr, &pgErr) && pgErr.Code == "23502" && requiresLegacyColumns(pgErr.ColumnName) {
			return r.createLegacy(ctx, sub)
		}
		return retryErr
	}

	return err
}

func (r *SubscriptionRepository) createAddressBased(ctx context.Context, sub Subscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, user_id, address_id, plan_id, status, time_window, instructions, current_period_start, current_period_end
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sub.ID, sub.UserID, sub.AddressID, sub.PlanID, sub.Status, sub.TimeWindow, sub.Instructions, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	legacySchemaRequired := strings.Contains(errMsg, `null value in column "complex_id"`) ||
		strings.Contains(errMsg, `null value in column "address_json"`) ||
		strings.Contains(errMsg, `null value in column "address_name"`)
	if !legacySchemaRequired {
		return err
	}

	_, legacyErr := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, user_id, address_id, plan_id, status, address_name, address_json, complex_id, time_window, instructions, current_period_start, current_period_end
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, sub.ID, sub.UserID, sub.AddressID, sub.PlanID, sub.Status, sub.AddressName, sub.AddressJSON, sub.ComplexID, sub.TimeWindow, sub.Instructions, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if legacyErr != nil {
		return legacyErr
	}

	return nil
}

func (r *SubscriptionRepository) createLegacy(ctx context.Context, sub Subscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, user_id, address_id, plan_id, status, address_name, address_json, complex_id, time_window, instructions, current_period_start, current_period_end
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, sub.ID, sub.UserID, sub.AddressID, sub.PlanID, sub.Status, sub.AddressName, sub.AddressJSON, sub.ComplexID, sub.TimeWindow, sub.Instructions, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	return err
}

func (r *SubscriptionRepository) ensureCompatPlanFromSubscriptionType(ctx context.Context, subscriptionTypeID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO plans (id, name, price_cents, frequency, bags_per_day, description, is_active)
		SELECT
			st.id,
			st.title,
			st.price_cents,
			'MONTHLY',
			0,
			st.subtitle,
			st.is_active
		FROM subscription_types st
		WHERE st.id = $1
		ON CONFLICT (id) DO NOTHING
	`, subscriptionTypeID)
	return err
}

func requiresLegacyColumns(columnName string) bool {
	switch columnName {
	case "complex_id", "address_json", "address_name":
		return true
	default:
		return false
	}
}

func (r *SubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.user_id,
			s.address_id,
			s.plan_id,
			s.status,
			a.name,
			a.address_json,
			a.complex_id,
			s.time_window,
			s.instructions,
			s.current_period_start,
			s.current_period_end,
			s.created_at
		FROM subscriptions s
		JOIN addresses a ON a.id = s.address_id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.AddressID, &sub.PlanID, &sub.Status, &sub.AddressName, &sub.AddressJSON, &sub.ComplexID, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) ListByComplex(ctx context.Context, complexID string) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.user_id,
			s.address_id,
			s.plan_id,
			s.status,
			a.name,
			a.address_json,
			a.complex_id,
			s.time_window,
			s.instructions,
			s.current_period_start,
			s.current_period_end,
			s.created_at
		FROM subscriptions s
		JOIN addresses a ON a.id = s.address_id
		WHERE a.complex_id = $1
		ORDER BY s.created_at DESC
	`, complexID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.AddressID, &sub.PlanID, &sub.Status, &sub.AddressName, &sub.AddressJSON, &sub.ComplexID, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt); err != nil {
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
		SELECT
			s.id,
			s.user_id,
			s.address_id,
			s.plan_id,
			s.status,
			a.name,
			a.address_json,
			a.complex_id,
			s.time_window,
			s.instructions,
			s.current_period_start,
			s.current_period_end,
			s.created_at
		FROM subscriptions s
		JOIN addresses a ON a.id = s.address_id
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.AddressID, &sub.PlanID, &sub.Status, &sub.AddressName, &sub.AddressJSON, &sub.ComplexID, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) Get(ctx context.Context, id string) (Subscription, error) {
	var sub Subscription
	err := r.db.QueryRowContext(ctx, `
		SELECT
			s.id,
			s.user_id,
			s.address_id,
			s.plan_id,
			s.status,
			a.name,
			a.address_json,
			a.complex_id,
			s.time_window,
			s.instructions,
			s.current_period_start,
			s.current_period_end,
			s.created_at
		FROM subscriptions s
		JOIN addresses a ON a.id = s.address_id
		WHERE s.id = $1
	`, id).Scan(&sub.ID, &sub.UserID, &sub.AddressID, &sub.PlanID, &sub.Status, &sub.AddressName, &sub.AddressJSON, &sub.ComplexID, &sub.TimeWindow, &sub.Instructions, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt)
	return sub, err
}
