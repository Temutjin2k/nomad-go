package ride

import (
	"context"
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
}

func NewRideService(repo RideRepo, logger logger.Logger, trm trm.TxManager) *RideService {
	return &RideService{
		repo: repo,
		logger: logger,
		trm: trm,
	}
}

func (s *RideService) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)  {
	ctx = wrap.WithAction(ctx, "create_ride")

	var createRide *models.Ride

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
				
		createRide, err = s.repo.Create(ctx, ride)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("could not create ride in repo: %w", err))
		}

		return nil
	})

	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	// TODO: добавить здесь публикацию в эксчейндж раббита

	return createRide, nil
}

func (s *RideService) Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error) {
	ctx = wrap.WithAction(ctx, "cancel_ride")

  var cancelledRide *models.Ride
    
    err := s.trm.Do(ctx, func(ctx context.Context) error {
        ride, err := s.repo.FindByID(ctx, rideID)
        if err != nil {
            return wrap.Error(ctx, types.ErrRideNotFound)
        }

        if ride.Status == "COMPLETED" || ride.Status == "CANCELLED" { // Насчет IN_PROGRESS момент спорный
            return wrap.Error(ctx, types.ErrRideCannotBeCancelled) 
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

    // TODO: здесь тоже опубликовать в раббит нужно
    
    return cancelledRide, nil
}
