package repo

import (
	"context"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	db *pgxpool.Pool
}

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{
		db: db,
	}
}

func (r *SessionRepo) Create(ctx context.Context, driverID uuid.UUID) (sessiondID uuid.UUID, err error) {
	const op = "SessionRepo.Create"
	query := `
		INSERT INTO driver_sessions(driver_id)
		VALUES($1)
		RETURNING id;`

	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, driverID).Scan(&sessiondID); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return uuid.UUID{}, wrap.Error(ctx, fmt.Errorf("%s: %v", op, err))
	}

	return sessiondID, nil
}
