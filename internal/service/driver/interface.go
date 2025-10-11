package drivergo

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

/*=================Driver Repository======================*/

type DriverRepo interface {
	DriverCreator
	DriverChecker
	LicenseChecker
	DriverModifier
}

type DriverCreator interface {
	Create(ctx context.Context, driver *models.Driver) error
}

type DriverModifier interface {
	ChangeStatus(ctx context.Context, driverID uuid.UUID, newStatus types.DriverStatus) (oldStatus types.DriverStatus, err error)
}

type LicenseChecker interface {
	IsUnique(ctx context.Context, validLicenseNum string) (bool, error)
}

type DriverChecker interface {
	IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error)
}

/*=================Driver Session Repository======================*/

type DriverSessionRepo interface {
	SessionCreator
}

type SessionCreator interface {
	Create(ctx context.Context, driverID uuid.UUID) (sessiondID uuid.UUID, err error)
}

/*=================Coordinate Repository==========================*/

type CoordinateRepo interface {
	Create(ctx context.Context, entityID uuid.UUID, entityType types.EntityType, address string, latitude, longitude float64) (uuid.UUID, error)
}

/*=================Driver Session Repository======================*/

type AddresGetter interface {
	GetAddress(ctx context.Context, longitude, latitude float64) (string, error)
}
