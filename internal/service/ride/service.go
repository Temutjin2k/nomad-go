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

func NewRideService(repo RideRepo, logger logger.Logger) *RideService {
	return &RideService{
		repo: repo,
		logger: logger,
	}
}

func (s *RideService) Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)  {
	ctx = wrap.WithAction(ctx, "create_ride")

	var createRide *models.Ride

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		distance := calculateDistance(ride.Pickup, ride.Destination) 
		duration := calculateDuration(distance) 
    fare := calculateFare(ride.RideType, distance, duration) 

		var datePart = time.Now().Format("20060102")
		var now = time.Now()

		counter, err := (s.repo.CountByDate(ctx, now)) 
		if err != nil {
			return wrap.Error(ctx, err)
		}

		rideNumber := fmt.Sprintf("RIDE_%s_%03d", datePart, counter+1)

		ride.EstimatedDistanceKm = distance
        ride.EstimatedDurationMin = duration
        ride.EstimatedFare = fare
        ride.RideNumber = rideNumber
        ride.Status = "REQUESTED"
				
		createRide, err = s.repo.Create(ctx, ride)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("could not create ride in repo: %w", err))
		}

		return nil
	})

	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	// добавить здесь публикацию в эксчейндж раббита

	return createRide, nil
}

func (s *RideService) Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error) {
	ctx = wrap.WithAction(ctx, "cancel_ride")

    var cancelledRide *models.Ride
    
    err := s.trm.Do(ctx, func(ctx context.Context) error {
        ride, err := s.repo.FindByID(ctx, rideID)
        if err != nil {
            return types.ErrRideNotFound
        }

        if ride.Status == "COMPLETED" || ride.Status == "CANCELLED" || ride.Status == "IN_PROGRESS" { // Насчет IN_PROGRESS момент спорный
            return wrap.Error(ctx, types.ErrRideCannotBeCancelled) 
        }

        now := time.Now()
        ride.Status = "CANCELLED"
        ride.CancellationReason = &reason
        ride.CancelledAt = &now
        
        err = s.repo.Update(ctx, ride)
        if err != nil {
            return err
        }
        
        cancelledRide = ride
        return nil
    })
    
    if err != nil {
        return nil, err
    }

    // здесь тоже опубликовать в раббит нужно
    
    return cancelledRide, nil
}
