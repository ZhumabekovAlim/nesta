package api

import (
	"database/sql"
	"testing"

	"nesta/internal/repositories"
)

func TestNormalizeSubscriptionType(t *testing.T) {
	t.Run("with subtitle", func(t *testing.T) {
		item := repositories.SubscriptionType{
			ID:         "sub_1",
			Title:      "Monthly",
			Subtitle:   sql.NullString{String: "Popular", Valid: true},
			PriceCents: 129900,
			Features:   []string{"a", "b"},
			IsActive:   true,
		}

		normalized := normalizeSubscriptionType(item)
		if normalized.Subtitle == nil || *normalized.Subtitle != "Popular" {
			t.Fatalf("subtitle must be normalized to pointer with value, got %#v", normalized.Subtitle)
		}
		if normalized.ID != item.ID || normalized.Title != item.Title || normalized.PriceCents != item.PriceCents || normalized.IsActive != item.IsActive {
			t.Fatalf("primitive fields must match source struct")
		}
	})

	t.Run("without subtitle", func(t *testing.T) {
		item := repositories.SubscriptionType{ID: "sub_2", Title: "Weekly"}

		normalized := normalizeSubscriptionType(item)
		if normalized.Subtitle != nil {
			t.Fatalf("subtitle must be nil when source subtitle is invalid")
		}
	})
}
