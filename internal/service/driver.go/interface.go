package drivergo

import (
	"context"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

/*=================Driver Repository======================*/

type DriverRepo interface {
	DriverCreator
	DriverChecker
	LicenseChecker
	DriverGetter
	DriverUpdater
}

type DriverCreator interface {
	Create(ctx context.Context, driver *models.Driver) error
}

type DriverUpdater interface {
	ChangeStatus(ctx context.Context, driverID uuid.UUID, newStatus types.DriverStatus) (oldStatus types.DriverStatus, err error)
	UpdateStats(ctx context.Context, driverID uuid.UUID, ridesCompleted int, earnings float64) error
}

type LicenseChecker interface {
	IsUnique(ctx context.Context, validLicenseNum string) (bool, error)
}

type DriverChecker interface {
	IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error)
}

type DriverGetter interface {
	Get(ctx context.Context, driverID uuid.UUID) (models.Driver, error)
}

/*=================Driver Session Repository======================*/

type DriverSessionRepo interface {
	SessionCreator
	SummaryGetter
}

type SessionCreator interface {
	Create(ctx context.Context, driverID uuid.UUID) (sessiondID uuid.UUID, err error)
}

type SummaryGetter interface {
	GetSummary(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error)
}

/*=================Coordinate Repository==========================*/

type CoordinateRepo interface {
	Create(ctx context.Context, entityID uuid.UUID, entityType types.EntityType, address string, latitude, longitude float64) (uuid.UUID, error)
	GetDriverLastCoordinate(ctx context.Context, driverID uuid.UUID) (models.Location, error)
}

/*===================== Address Geo Coder ========================*/

type GeoCoder interface {
	GetAddress(ctx context.Context, longitude, latitude float64) (string, error)
}

/*=====================User Repository============================*/

type UserRepo interface {
	RoleChanger
}

type RoleChanger interface {
	ChangeRole(ctx context.Context, userID uuid.UUID, new types.UserRole) (old types.UserRole, err error)
}

/*=====================Ride Repository============================*/

type RideRepo interface {
	RideStatusChanger
	RideGetter
}

type RideStatusChanger interface {
	StartRide(ctx context.Context, rideID, driverID uuid.UUID, startedAt time.Time, rideEvent models.RideEvent) error
}

type RideGetter interface {
	Get(ctx context.Context, rideID uuid.UUID) (models.Ride, error)
}

/*========================Publisher===============================*/
type Publisher interface {
	StatusPublisher
}

type StatusPublisher interface {
	PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error
	PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error
}
