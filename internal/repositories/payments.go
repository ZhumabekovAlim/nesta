package repositories

import (
	"context"
	"database/sql"
)

type Payment struct {
	ID              string
	Type            string
	EntityID        string
	Provider        string
	ProviderPayment sql.NullString
	Status          string
	AmountCents     int
	PayloadRaw      []byte
}

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment Payment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments (id, type, entity_id, provider, provider_payment_id, status, amount_cents, payload_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, payment.ID, payment.Type, payment.EntityID, payment.Provider, payment.ProviderPayment, payment.Status, payment.AmountCents, payment.PayloadRaw)
	return err
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id, status string, payload []byte) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET status = $2, payload_json = $3 WHERE id = $1
	`, id, status, payload)
	return err
}

func (r *PaymentRepository) FindByProviderID(ctx context.Context, provider, providerPaymentID string) (Payment, error) {
	var payment Payment
	err := r.db.QueryRowContext(ctx, `
		SELECT id, type, entity_id, provider, provider_payment_id, status, amount_cents, payload_json
		FROM payments
		WHERE provider = $1 AND provider_payment_id = $2
	`, provider, providerPaymentID).Scan(&payment.ID, &payment.Type, &payment.EntityID, &payment.Provider, &payment.ProviderPayment, &payment.Status, &payment.AmountCents, &payment.PayloadRaw)
	return payment, err
}
