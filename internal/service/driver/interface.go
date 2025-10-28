package drivergo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

/*=================Driver Repository======================*/

type DriverRepo interface {
	LicenseChecker
	Create(ctx context.Context, driver *models.Driver) error
	IsDriverExist(ctx context.Context, id uuid.UUID) (bool, error)
	Get(ctx context.Context, driverID uuid.UUID) (*models.Driver, error)
	SearchDrivers(ctx context.Context, rideType string, pickUplocation models.Location) ([]models.DriverWithDistance, error)
	ChangeStatus(ctx context.Context, driverID uuid.UUID, newStatus types.DriverStatus) (oldStatus types.DriverStatus, err error)
	UpdateStats(ctx context.Context, driverID uuid.UUID, ridesCompleted int, earnings float64) error
}

type LicenseChecker interface {
	IsLicenseExists(ctx context.Context, validLicenseNum string) (bool, error)
}

/*=================Driver Session Repository======================*/

type DriverSessionRepo interface {
	Create(ctx context.Context, driverID uuid.UUID) (sessiondID uuid.UUID, err error)
	GetSummary(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error)
	Update(ctx context.Context, driverID uuid.UUID, ridesCompleted int, earnings float64) error
}

/*=================Coordinate Repository==========================*/

type CoordinateRepo interface {
	CreateCoordinate(ctx context.Context, entityID uuid.UUID, entityType types.EntityType, location models.Location, updatedAt time.Time) (uuid.UUID, error)
	CreateLocationHistory(ctx context.Context, coordinateID, driverID uuid.UUID, rideID *uuid.UUID, location models.Location, accuracyMeters, speedKmh, headingDegrees float64) (uuid.UUID, error)
	GetDriverLastCoordinate(ctx context.Context, driverID uuid.UUID) (models.Location, error)
}

/*===================== Address Geo Coder ========================*/

type GeoCoder interface {
	GetAddress(ctx context.Context, longitude, latitude float64) (string, error)
}

/*=====================User Repository============================*/

type UserRepo interface {
	ChangeRole(ctx context.Context, userID uuid.UUID, new types.UserRole) (old types.UserRole, err error)
}

/*=====================Ride Repository============================*/

type RideGetter interface {
	Get(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
	GetDetails(ctx context.Context, rideID uuid.UUID) (*models.RideDetails, error)
	GetPickupCoordinate(ctx context.Context, rideID uuid.UUID) (*models.Location, error)
}

/*========================Publisher===============================*/

type Publisher interface {
	PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error
	PublishDriverResponse(ctx context.Context, resp models.DriverMatchResponse) error
	PublishLocationUpdate(ctx context.Context, msg models.RideLocationUpdate) error
}

/*===========================Sender===============================*/

type DriverCommunicator interface {
	SendRideOffer(ctx context.Context, driverID uuid.UUID, offer models.RideOffer) (bool, error)
	SendRideDetails(ctx context.Context, details models.RideDetails) error
	ListenLocationUpdates(ctx context.Context, driverID, rideID uuid.UUID, handler func(ctx context.Context, location models.RideLocationUpdate) error) error
}

type RideEventRepository interface {
	// CreateEvent записывает событие, связанное с поездкой в таблицу ride_events
	CreateEvent(ctx context.Context, rideID uuid.UUID, eventType types.RideEvent, eventData json.RawMessage) error
}
