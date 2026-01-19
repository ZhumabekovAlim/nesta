package storage

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres(databaseURL string) (*Postgres, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &Postgres{DB: db}, nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}
