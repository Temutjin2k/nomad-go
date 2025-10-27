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
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideRepo struct {
	db *pgxpool.Pool
}

func NewRideRepo(db *pgxpool.Pool) *RideRepo {
	return &RideRepo{db: db}
}

func (r *RideRepo) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error) {
	q := TxorDB(ctx, r.db)

	var pickupCoordID uuid.UUID
	coordQuery := `INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude)
                   VALUES ($1, 'passenger', $2, $3, $4) RETURNING id;`

	err := q.QueryRow(ctx, coordQuery, ride.PassengerID, ride.Pickup.Address, ride.Pickup.Latitude, ride.Pickup.Longitude).Scan(&pickupCoordID)
	if err != nil {
		return nil, fmt.Errorf("rideRide Repo: Create (pickup coord): %w", err)
	}

	var destCoordID uuid.UUID
	err = q.QueryRow(ctx, coordQuery, ride.PassengerID, ride.Destination.Address, ride.Destination.Latitude, ride.Destination.Longitude).Scan(&destCoordID)
	if err != nil {
		return nil, fmt.Errorf("ride repo: Create (dest coord): %w", err)
	}

	rideQuery := `INSERT INTO rides (ride_number, passenger_id, vehicle_type, status, estimated_fare, 
                                     pickup_coordinate_id, destination_coordinate_id, priority )
                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                  RETURNING id, created_at;`

	err = q.QueryRow(ctx, rideQuery, ride.RideNumber, ride.PassengerID, ride.RideType, ride.Status, ride.EstimatedFare, pickupCoordID, destCoordID, ride.Priority).Scan(&ride.ID, &ride.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("ride repo: Create (ride): %w", err)
	}

	return ride, nil
}

func (r *RideRepo) CountByDate(ctx context.Context, date time.Time) (int, error) {
	q := TxorDB(ctx, r.db)

	var count int
	query := "SELECT COUNT(*) FROM rides WHERE DATE(created_at) = $1;"

	err := q.QueryRow(ctx, query, date.Format("2006-01-02")).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ride repo: CountByDate: %w", err)
	}
	return count, nil
}

func (r *RideRepo) Get(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
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
			return nil, types.ErrRideNotFound
		}
		return nil, fmt.Errorf("ride repo: Get: %w", err)
	}

	return &ride, nil
}

func (r *RideRepo) Update(ctx context.Context, ride *models.Ride) error {
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
		return fmt.Errorf("ride repo: Update: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return wrap.Error(ctx, types.ErrNotFound)
	}

	return nil
}

func (r *RideRepo) StartRide(ctx context.Context, rideID, driverID uuid.UUID, startedAt time.Time, rideEvent models.RideEvent) error {
	const op = "rideRepo.StartRide"

	// Update the ride status to 'IN_PROGRESS' and set the started_at timestamp
	updateQuery := `
        UPDATE rides
        SET status = 'IN_PROGRESS', updated_at = now(), started_at = $2
        WHERE id = $1'`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, updateQuery, rideID, startedAt); err != nil {
		return wrap.Error(wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed), fmt.Errorf("%s: failed to update ride status: %w", op, err))
	}

	// Insert a new ride event into the ride_events table
	insertEventQuery := `
        INSERT INTO ride_events(ride_id, event_type, event_data)
        VALUES($1, 'RIDE_STARTED', $2)`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, insertEventQuery, rideID, rideEvent); err != nil {
		if postgres.IsForeignKeyViolation(err) {
			return types.ErrRideNotFound
		}

		return wrap.Error(wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed), fmt.Errorf("%s: failed to insert ride event: %w", op, err))
	}

	return nil
}

func (r *RideRepo) CompleteRide(ctx context.Context, rideID, driverID uuid.UUID, completedAt time.Time, rideEvent models.RideEvent) error {
	const op = "ideRepo.CompleteRide"

	// Update the ride status to 'COMPLETED' and set the completed_at timestamp
	updateQuery := `
        UPDATE rides
        SET status = 'COMPLETED', updated_at = now(), completed_at = $2
        WHERE id = $1;`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, updateQuery, rideID, completedAt); err != nil {
		return wrap.Error(wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed), fmt.Errorf("%s: failed to update ride status: %w", op, err))
	}

	// Insert a new ride event into the ride_events table
	insertEventQuery := `
        INSERT INTO ride_events(ride_id, event_type, event_data)
        VALUES($1, 'RIDE_COMPLETED', $2);`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, insertEventQuery, rideID, rideEvent); err != nil {
		if postgres.IsForeignKeyViolation(err) {
			return types.ErrRideNotFound
		}
		return wrap.Error(wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed), fmt.Errorf("%s: failed to insert ride event: %w", op, err))
	}

	return nil
}

func (r *RideRepo) UpdateStatus(ctx context.Context, rideID uuid.UUID, status types.RideStatus) error {
	q := TxorDB(ctx, r.db)

	query := `
		UPDATE rides
		SET
			status = $2,
			updated_at = now()
		WHERE id = $1;`
	cmdTag, err := q.Exec(ctx, query,
		rideID,
		status,
	)

	if err != nil {
		return fmt.Errorf("ride repo: UpdateStatus: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return wrap.Error(ctx, types.ErrNotFound)
	}

	return nil
}

func (r *RideRepo) GetDetails(ctx context.Context, rideID uuid.UUID) (*models.RideDetails, error) {
	const op = "RideRepo.RideDetails"
	query := `
		SELECT 
    		r.id AS ride_id,
			COALESCE(r.driver_id, ''::uuid) AS driver_id,
    		u.attrs->>'name' AS passenger_name,
    		u.attrs->>'phone' AS passenger_phone,
    		c.latitude AS pickup_latitude,
    		c.longitude AS pickup_longitude
		FROM rides r
		INNER JOIN users u ON r.passenger_id = u.id
		INNER JOIN coordinates c ON r.pickup_coordinate_id = c.id
		WHERE r.id = $1;`

	var details models.RideDetails
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, rideID).Scan(&details.RideID, &details.DriverID, &details.Passenger.Name, &details.Passenger.Phone, &details.PickupLocation.Latitude, &details.PickupLocation.Longitude); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, types.ErrRideNotFound
		}
		return nil, wrap.Error(ctx, fmt.Errorf("%s: failed to get ride details: %w", op, err))
	}

	return &details, nil
}

func (r *CoordinateRepo) GetPickupCoordinate(ctx context.Context, rideID uuid.UUID) (*models.Location, error) {
	const op = "CoordinateRepo.GetDestination"
	query := `
		SELECT c.latitude, c.longitude, c.address FROM rides r
		INNER JOIN coordinates c ON r.pickup_coordinate_id = c.id
		WHERE r.id = $1`

	var location models.Location
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, rideID).Scan(&location.Latitude, &location.Longitude, &location.Address); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, types.ErrRideNotFound
		}
		return nil, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return &location, nil
}
