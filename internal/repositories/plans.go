package repositories

import (
	"context"
	"database/sql"
)

type Plan struct {
	ID          string
	Name        string
	PriceCents  int
	Frequency   string
	BagsPerDay  int
	Description sql.NullString
	IsActive    bool
}

type PlanRepository struct {
	db *sql.DB
}

func NewPlanRepository(db *sql.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

func (r *PlanRepository) ListActive(ctx context.Context) ([]Plan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, price_cents, frequency, bags_per_day, description, is_active
		FROM plans
		WHERE is_active = TRUE
		ORDER BY price_cents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var plan Plan
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.PriceCents, &plan.Frequency, &plan.BagsPerDay, &plan.Description, &plan.IsActive); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (r *PlanRepository) Create(ctx context.Context, plan Plan) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO plans (id, name, price_cents, frequency, bags_per_day, description, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, plan.ID, plan.Name, plan.PriceCents, plan.Frequency, plan.BagsPerDay, plan.Description, plan.IsActive)
	return err
}

func (r *PlanRepository) Get(ctx context.Context, id string) (Plan, error) {
	var plan Plan
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, price_cents, frequency, bags_per_day, description, is_active
		FROM plans
		WHERE id = $1
	`, id).Scan(&plan.ID, &plan.Name, &plan.PriceCents, &plan.Frequency, &plan.BagsPerDay, &plan.Description, &plan.IsActive)
	return plan, err
}

func (r *PlanRepository) Update(ctx context.Context, plan Plan) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE plans
		SET name = $2, price_cents = $3, frequency = $4, bags_per_day = $5, description = $6, is_active = $7
		WHERE id = $1
	`, plan.ID, plan.Name, plan.PriceCents, plan.Frequency, plan.BagsPerDay, plan.Description, plan.IsActive)
	return err
}
