package repo

import (
	"context"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	if _, err := r.db.Exec(ctx, query,
		driver.ID,
		driver.Name,
		driver.LicenseNumber,
		driver.Vehicle.Type,
		driver.IsVerified,
		driver.Vehicle,
	); err != nil {
		return fmt.Errorf("%s: %v", op, err)
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
	if err := r.db.QueryRow(ctx, query, validLicenseNum).Scan(&exist); err != nil {
		return false, fmt.Errorf("%s: %v", op, err)
	}

	return !exist, nil
}

func (r *DriverRepo) IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error) {
	const op = "DriverRepo.IsDriverExist"
	query := `
		SELECT d.ID is NOT NULL as driverExistence 
		FROM users u
		LEFT JOIN drivers d ON d.id = u.id
		WHERE u.id = $1 AND u.role =  $2;	
	`

	var exist bool
	if err := r.db.QueryRow(ctx, query, id, types.DriverRole).Scan(&exist); err != nil {
		if err == pgx.ErrNoRows {
			return false, types.ErrUserNotFound
		}
		return false, fmt.Errorf("%s: %v", op, err)
	}

	return exist, nil
}
