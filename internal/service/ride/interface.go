package ride

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/rabbit"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type (
	RideRepo interface {
		Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)
		Update(ctx context.Context, ride *models.Ride) error
		Get(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
		UpdateStatus(ctx context.Context, rideID uuid.UUID, status types.RideStatus) error
		UpdateMatchedAt(ctx context.Context, rideID uuid.UUID) error
		UpdateArrivedAt(ctx context.Context, rideID uuid.UUID) error
		UpdateCompletedAt(ctx context.Context, rideID uuid.UUID) error
		UpdateStartedAt(ctx context.Context, rideID uuid.UUID) error
		// для генерации уникального номера поездки (ride_number)
		CountByDate(ctx context.Context, date time.Time) (int, error)

		// проверить, есть ли у пассажира активная поездка
		CheckActiveRideByPassengerID(ctx context.Context, passengerID uuid.UUID) (*models.Ride, error)

		DriverMatchedForRide(ctx context.Context, rideID, driverID uuid.UUID, finalFare float64) error
	}

	RideMsgBroker interface {
		PublishRideRequested(ctx context.Context, msg models.RideRequestedMessage) error
		PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error
		ConsumeDriverResponse(ctx context.Context, rideID uuid.UUID, handler rabbit.DriverResponseHandler) error
	}

	RideWsHandler interface {
		SendToPassenger(ctx context.Context, passengerID uuid.UUID, data any) error
	}

	// RideEventRepository defines methods for logging ride events.
	RideEventRepository interface {
		// CreateEvent записывает событие, связанное с поездкой в таблицу ride_events
		CreateEvent(ctx context.Context, rideID uuid.UUID, eventType types.RideEvent, eventData json.RawMessage) error
	}
)
