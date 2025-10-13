package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type DriverRepo struct {
	db *pgxpool.Pool
}

func NewDriverRepo(db *pgxpool.Pool) *DriverRepo {
	return &DriverRepo{
		db: db,
	}
}

func (r *DriverRepo) Create(ctx context.Context, driver *models.Driver) error {
	const op = "DriverRepo.Create"
	query := `
		INSERT INTO drivers(id, name, license_number, vehicle_type, is_verified, vehicle_attrs)
		VALUES($1, $2, $3, $4, $5, $6)`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, query,
		driver.ID,
		driver.Name,
		driver.LicenseNumber,
		driver.Vehicle.Type,
		driver.IsVerified,
		driver.Vehicle,
	); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return nil
}

// IsUnique checks driver uniqueness by license num
func (r *DriverRepo) IsUnique(ctx context.Context, validLicenseNum string) (bool, error) {
	const op = "DriverRepo.IsUnique"
	query := `
		SELECT EXISTS(
			SELECT 1 FROM drivers
			WHERE license_number = $1
		)`

	var exist bool
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, validLicenseNum).Scan(&exist); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return false, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return !exist, nil
}

func (r *DriverRepo) IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error) {
	const op = "DriverRepo.IsDriverExist"
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users u
			JOIN drivers d ON d.id = u.id
			WHERE u.id = $1 AND u.role = $2
		) AS driverExistence`

	var exist bool
	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, id, types.RoleDriver).Scan(&exist); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return false, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return exist, nil
}

func (r *DriverRepo) ChangeStatus(ctx context.Context, driverID uuid.UUID, newStatus types.DriverStatus) (oldStatus types.DriverStatus, err error) {
	const op = "DriverRepo.ChangeStatus"
	query := `
		WITH old AS (
    		SELECT id, status
    		FROM drivers
    		WHERE id = $1
		)
		UPDATE drivers
		SET status = $2, updated_at = now()
		FROM old
		WHERE drivers.id = old.id
		RETURNING old.status;`

	if err := TxorDB(ctx, r.db).QueryRow(ctx, query, driverID, newStatus).Scan(&oldStatus); err != nil {
		ctx = wrap.WithAction(ctx, types.ActionDatabaseTransactionFailed)
		return "", wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return oldStatus, nil
}

func (r *DriverRepo) UpdateStats(ctx context.Context, driverID uuid.UUID, ridesCompleted int, earnings float64) error {
	const op = "DriverRepo.UpdateStats"
	query := `
		UPDATE drivers
		SET 
			total_rides = total_rides + $1,
		 	total_earnings = total_earnings + $2,
			updated_at = now()
		WHERE id = $3`

	if _, err := TxorDB(ctx, r.db).Exec(ctx, query, ridesCompleted, earnings, driverID); err != nil {
		return wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return nil
}

func (r *DriverRepo) Get(ctx context.Context, driverID uuid.UUID) (*models.Driver, error) {
	const op = "DriverRepo.Get"
	query := `
        SELECT id,
               name, 
               created_at, 
               updated_at, 
               license_number, 
               vehicle_type, 
               vehicle_attrs, 
               rating, 
               total_rides, 
               total_earnings, 
               status, 
               is_verified
        FROM drivers
        WHERE id = $1`

	var driver models.Driver
	err := TxorDB(ctx, r.db).QueryRow(ctx, query, driverID).Scan(
		&driver.ID,
		&driver.Name,
		&driver.CreatedAt,
		&driver.UpdatedAt,
		&driver.LicenseNumber,
		&driver.Vehicle.Type,
		&driver.Vehicle,
		&driver.Rating,
		&driver.TotalRides,
		&driver.TotalEarnings,
		&driver.Status,
		&driver.IsVerified,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, types.ErrUserNotFound
		}
		return nil, wrap.Error(ctx, fmt.Errorf("%s: %w", op, err))
	}

	return &driver, nil
}
