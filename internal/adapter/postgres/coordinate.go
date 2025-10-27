package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *CoordinateRepo) CreateCoordinate(ctx context.Context, entityID uuid.UUID, entityType types.EntityType, location models.Location, updatedAt time.Time) (uuid.UUID, error) {
	const op = "CoordinateRepo.CreateCoordinate"
	query := `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`

	var id uuid.UUID
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, entityID, entityType, location.Address, location.Latitude, location.Longitude, updatedAt).Scan(&id); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return uuid.UUID{}, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return id, nil
}

func (r *CoordinateRepo) CreateLocationHistory(ctx context.Context, coordinateID, driverID uuid.UUID, rideID *uuid.UUID, location models.Location, accuracyMeters, speedKmh, headingDegrees float64) (uuid.UUID, error) {
	const op = "CoordinateRepo.CreateLocationHistory"
	query := `
		INSERT INTO location_history(coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, ride_id)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ID;`
	var id uuid.UUID
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, coordinateID, driverID, location.Latitude, location.Longitude, accuracyMeters, speedKmh, headingDegrees, rideID).Scan(&id); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return uuid.UUID{}, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}
	return id, nil
}

func (r *CoordinateRepo) GetDriverLastCoordinate(ctx context.Context, driverID uuid.UUID) (models.Location, error) {
	const op = "CoordinateRepo.GetDriverLastCoordinate"
	query := `
		SELECT latitude, longitude
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'driver'
		ORDER BY created_at DESC
		LIMIT 1;`

	var location models.Location
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, driverID).Scan(&location.Latitude, &location.Longitude); err != nil {
		if err == pgx.ErrNoRows {
			return models.Location{}, types.ErrNoCoordinates
		}
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return models.Location{}, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}
	return location, nil
}
