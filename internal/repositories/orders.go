package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Order struct {
	ID         string
	UserID     string
	Status     string
	AddressRaw []byte
	Comment    sql.NullString
	TotalCents int
	CreatedAt  time.Time
}

type OrderItem struct {
	ID         string
	OrderID    string
	ProductID  string
	Quantity   int
	PriceCents int
}

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order Order, items []OrderItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, status, address_json, comment, total_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, order.ID, order.UserID, order.Status, order.AddressRaw, order.Comment, order.TotalCents)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, price_cents)
			VALUES ($1, $2, $3, $4, $5)
		`, item.ID, item.OrderID, item.ProductID, item.Quantity, item.PriceCents)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID string) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, status, address_json, comment, total_cents, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.AddressRaw, &order.Comment, &order.TotalCents, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) Get(ctx context.Context, id string) (Order, error) {
	var order Order
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, status, address_json, comment, total_cents, created_at
		FROM orders
		WHERE id = $1
	`, id).Scan(&order.ID, &order.UserID, &order.Status, &order.AddressRaw, &order.Comment, &order.TotalCents, &order.CreatedAt)
	return order, err
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = $2 WHERE id = $1
	`, id, status)
	return err
}

func (r *OrderRepository) Items(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, quantity, price_cents
		FROM order_items
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *OrderRepository) ListAll(ctx context.Context) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, status, address_json, comment, total_cents, created_at
		FROM orders
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.AddressRaw, &order.Comment, &order.TotalCents, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
