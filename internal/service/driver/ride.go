package drivergo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

var ErrDriverSearchTimeout = errors.New("driver search time exceeded")

func (s *Service) SearchDriver(ctx context.Context, req models.RideRequestedMessage) error {
	// Consume from driver_matching queue and unmarshal
	ctx = wrap.WithAction(ctx, "search_driver")

	// Search available drivers by matching algorithm
	drivers, err := s.repos.driver.SearchDrivers(
		ctx,
		req.RideType,
		models.Location{
			Latitude:  req.PickupLocation.Lat,
			Longitude: req.PickupLocation.Lng,
			Address:   req.PickupLocation.Address,
		})
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("failed to find available drivers to ride: %w", err))
	}

	timer := time.NewTimer(time.Hour * 12)
	for {
		select {
		case <-timer.C:
			s.l.Error(ctx, "driver search timeout", ErrDriverSearchTimeout)
			return ErrDriverSearchTimeout
		case <-ctx.Done():
			s.l.Info(ctx, "driver search loop close (ctx done)")
			return nil
		default:
			// Send Ride offers to 3-5 drivers
			for _, driver := range drivers {
				// Создаю горутину которая ждет флаг send

				// Кидаю предложение на поездку
				// SendRideOffer(ctx, rideOffer) error

				// Включаю прослушку для него
				// И потом меняю флаг send на true, разблокируя мьютекс

				// Жду 30 секунд для каждого водителя
				// Если принимает предложение выхожу из цикла и кидаю ответ юзеру

				// Если не принял, кидаю предложение другому

			}
		}
	}

	// Send driver response to client by publishing it to driver_topic
}

func (s *Service) MatchDriver(ctx context.Context) {
	// Consume confirmation data from queue

	// Check if driver is not busy

	// Send ride details(pickup location, navigation)

	// Change status to en_route

	// Track him in a real time
	// ListenUpdates(ctx, driverID)
}

func (s *Service) ListenUpdates(ctx context.Context, driverID uuid.UUID) {
	// Open message receiver
	// Dont forget about graceful shutdown
	// Also listen driver status, if he is available, return function

	// Handle messages by type:
	/*
		// Simple example:
			switch msg{
				case UpdateMsg:
					// Driver must send coordinates every 3-5 seconds(problem at client side)

					// Receive location data
					// Update coordinates in database

					// Publish to location_fanout
				case ArriveMsg:
					// Receive arrive msg
					// Publish status change
					return
			}
	*/

}
