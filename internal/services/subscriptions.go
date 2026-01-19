package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"nesta/internal/repositories"
)

type SubscriptionService struct {
	Subscriptions *repositories.SubscriptionRepository
	Complexes     *repositories.ComplexRepository
	Plans         *repositories.PlanRepository
}

type SubscriptionCreateResult struct {
	Subscription    repositories.Subscription
	RequiresPayment bool
}

func (s *SubscriptionService) Create(ctx context.Context, userID, complexID, planID string, address []byte, timeWindow, instructions string) (SubscriptionCreateResult, error) {
	complex, err := s.Complexes.Get(ctx, complexID)
	if err != nil {
		return SubscriptionCreateResult{}, err
	}
	if complex.Status != "ACTIVE" {
		return SubscriptionCreateResult{}, errors.New("complex is not active")
	}

	plan, err := s.Plans.Get(ctx, planID)
	if err != nil {
		return SubscriptionCreateResult{}, err
	}
	if !plan.IsActive {
		return SubscriptionCreateResult{}, errors.New("plan not active")
	}

	id, err := NewID()
	if err != nil {
		return SubscriptionCreateResult{}, err
	}

	status := "ACTIVE"
	requiresPayment := plan.PriceCents > 0
	if requiresPayment {
		status = "PAYMENT_PENDING"
	}

	subscription := repositories.Subscription{
		ID:          id,
		UserID:      userID,
		ComplexID:   complexID,
		PlanID:      planID,
		Status:      status,
		AddressJSON: address,
	}

	if timeWindow != "" {
		subscription.TimeWindow = sql.NullString{String: timeWindow, Valid: true}
	}
	if instructions != "" {
		subscription.Instructions = sql.NullString{String: instructions, Valid: true}
	}

	if err := s.Subscriptions.Create(ctx, subscription); err != nil {
		return SubscriptionCreateResult{}, err
	}

	subscription.CurrentPeriodStart = sql.NullTime{Time: time.Now(), Valid: true}
	subscription.CurrentPeriodEnd = sql.NullTime{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true}
	return SubscriptionCreateResult{Subscription: subscription, RequiresPayment: requiresPayment}, nil
}

func (s *SubscriptionService) UpdateStatus(ctx context.Context, id, status string) error {
	return s.Subscriptions.UpdateStatus(ctx, id, status)
}
