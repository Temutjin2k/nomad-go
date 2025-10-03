package repo

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/pkg/trm"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

func TxorDB(ctx context.Context, db *pgxpool.Pool) Querier {
	tx, ok := ctx.Value(trm.TxKey).(pgx.Tx)
	if !ok {
		return db
	}
	return tx
}
