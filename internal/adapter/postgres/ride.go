package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideRepository struct {
	db *pgxpool.Pool
}

func NewRideRepository(db *pgxpool.Pool) *RideRepository {
	return &RideRepository{db: db}
}

func (r *RideRepository) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error) {
	q := TxorDB(ctx, r.db)

	var pickupCoordID uuid.UUID
	coordQuery := `INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude)
                   VALUES ($1, 'passenger', $2, $3, $4) RETURNING id;`

	err := q.QueryRow(ctx, coordQuery, ride.PassengerID, ride.Pickup.Address, ride.Pickup.Latitude, ride.Pickup.Longitude).Scan(&pickupCoordID)
	if err != nil {
		return nil, wrap.Error(ctx, fmt.Errorf("RideRide Repo: Create (pickup coord): %w", err))
	}

	var destCoordID uuid.UUID
	err = q.QueryRow(ctx, coordQuery, ride.PassengerID, ride.Destination.Address, ride.Destination.Latitude, ride.Destination.Longitude).Scan(&destCoordID)
	if err != nil {
		return nil, fmt.Errorf("Ride repo: Create (dest coord): %w", err)
	}

	rideQuery := `INSERT INTO rides (ride_number, passenger_id, vehicle_type, status, estimated_fare, 
                                     pickup_coordinate_id, destination_coordinate_id)
                  VALUES ($1, $2, $3, $4, $5, $6, $7)
                  RETURNING id, created_at;`

	err = q.QueryRow(ctx, rideQuery, ride.RideNumber, ride.PassengerID, ride.RideType, ride.Status, ride.EstimatedFare, pickupCoordID, destCoordID).Scan(&ride.ID, &ride.CreatedAt)
	if err != nil {
		return nil, wrap.Error(ctx, fmt.Errorf("Ride repo: Create (ride): %w", err))
	}

	return ride, nil
}

func (r *RideRepository) CountByDate(ctx context.Context, date time.Time) (int, error) {
    q := TxorDB(ctx, r.db) 

	var count int
	query := "SELECT COUNT(*) FROM rides WHERE DATE(created_at) = $1;"
	
	err := q.QueryRow(ctx, query, date.Format("2006-01-02")).Scan(&count)
	if err != nil {
		return 0, wrap.Error(ctx,fmt.Errorf("Ride repo: CountByDate: %w", err))
	}
	return count, nil
}

func (r *RideRepository) FindByID(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
    q := TxorDB(ctx, r.db)

    var ride models.Ride
    // JOIN чтобы сразу получить адреса
    query := `
        SELECT
            r.id, r.ride_number, r.status, r.passenger_id, r.driver_id, r.vehicle_type,
            r.estimated_fare, r.final_fare, r.cancellation_reason,
            r.created_at, r.matched_at, r.arrived_at, r.started_at, r.completed_at, r.cancelled_at,
            p.address as pickup_address, p.latitude as pickup_lat, p.longitude as pickup_lon,
            d.address as dest_address, d.latitude as dest_lat, d.longitude as dest_lon
        FROM rides r
        JOIN coordinates p ON r.pickup_coordinate_id = p.id
        JOIN coordinates d ON r.destination_coordinate_id = d.id
        WHERE r.id = $1;`

    row := q.QueryRow(ctx, query, rideID)
    err := row.Scan(
        &ride.ID, &ride.RideNumber, &ride.Status, &ride.PassengerID, &ride.DriverID, &ride.RideType,
        &ride.EstimatedFare, &ride.FinalFare, &ride.CancellationReason,
        &ride.CreatedAt, &ride.MatchedAt, &ride.ArrivedAt, &ride.StartedAt, &ride.CompletedAt, &ride.CancelledAt,
        &ride.Pickup.Address, &ride.Pickup.Latitude, &ride.Pickup.Longitude,
        &ride.Destination.Address, &ride.Destination.Latitude, &ride.Destination.Longitude,
    )

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, wrap.Error(ctx, types.ErrNotFound) 
        }
        return nil, wrap.Error(ctx, fmt.Errorf("Ride repo: FindByID: %w", err))
    }

    return &ride, nil
}

func (r *RideRepository) Update(ctx context.Context, ride *models.Ride) error {
    q := TxorDB(ctx, r.db)

    query := `
        UPDATE rides
        SET
            status = $2,
            driver_id = $3,
            final_fare = $4,
            cancellation_reason = $5,
            matched_at = $6,
            arrived_at = $7,
            started_at = $8,
            completed_at = $9,
            cancelled_at = $10,
            updated_at = now()
        WHERE id = $1;`

    cmdTag, err := q.Exec(ctx, query,
        ride.ID,
        ride.Status,
        ride.DriverID,
        ride.FinalFare,
        ride.CancellationReason,
        ride.MatchedAt,
        ride.ArrivedAt,
        ride.StartedAt,
        ride.CompletedAt,
        ride.CancelledAt,
    )

    if err != nil {
        return wrap.Error(ctx, fmt.Errorf("Ride repo: Update: %w", err))
    }

    if cmdTag.RowsAffected() == 0 {
        return wrap.Error(ctx, types.ErrNotFound) 
    }

    return nil
}
