package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RideRepo struct {
	db *pgxpool.Pool
}

func NewRideRepo(db *pgxpool.Pool) *RideRepo {
	return &RideRepo{
		db: db,
	}
}

// StartRide marks a ride as started by updating its status and recording the start time.
// It also logs the ride event in the ride_events table.
func (r *RideRepo) StartRide(ctx context.Context, rideID, driverID uuid.UUID, startedAt time.Time, rideEvent models.RideEvent) error {
	const op = "RideRepo.StartRide"

	// Update the ride status to 'IN_PROGRESS' and set the started_at timestamp
	updateQuery := `
        UPDATE rides
        SET status = 'IN_PROGRESS', updated_at = now(), started_at = $2
        WHERE id = $1'`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, updateQuery, rideID, startedAt); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return wrap.Error(ctx, fmt.Errorf("%s: failed to update ride status: %w", op, err))
	}

	// Insert a new ride event into the ride_events table
	insertEventQuery := `
        INSERT INTO ride_events(ride_id, event_type, event_data)
        VALUES($1, 'RIDE_STARTED', $2)`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, insertEventQuery, rideID, rideEvent); err != nil {
		if postgres.IsForeignKeyViolation(err) {
			return types.ErrRideNotFound
		}

		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return wrap.Error(ctx, fmt.Errorf("%s: failed to insert ride event: %w", op, err))
	}

	return nil
}
