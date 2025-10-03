package drivergo

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type Repo interface {
	DriverCreator
	DriverChecker
	LicenseChecker
}

type DriverCreator interface {
	Create(ctx context.Context, driver *models.Driver) error
}

type LicenseChecker interface {
	IsUnique(ctx context.Context, validLicenseNum string) (bool, error)
}

type DriverChecker interface {
	IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error)
}
