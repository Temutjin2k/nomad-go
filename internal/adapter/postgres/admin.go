package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepo(db *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{
		db: db,
	}
}

func (r *AdminRepo) GetOverview(ctx context.Context) (any, error) {
	return nil, nil
}

func (r *AdminRepo) GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error) {
	const (
		page     = 1
		pageSize = 20
	)
	offset := (page - 1) * pageSize

	// 1) total_count
	var totalCount int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM rides
		WHERE status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
	`).Scan(&totalCount); err != nil {
		return nil, err
	}

	// If no active rides, return empty response early
	if totalCount == 0 {
		return &models.ActiveRidesResponse{
			Rides:      []models.RideInfo{},
			TotalCount: 0,
			Page:       page,
			PageSize:   pageSize,
		}, nil
	}

	// 2) данные по активным поездкам
	//  - pickup/destination берём из coordinates по *_coordinate_id
	//  - текущую точку водителя берём из coordinates по driver_id, entity_type='driver', is_current=true
	//  - estimated_completion: если известен started_at и у текущей точки водителя есть duration_minutes, добавим их
	//  - distance_completed_km заполним из coordinates.distance_km, если есть (иначе 0)
	//  - distance_remaining_km пока 0 (нужна логика маршрута — можно будет доработать)
	rows, err := r.db.Query(ctx, `
		WITH cur AS (
			SELECT entity_id, latitude, longitude, distance_km, duration_minutes
			FROM coordinates
			WHERE entity_type = 'driver' AND is_current = TRUE
		)
		SELECT
			r.id,
			r.ride_number,
			r.status,
			r.passenger_id,
			r.driver_id,
			COALESCE(pc.address, '') AS pickup_address,
			COALESCE(dc.address, '') AS destination_address,
			r.started_at,
			CASE
				WHEN r.started_at IS NOT NULL AND cur.duration_minutes IS NOT NULL
					THEN r.started_at + make_interval(mins => cur.duration_minutes)
				ELSE NULL
			END AS estimated_completion,
			cur.latitude,
			cur.longitude,
			COALESCE(cur.distance_km, 0)::float AS distance_completed_km
		FROM rides r
		LEFT JOIN coordinates pc ON pc.id = r.pickup_coordinate_id
		LEFT JOIN coordinates dc ON dc.id = r.destination_coordinate_id
		LEFT JOIN cur ON cur.entity_id = r.driver_id
		WHERE r.status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := make([]models.RideInfo, 0, pageSize)
	for rows.Next() {
		var (
			rideID       uuid.UUID
			rideNumber   string
			status       string
			passengerID  uuid.UUID
			driverID     uuid.UUID
			pickupAddr   string
			destAddr     string
			startedAtPtr *time.Time
			estComplPtr  *time.Time
			latNull      sql.NullFloat64
			lonNull      sql.NullFloat64
			distDoneKm   float64
		)

		if err := rows.Scan(
			&rideID,
			&rideNumber,
			&status,
			&passengerID,
			&driverID,
			&pickupAddr,
			&destAddr,
			&startedAtPtr,
			&estComplPtr,
			&latNull,
			&lonNull,
			&distDoneKm,
		); err != nil {
			return nil, err
		}

		ri := models.RideInfo{
			RideID:             rideID,
			RideNumber:         rideNumber,
			Status:             status,
			PassengerID:        passengerID,
			DriverID:           driverID,
			PickupAddress:      pickupAddr,
			DestinationAddress: destAddr,
			CurrentDriverLocation: models.Location{
				Latitude:  0,
				Longitude: 0,
			},
			DistanceCompletedKm: distDoneKm,
			// Нет прямого источника для оставшейся дистанции — оставляем 0.
			DistanceRemainingKm: 0,
		}

		if startedAtPtr != nil {
			ri.StartedAt = *startedAtPtr
		}
		if estComplPtr != nil {
			ri.EstimatedCompletion = *estComplPtr
		}
		if latNull.Valid {
			ri.CurrentDriverLocation.Latitude = latNull.Float64
		}
		if lonNull.Valid {
			ri.CurrentDriverLocation.Longitude = lonNull.Float64
		}

		rides = append(rides, ri)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &models.ActiveRidesResponse{
		Rides:      rides,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}
	return resp, nil
}
