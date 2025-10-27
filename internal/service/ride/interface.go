package ride

import (
	"context"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideRepo interface {
	Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)
	Update(ctx context.Context, ride *models.Ride) error
	Get(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
	UpdateStatus(ctx context.Context, rideID uuid.UUID, status types.RideStatus) error
	UpdateMatchedAt(ctx context.Context, rideID uuid.UUID) error
	// для генерации уникального номера поездки (ride_number)
	CountByDate(ctx context.Context, date time.Time) (int, error)
}

type RideMsgBroker interface {
	PublishRideRequested(ctx context.Context, msg models.RideRequestedMessage) error
	PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error
}

type RideWsHandler interface {
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, data any) error
}