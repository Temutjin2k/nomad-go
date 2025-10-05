package repo

import (
	"context"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CoordinateRepo struct {
	db *pgxpool.Pool
}

func NewCoordinateRepo(db *pgxpool.Pool) *CoordinateRepo {
	return &CoordinateRepo{
		db: db,
	}
}

func (r *CoordinateRepo) Create(ctx context.Context, entityID uuid.UUID, entityType types.EntityType, address string, latitude, longitude float64) (uuid.UUID, error) {
	const op = "CoordinateRepo.Create"
	query := `
		INSERT INTO coordinates(entity_id,entity_type, address, latitude, longitude)
		VALUES($1, $2, $3, $4, $5)
		RETURNING ID;`

	var id uuid.UUID
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, entityID, entityType, address, latitude, longitude).Scan(&id); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return uuid.UUID{}, wrap.Error(ctx, fmt.Errorf("%s: %v", op, err))
	}

	return id, nil
}
