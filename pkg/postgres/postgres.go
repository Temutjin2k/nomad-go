package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreDB struct {
	Pool     *pgxpool.Pool
	DBConfig *pgxpool.Config
}

type Config interface {
	GetDSN() string
}

func New(ctx context.Context, config Config) (*PostgreDB, error) {
	dbConfig, err := pgxpool.ParseConfig(config.GetDSN())
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, err
	}

	// Ping the database
	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &PostgreDB{
		Pool:     pool,
		DBConfig: dbConfig,
	}, nil
}
