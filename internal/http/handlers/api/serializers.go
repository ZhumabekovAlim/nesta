package api

import "nesta/internal/repositories"

type subscriptionTypeResponse struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Subtitle   *string  `json:"subtitle"`
	PriceCents int      `json:"price_cents"`
	Features   []string `json:"features,omitempty"`
	IsActive   bool     `json:"is_active"`
}

func normalizeSubscriptionType(item repositories.SubscriptionType) subscriptionTypeResponse {
	resp := subscriptionTypeResponse{
		ID:         item.ID,
		Title:      item.Title,
		PriceCents: item.PriceCents,
		Features:   item.Features,
		IsActive:   item.IsActive,
	}

	if item.Subtitle.Valid {
		value := item.Subtitle.String
		resp.Subtitle = &value
	}

	return resp
}

func normalizeSubscriptionTypes(items []repositories.SubscriptionType) []subscriptionTypeResponse {
	result := make([]subscriptionTypeResponse, 0, len(items))
	for _, item := range items {
		result = append(result, normalizeSubscriptionType(item))
	}
	return result
}

