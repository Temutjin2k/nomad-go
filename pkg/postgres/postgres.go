package postgres

import (
	"context"
	"time"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreDB struct {
	Pool     *pgxpool.Pool
	DBConfig *pgxpool.Config
}

type Config interface {
	GetDSN() string
}

func New(ctx context.Context, config config.DatabaseConfig) (*PostgreDB, error) {
	dbConfig, err := pgxpool.ParseConfig(config.GetDSN())
	if err != nil {
		return nil, err
	}

	// Use the time.ParseDuration() function to convert the idle timeout duration string
	// to a time.Duration type.
	duration, err := time.ParseDuration(config.MaxIdleTime)
	if err != nil {
		return nil, err
	}

	dbConfig.MaxConnIdleTime = duration

	dbConfig.MaxConns = config.MaxConns
	dbConfig.MinConns = config.MinConns
	dbConfig.MaxConnLifetime = config.MaxConnLifetime
	dbConfig.MaxConnIdleTime = config.MaxConnIdleTime

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
