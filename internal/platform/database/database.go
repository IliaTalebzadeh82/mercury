package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxConnections int32 = 4

var (
	ErrInvalidConfiguration = errors.New("invalid database configuration")
	ErrPoolCreation         = errors.New("database pool creation failed")
	ErrConnection           = errors.New("database connection failed")
)

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	poolConfig.MaxConns = maxConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, ErrPoolCreation
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrConnection
	}

	return pool, nil
}
