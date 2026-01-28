package postgres

import (
	"context"
	"fmt"
	"time"

	"ride-hail-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, c config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
	c.DB.User, c.DB.Pass, c.DB.Host, c.DB.Port, c.DB.Name,
)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return pool, pool.Ping(ctx)
}
