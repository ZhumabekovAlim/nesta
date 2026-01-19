package repositories

import (
	"context"
	"database/sql"
	"strings"
)

type Product struct {
	ID          string
	Title       string
	Description sql.NullString
	PriceCents  int
	Stock       int
	CategoryID  sql.NullString
	IsActive    bool
}

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) List(ctx context.Context, category, search string, inStock bool, limit, offset int) ([]Product, error) {
	filters := []string{"is_active = TRUE"}
	args := []any{}
	idx := 1

	if category != "" {
		filters = append(filters, "category_id = $"+itoa(idx))
		args = append(args, category)
		idx++
	}
	if search != "" {
		filters = append(filters, "LOWER(title) LIKE $"+itoa(idx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		idx++
	}
	if inStock {
		filters = append(filters, "stock > 0")
	}

	query := `SELECT id, title, description, price_cents, stock, category_id, is_active FROM products WHERE ` + strings.Join(filters, " AND ") + ` ORDER BY created_at DESC LIMIT $` + itoa(idx) + ` OFFSET $` + itoa(idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Title, &product.Description, &product.PriceCents, &product.Stock, &product.CategoryID, &product.IsActive); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *ProductRepository) ListAll(ctx context.Context, limit, offset int) ([]Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, price_cents, stock, category_id, is_active
		FROM products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Title, &product.Description, &product.PriceCents, &product.Stock, &product.CategoryID, &product.IsActive); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *ProductRepository) Get(ctx context.Context, id string) (Product, error) {
	var product Product
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, price_cents, stock, category_id, is_active
		FROM products
		WHERE id = $1
	`, id).Scan(&product.ID, &product.Title, &product.Description, &product.PriceCents, &product.Stock, &product.CategoryID, &product.IsActive)
	return product, err
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id string, stock int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE products SET stock = $2 WHERE id = $1
	`, id, stock)
	return err
}

func (r *ProductRepository) Create(ctx context.Context, product Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products (id, title, description, price_cents, stock, category_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, product.ID, product.Title, product.Description, product.PriceCents, product.Stock, product.CategoryID, product.IsActive)
	return err
}

func (r *ProductRepository) Update(ctx context.Context, product Product) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE products SET title = $2, description = $3, price_cents = $4, stock = $5, category_id = $6, is_active = $7
		WHERE id = $1
	`, product.ID, product.Title, product.Description, product.PriceCents, product.Stock, product.CategoryID, product.IsActive)
	return err
}
