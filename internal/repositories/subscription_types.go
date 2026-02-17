package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SubscriptionType struct {
	ID         string
	Title      string
	Subtitle   sql.NullString
	PriceCents int
	Features   []string
	IsActive   bool
}

type SubscriptionTypeRepository struct {
	db *sql.DB
}

func NewSubscriptionTypeRepository(db *sql.DB) *SubscriptionTypeRepository {
	return &SubscriptionTypeRepository{db: db}
}

func (r *SubscriptionTypeRepository) ListActive(ctx context.Context) ([]SubscriptionType, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, subtitle, price_cents, features, is_active
		FROM subscription_types
		WHERE is_active = TRUE
		ORDER BY price_cents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SubscriptionType
	for rows.Next() {
		var item SubscriptionType
		var featuresRaw []byte
		if err := rows.Scan(&item.ID, &item.Title, &item.Subtitle, &item.PriceCents, &featuresRaw, &item.IsActive); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(featuresRaw, &item.Features)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SubscriptionTypeRepository) Create(ctx context.Context, item SubscriptionType) error {
	featuresRaw, err := json.Marshal(item.Features)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO subscription_types (id, title, subtitle, price_cents, features, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.Title, item.Subtitle, item.PriceCents, featuresRaw, item.IsActive)
	return err
}

func (r *SubscriptionTypeRepository) Update(ctx context.Context, item SubscriptionType) error {
	featuresRaw, err := json.Marshal(item.Features)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE subscription_types
		SET title = $2, subtitle = $3, price_cents = $4, features = $5, is_active = $6
		WHERE id = $1
	`, item.ID, item.Title, item.Subtitle, item.PriceCents, featuresRaw, item.IsActive)
	return err
}

func (r *SubscriptionTypeRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM subscription_types
		WHERE id = $1
	`, id)
	return err
}
