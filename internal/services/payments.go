package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"nesta/internal/repositories"
)

type PaymentService struct {
	DB            *sql.DB
	Payments      *repositories.PaymentRepository
	Orders        *repositories.OrderRepository
	Subscriptions *repositories.SubscriptionRepository
}

type PaymentInitRequest struct {
	Type              string
	EntityID          string
	Provider          string
	ProviderPaymentID string
	AmountCents       int
}

type PaymentWebhook struct {
	Provider          string
	ProviderPaymentID string
	Status            string
	Payload           any
}

func (s *PaymentService) Init(ctx context.Context, req PaymentInitRequest) (repositories.Payment, error) {
	id, err := NewID()
	if err != nil {
		return repositories.Payment{}, err
	}
	payment := repositories.Payment{
		ID:          id,
		Type:        req.Type,
		EntityID:    req.EntityID,
		Provider:    req.Provider,
		Status:      "INIT",
		AmountCents: req.AmountCents,
	}
	if req.ProviderPaymentID != "" {
		payment.ProviderPayment = sql.NullString{String: req.ProviderPaymentID, Valid: true}
	}
	if err := s.Payments.Create(ctx, payment); err != nil {
		return repositories.Payment{}, err
	}
	return payment, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, webhook PaymentWebhook) error {
	payload, _ := json.Marshal(webhook.Payload)

	existing, err := s.Payments.FindByProviderID(ctx, webhook.Provider, webhook.ProviderPaymentID)
	if err == nil {
		if existing.Status == webhook.Status {
			return nil
		}
		return s.updatePaymentAndEntity(ctx, existing, webhook.Status, payload)
	}

	return errors.New("payment not found")
}

func (s *PaymentService) updatePaymentAndEntity(ctx context.Context, payment repositories.Payment, status string, payload []byte) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `UPDATE payments SET status = $2, payload_json = $3 WHERE id = $1`, payment.ID, status, payload)
	if err != nil {
		return err
	}

	if payment.Type == "order" && status == "PAID" {
		_, err = tx.ExecContext(ctx, `UPDATE orders SET status = 'PAID' WHERE id = $1`, payment.EntityID)
		if err != nil {
			return err
		}

		items, err := s.Orders.Items(ctx, payment.EntityID)
		if err != nil {
			return err
		}
		for _, item := range items {
			var stock int
			if err := tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE id = $1 FOR UPDATE`, item.ProductID).Scan(&stock); err != nil {
				return err
			}
			newStock := stock - item.Quantity
			if newStock < 0 {
				return errors.New("insufficient stock")
			}
			_, err = tx.ExecContext(ctx, `UPDATE products SET stock = $2 WHERE id = $1`, item.ProductID, newStock)
			if err != nil {
				return err
			}
		}
	}

	if payment.Type == "subscription" && status == "PAID" {
		_, err = tx.ExecContext(ctx, `UPDATE subscriptions SET status = 'ACTIVE', current_period_start = $2, current_period_end = $3 WHERE id = $1`, payment.EntityID, time.Now(), time.Now().Add(30*24*time.Hour))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
