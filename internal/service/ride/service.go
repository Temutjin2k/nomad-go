package ride

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideService struct {
	repo   RideRepo
	logger logger.Logger
	trm    trm.TxManager
	publisher RideMsgBroker
}

func NewRideService(repo RideRepo, logger logger.Logger, trm trm.TxManager, publisher RideMsgBroker) *RideService {
	return &RideService{
		repo: repo,
		logger: logger,
		trm: trm,
		publisher: publisher,
	}
}

func (s *RideService) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error) {
	ctx = wrap.WithAction(ctx, "create_ride")

	var createdRide *models.Ride

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		distance := calculateDistance(ride.Pickup, ride.Destination)
		duration := calculateDuration(distance)
		fare := calculateFare(ride.RideType, distance, duration)
		priority := calculatePriority(ride)
		rideNumber, err := s.generateRideNumber(ctx)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("could not generate ride number: %w", err))
		}

		ride.EstimatedDistanceKm = distance
		ride.EstimatedDurationMin = duration
		ride.EstimatedFare = fare
		ride.RideNumber = rideNumber
		ride.Status = "REQUESTED"
		ride.Priority = priority

		createdRide, err = s.repo.Create(ctx, ride)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("could not create ride in repo: %w", err))
		}

		return nil
	})

	if err != nil {
		return nil, wrap.Error(ctx, err)
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
		CorrelationID:  createdRide.RideNumber,
	}

	if err := s.publisher.PublishRideRequested(ctx, message); err != nil {
		s.logger.Error(wrap.ErrorCtx(ctx, err), "critical: failed to publish ride requested event", err)
	}

	s.logger.Info(ctx, "ride created successfully", "ride_id", createdRide.ID)

	return createdRide, nil
}

func (s *RideService) Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error) {
	ctx = wrap.WithAction(ctx, "cancel_ride")

	var cancelledRide *models.Ride

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		ride, err := s.repo.Get(ctx, rideID)
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return wrap.Error(ctx,types.ErrRideNotFound)
			}
			return wrap.Error(ctx, fmt.Errorf("could not find ride by id: %w", err))
		}

		if ride.Status == "COMPLETED" || ride.Status == "CANCELLED" {
			return types.ErrRideCannotBeCancelled
		}

		now := time.Now()
		ride.Status = "CANCELLED"
		ride.CancellationReason = &reason
		ride.CancelledAt = &now

		err = s.repo.Update(ctx, ride)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("could not update ride: %w", err))
		}

		cancelledRide = ride
		return nil
	})

	if err != nil {
		return nil, err 
	}

	message := models.RideStatusUpdateMessage{
		RideID:        cancelledRide.ID,
		Status:        cancelledRide.Status,
		Timestamp:     *cancelledRide.CancelledAt,
		DriverID:      cancelledRide.DriverID,
		CorrelationID: cancelledRide.RideNumber,
	}

	if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
		s.logger.Error(wrap.ErrorCtx(ctx, err), "critical: failed to publish ride cancelled event", err)
	}

	s.logger.Info(ctx, "ride cancelled successfully", "ride_id", cancelledRide.ID)

	return cancelledRide, nil
}
