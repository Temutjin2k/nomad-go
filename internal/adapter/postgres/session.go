package postgres

import (
	"context"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5"
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
		return uuid.UUID{}, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return sessiondID, nil
}

func (r *SessionRepo) GetSummary(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error) {
	const op = "SessionRepo.GetSummary"
	query := `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE ended_at IS NULL AND driver_id = $1
		RETURNING id, total_rides, total_earnings, EXTRACT(EPOCH FROM (now() - started_at)) / 3600.0 AS hours`

	var summary models.SessionSummary
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, driverID).Scan(&summary.SessionID, &summary.RidesCompleted, &summary.Earnings, &summary.DurationHours); err != nil {
		if err == pgx.ErrNoRows {
			return models.SessionSummary{}, types.ErrSessionNotFound
		}

		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return models.SessionSummary{}, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return summary, nil
}
