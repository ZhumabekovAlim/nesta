package services

import (
	"context"
	"errors"

	"nesta/internal/repositories"
)

type OrderService struct {
	Orders   *repositories.OrderRepository
	Products *repositories.ProductRepository
}

type OrderItemInput struct {
	ProductID string
	Quantity  int
}

func (s *OrderService) Create(ctx context.Context, userID string, address []byte, comment string, items []OrderItemInput) (repositories.Order, []repositories.OrderItem, error) {
	if len(items) == 0 {
		return repositories.Order{}, nil, errors.New("items required")
	}

	orderID, err := NewID()
	if err != nil {
		return repositories.Order{}, nil, err
	}

	var total int
	var orderItems []repositories.OrderItem
	for _, item := range items {
		if item.Quantity <= 0 {
			return repositories.Order{}, nil, errors.New("invalid quantity")
		}
		product, err := s.Products.Get(ctx, item.ProductID)
		if err != nil {
			return repositories.Order{}, nil, err
		}
		lineTotal := product.PriceCents * item.Quantity
		total += lineTotal

		itemID, err := NewID()
		if err != nil {
			return repositories.Order{}, nil, err
		}
		orderItems = append(orderItems, repositories.OrderItem{
			ID:         itemID,
			OrderID:    orderID,
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		})
	}

	order := repositories.Order{
		ID:         orderID,
		UserID:     userID,
		Status:     "NEW",
		AddressRaw: address,
		TotalCents: total,
	}
	if comment != "" {
		order.Comment.Valid = true
		order.Comment.String = comment
	}

	if err := s.Orders.Create(ctx, order, orderItems); err != nil {
		return repositories.Order{}, nil, err
	}

	return order, orderItems, nil
}
