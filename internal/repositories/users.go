package repositories

import (
	"context"
	"database/sql"
)

type User struct {
	ID                string
	Phone             string
	Name              sql.NullString
	Email             sql.NullString
	Role              string
	DefaultAddressRaw []byte
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, phone, name, email, role, default_address_json)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, user.ID, user.Phone, user.Name, user.Email, user.Role, user.DefaultAddressRaw)
	return err
}

func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, phone, name, email, role, default_address_json
		FROM users
		WHERE phone = $1
	`, phone).Scan(&user.ID, &user.Phone, &user.Name, &user.Email, &user.Role, &user.DefaultAddressRaw)
	return user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, phone, name, email, role, default_address_json
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Phone, &user.Name, &user.Email, &user.Role, &user.DefaultAddressRaw)
	return user, err
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id string, name, email sql.NullString, defaultAddress []byte) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET name = $2, email = $3, default_address_json = $4
		WHERE id = $1
	`, id, name, email, defaultAddress)
	return err
}
