package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type AdminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepo(db *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{
		db: db,
	}
}

func (r *AdminRepo) GetOverview(ctx context.Context) (*models.OverviewResponse, error) {
	db := TxorDB(ctx, r.db)

	// Metrics container
	var (
		activeRides    int
		availableDrvs  int
		busyDrvs       int
		totalToday     int
		revenueToday   float64
		avgWaitMin     float64
		avgDurMin      float64
		cancelledToday int
	)

	// Active rides
	if err := db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM rides
        WHERE status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
    `).Scan(&activeRides); err != nil {
		return nil, err
	}

	// Available drivers
	if err := db.QueryRow(ctx, `
        SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE'
    `).Scan(&availableDrvs); err != nil {
		return nil, err
	}

	// Busy drivers (BUSY or EN_ROUTE)
	if err := db.QueryRow(ctx, `
        SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY','EN_ROUTE')
    `).Scan(&busyDrvs); err != nil {
		return nil, err
	}

	// Total rides today (by creation date)
	if err := db.QueryRow(ctx, `
        SELECT COUNT(*) FROM rides WHERE created_at::date = CURRENT_DATE
    `).Scan(&totalToday); err != nil {
		return nil, err
	}

	// Total revenue today (sum of final_fare on completed today)
	if err := db.QueryRow(ctx, `
        SELECT COALESCE(SUM(final_fare), 0)::float
        FROM rides
        WHERE completed_at::date = CURRENT_DATE
    `).Scan(&revenueToday); err != nil {
		return nil, err
	}

	// Average wait time minutes (matched - requested) for rides requested today
	if err := db.QueryRow(ctx, `
        SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))), 0)::float / 60.0
        FROM rides
        WHERE requested_at::date = CURRENT_DATE AND matched_at IS NOT NULL
    `).Scan(&avgWaitMin); err != nil {
		return nil, err
	}

	// Average ride duration minutes (completed - started) for rides completed today
	if err := db.QueryRow(ctx, `
        SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))), 0)::float / 60.0
        FROM rides
        WHERE completed_at::date = CURRENT_DATE AND started_at IS NOT NULL
    `).Scan(&avgDurMin); err != nil {
		return nil, err
	}

	// Cancellation rate components
	if err := db.QueryRow(ctx, `
        SELECT COUNT(*) FROM rides WHERE created_at::date = CURRENT_DATE AND status = 'CANCELLED'
    `).Scan(&cancelledToday); err != nil {
		return nil, err
	}
	var cancellationRate float64
	if totalToday > 0 {
		cancellationRate = float64(cancelledToday) / float64(totalToday)
	} else {
		cancellationRate = 0
	}

	// Driver distribution by vehicle_type for AVAILABLE drivers
	driverDistribution := make(map[string]int)
	distRows, err := db.Query(ctx, `
        SELECT COALESCE(vehicle_type, 'UNKNOWN') AS vt, COUNT(*)
        FROM drivers
        WHERE status = 'AVAILABLE'
        GROUP BY vt
    `)
	if err != nil {
		return nil, err
	}
	defer distRows.Close()
	for distRows.Next() {
		var vt string
		var cnt int
		if err := distRows.Scan(&vt, &cnt); err != nil {
			return nil, err
		}
		driverDistribution[vt] = cnt
	}
	if err := distRows.Err(); err != nil {
		return nil, err
	}

	// Hotspots: combine active rides by pickup address and available drivers waiting by their current address
	hotspotRows, err := db.Query(ctx, `
        WITH active_by_pickup AS (
            SELECT c.address, COUNT(*)::int AS active_rides
            FROM rides r
            JOIN coordinates c ON c.id = r.pickup_coordinate_id
            WHERE r.status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
            GROUP BY c.address
        ), waiting_by_address AS (
            SELECT c.address, COUNT(*)::int AS waiting_drivers
            FROM coordinates c
            JOIN drivers d ON d.id = c.entity_id
            WHERE c.entity_type = 'driver' AND c.is_current = TRUE AND d.status = 'AVAILABLE'
            GROUP BY c.address
        )
        SELECT COALESCE(a.address, w.address) AS location,
               COALESCE(a.active_rides, 0)    AS active_rides,
               COALESCE(w.waiting_drivers, 0) AS waiting_drivers
        FROM active_by_pickup a
        FULL OUTER JOIN waiting_by_address w ON a.address = w.address
        ORDER BY (COALESCE(a.active_rides,0) + COALESCE(w.waiting_drivers,0)) DESC
        LIMIT 5;
    `)
	if err != nil {
		return nil, err
	}
	defer hotspotRows.Close()

	hotspots := make([]models.Hotspot, 0, 5)
	for hotspotRows.Next() {
		var h models.Hotspot
		if err := hotspotRows.Scan(&h.Location, &h.ActiveRides, &h.WaitingDrivers); err != nil {
			return nil, err
		}
		hotspots = append(hotspots, h)
	}
	if err := hotspotRows.Err(); err != nil {
		return nil, err
	}

	resp := &models.OverviewResponse{
		Timestamp: time.Now().UTC(),
		Metrics: models.Metrics{
			ActiveRides:                activeRides,
			AvailableDrivers:           availableDrvs,
			BusyDrivers:                busyDrvs,
			TotalRidesToday:            totalToday,
			TotalRevenueToday:          revenueToday,
			AverageWaitTimeMinutes:     avgWaitMin,
			AverageRideDurationMinutes: avgDurMin,
			CancellationRate:           cancellationRate,
		},
		DriverDistribution: driverDistribution,
		Hotspots:           hotspots,
	}

	return resp, nil
}

func (r *AdminRepo) GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error) {
	const (
		page     = 1
		pageSize = 20
	)
	offset := (page - 1) * pageSize

	db := TxorDB(ctx, r.db)

	// 1) total_count
	var totalCount int
	if err := db.QueryRow(ctx, `
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
	rows, err := db.Query(ctx, `
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
			dc.latitude AS destination_latitude,
			dc.longitude AS destination_longitude,
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
            driverIDNull sql.NullString
            pickupAddr   string
            destAddr     string
            destLatNull  sql.NullFloat64
            destLonNull  sql.NullFloat64
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
			&driverIDNull,
            &pickupAddr,
            &destAddr,
            &destLatNull,
            &destLonNull,
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
			PickupAddress:      pickupAddr,
			DestinationAddress: destAddr,
			CurrentDriverLocation: models.LocationInfo{
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
		if driverIDNull.Valid {
			if parsed, err := uuid.Parse(driverIDNull.String); err == nil {
				ri.DriverID = parsed
			}
		}
		if latNull.Valid {
			ri.CurrentDriverLocation.Latitude = latNull.Float64
		}
            if lonNull.Valid {
                ri.CurrentDriverLocation.Longitude = lonNull.Float64
            }

            if destLatNull.Valid {
                ri.DestinationLocation.Latitude = destLatNull.Float64
            }
            if destLonNull.Valid {
                ri.DestinationLocation.Longitude = destLonNull.Float64
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
