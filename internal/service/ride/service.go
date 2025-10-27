package ride

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	wshandler "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/ws"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	ridecalc "github.com/Temutjin2k/ride-hail-system/internal/service/calculator"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideService struct {
	repo      RideRepo
	logger    logger.Logger
	trm       trm.TxManager
	publisher RideMsgBroker
	calculate ridecalc.Calculator

	passengerSender wshandler.RideWsHandler
}

func NewRideService(repo RideRepo, calculate ridecalc.Calculator, trm trm.TxManager, publisher RideMsgBroker, passengerSender wshandler.RideWsHandler, logger logger.Logger) *RideService {
	return &RideService{
		repo:            repo,
		calculate:       calculate,
		trm:             trm,
		publisher:       publisher,
		passengerSender: passengerSender,
		logger:          logger,
	}
}

func (s *RideService) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error) {
	ctx = wrap.WithAction(ctx, "create_ride")

	var createdRide *models.Ride

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		// проверить, есть ли у пассажира активная поездка
		activeRide, err := s.repo.CheckActiveRideByPassengerID(ctx, ride.PassengerID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check passenger's active ride: %w", err))
		}

		// если у пассажира уже есть активная поездка, вернуть ошибку
		if activeRide != nil {
			return types.ErrPassengerHasActiveRide
		}

		distance := s.calculate.Distance(ride.Pickup, ride.Destination)
		duration := s.calculate.Duration(distance)
		fare := s.calculate.Fare(ride.RideType, distance, duration)
		priority := s.calculate.Priority(ride)
		rideNumber, err := s.generateRideNumber(ctx)
		if err != nil {
			return fmt.Errorf("could not generate ride number: %w", err)
		}

		ride.EstimatedDistanceKm = distance
		ride.EstimatedDurationMin = duration
		ride.EstimatedFare = fare
		ride.RideNumber = rideNumber
		ride.Status = types.StatusRequested
		ride.Priority = priority

		createdRide, err = s.repo.Create(ctx, ride)
		if err != nil {
			return fmt.Errorf("could not create ride in repo: %w", err)
		}
		ctx = wrap.WithRideID(ctx, createdRide.ID.String())

		correlationID := wrap.GetRequestID(ctx) // Используем RequestID как CorrelationID
		if correlationID == "" {                // На случай, если RequestID отсутствует
			correlationID = newCorrelationID()
		}

		message := models.RideRequestedMessage{
			RideID:     createdRide.ID,
			RideNumber: createdRide.RideNumber,
			PickupLocation: models.LocationMessage{
				Lat:     createdRide.Pickup.Latitude,
				Lng:     createdRide.Pickup.Longitude,
				Address: createdRide.Pickup.Address,
			},
			DestinationLocation: models.LocationMessage{
				Lat:     createdRide.Destination.Latitude,
				Lng:     createdRide.Destination.Longitude,
				Address: createdRide.Destination.Address,
			},
			RideType:       createdRide.RideType,
			EstimatedFare:  createdRide.EstimatedFare,
			MaxDistanceKm:  5.0, // Это чтобы не ожидать драйвера из какого нибудь Мадагаскара
			TimeoutSeconds: 120,
			CorrelationID:  correlationID,
			Priority:       uint8(createdRide.Priority),
		}

		if err := s.publisher.PublishRideRequested(ctx, message); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to publish ride requested event: %w", err))
		}

		return nil
	})

	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	s.logger.Info(ctx, "ride created successfully", "ride_id", createdRide.ID)

	return createdRide, nil
}

func (s *RideService) Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error) {
	ctx = wrap.WithAction(wrap.WithRideID(ctx, rideID.String()), "cancel_ride")

	var cancelledRide *models.Ride

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		ride, err := s.repo.Get(ctx, rideID)
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return wrap.Error(ctx, types.ErrRideNotFound)
			}
			return wrap.Error(ctx, fmt.Errorf("could not find ride by id: %w", err))
		}

		if ride.Status == types.StatusCompleted || ride.Status == types.StatusCancelled {
			return types.ErrRideCannotBeCancelled
		}

		now := time.Now()
		ride.Status = types.StatusCancelled
		ride.CancellationReason = &reason
		ride.CancelledAt = &now

		err = s.repo.Update(ctx, ride)
		if err != nil {
			return fmt.Errorf("could not update ride: %w", err)
		}

		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        ride.Status,
			Timestamp:     *ride.CancelledAt,
			DriverID:      ride.DriverID,
			CorrelationID: ride.RideNumber,
		}

		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return fmt.Errorf("failed to publish ride cancelled event: %w", err)
		}

		cancelledRide = ride
		return nil
	})

	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	s.logger.Info(ctx, "ride cancelled successfully")

	return cancelledRide, nil
}

func newCorrelationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp if crypto/rand fails
		return hex.EncodeToString(fmt.Appendf(nil, "%d", time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}
